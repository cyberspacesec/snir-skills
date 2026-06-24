package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"

	"github.com/chromedp/chromedp"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/phash"
	"github.com/cyberspacesec/snir-skills/pkg/techdetect"
)

// ChromeDP implements the Driver interface using chromedp
type ChromeDP struct {
	ctx    context.Context
	cancel context.CancelFunc
	opts   *Options
}

// NewChromeDP creates a new ChromeDP driver
func NewChromeDP(opts *Options) (*ChromeDP, error) {
	// 使用共享的 allocOptions 构建逻辑
	chromedpOpts := buildAllocOptions(opts)

	// 创建Chrome上下文
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), chromedpOpts...)

	// 创建新的Chrome实例
	ctx, cancel = chromedp.NewContext(ctx)

	// 设置超时
	if opts.Chrome.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.Chrome.Timeout)*time.Second)
	}

	return &ChromeDP{
		ctx:    ctx,
		cancel: cancel,
		opts:   opts,
	}, nil
}

// Witness implements the Driver interface
func (c *ChromeDP) Witness(target string, opts *Options) (*models.Result, error) {
	if opts != nil {
		c.opts = opts
	}

	result := &models.Result{
		URL:      target,
		ProbedAt: time.Now(),
	}
	defer result.EnrichEndpoint()

	if c.opts.Scan.JavaScriptFile != "" && c.opts.Scan.JavaScript == "" {
		javascript, err := os.ReadFile(c.opts.Scan.JavaScriptFile)
		if err != nil {
			result.Failed = true
			result.FailedReason = err.Error()
			return result, err
		}
		c.opts.Scan.JavaScript = string(javascript)
	}

	// 创建网络事件监听器
	networkEvents := make(map[string]*models.NetworkLog)
	var responseHeaders []models.Header
	var responseTLS models.TLS
	var finalURL string
	var responseReason string
	var protocol string
	var contentLength int64
	var isPDF bool
	var consoleLogs []models.ConsoleLog

	chromedp.ListenTarget(c.ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			networkEvents[e.RequestID.String()] = &models.NetworkLog{
				Type:   models.HTTP,
				URL:    e.Request.URL,
				Method: e.Request.Method,
			}
			// 跟踪重定向：记录重定向后的最终URL
			if e.RedirectResponse != nil {
				finalURL = e.RedirectResponse.URL
			}
		case *network.EventResponseReceived:
			if nl, ok := networkEvents[e.RequestID.String()]; ok {
				nl.StatusCode = int(e.Response.Status)
				nl.ContentType = e.Response.MimeType
			}
			// 记录主请求的响应详情（精确匹配或后缀匹配目标URL）
			respURL := e.Response.URL
			if respURL == target || strings.HasSuffix(respURL, target) || strings.HasSuffix(target, respURL) {
				// 提取响应头
				if e.Response.Headers != nil {
					for name, val := range e.Response.Headers {
						responseHeaders = append(responseHeaders, models.Header{
							Name:  name,
							Value: fmt.Sprintf("%v", val),
						})
					}
				}
				// 提取TLS信息
				if e.Response.SecurityDetails != nil {
					sd := e.Response.SecurityDetails
					responseTLS = models.TLS{
						Version:     sd.Protocol,
						CipherSuite: sd.Cipher,
						Issuer:      sd.Issuer,
						Subject:     sd.SubjectName,
					}
					if sd.ValidFrom != nil {
						responseTLS.NotBefore = time.Time(*sd.ValidFrom)
					}
					if sd.ValidTo != nil {
						responseTLS.NotAfter = time.Time(*sd.ValidTo)
					}
					if len(sd.SanList) > 0 {
						responseTLS.SANs = strings.Join(sd.SanList, ", ")
					}
				}
				// 记录最终URL（重定向后的真实URL）
				finalURL = respURL
				// 记录响应原因短语
				responseReason = e.Response.StatusText
				// 记录协议版本
				if e.Response.Protocol != "" {
					protocol = e.Response.Protocol
				}
				// 记录内容长度（从响应头中提取）
				if cl, ok := e.Response.Headers["Content-Length"]; ok {
					if clStr, ok := cl.(string); ok {
						if clInt, err := strconv.ParseInt(clStr, 10, 64); err == nil {
							contentLength = clInt
						}
					}
				}
				// 检测PDF
				if strings.Contains(strings.ToLower(e.Response.MimeType), "pdf") {
					isPDF = true
				}
			}
		case *runtime.EventConsoleAPICalled:
			// 捕获控制台日志
			level := string(e.Type)
			var msgParts []string
			for _, arg := range e.Args {
				if len(arg.Value) > 0 {
					msgParts = append(msgParts, string(arg.Value))
				} else if arg.Description != "" {
					msgParts = append(msgParts, arg.Description)
				}
			}
			consoleLogs = append(consoleLogs, models.ConsoleLog{
				Level:   level,
				Message: strings.Join(msgParts, " "),
			})
		case *runtime.EventExceptionThrown:
			// 捕获未捕获的异常
			if e.ExceptionDetails != nil {
				msg := e.ExceptionDetails.Text
				if e.ExceptionDetails.Exception != nil && e.ExceptionDetails.Exception.Description != "" {
					msg = e.ExceptionDetails.Exception.Description
				}
				consoleLogs = append(consoleLogs, models.ConsoleLog{
					Level:   "error",
					Message: msg,
				})
			}
		default:
			// 忽略其他 CDP 事件
		}
	})

	// 准备任务序列
	tasks := []chromedp.Action{
		network.Enable(),
		runtime.Enable(),
		page.Enable(),
	}

	if c.opts.Chrome.UserAgent != "" {
		userAgentOverride := emulation.SetUserAgentOverride(c.opts.Chrome.UserAgent)
		if c.opts.Chrome.AcceptLanguage != "" {
			userAgentOverride = userAgentOverride.WithAcceptLanguage(c.opts.Chrome.AcceptLanguage)
		}
		if c.opts.Chrome.Platform != "" {
			userAgentOverride = userAgentOverride.WithPlatform(c.opts.Chrome.Platform)
		}
		tasks = append(tasks, userAgentOverride)
	} else if c.opts.Chrome.AcceptLanguage != "" || c.opts.Chrome.Platform != "" {
		// CDP requires a userAgent value for SetUserAgentOverride. Use the
		// current navigator value while still applying language/platform.
		var currentUserAgent string
		tasks = append(tasks,
			chromedp.Evaluate(`navigator.userAgent`, &currentUserAgent),
			chromedp.ActionFunc(func(ctx context.Context) error {
				if currentUserAgent == "" {
					return nil
				}
				params := emulation.SetUserAgentOverride(currentUserAgent)
				if c.opts.Chrome.AcceptLanguage != "" {
					params = params.WithAcceptLanguage(c.opts.Chrome.AcceptLanguage)
				}
				if c.opts.Chrome.Platform != "" {
					params = params.WithPlatform(c.opts.Chrome.Platform)
				}
				return params.Do(ctx)
			}),
		)
	}

	tasks = append(tasks, buildDeviceEmulationActions(c.opts)...)

	if c.opts.Chrome.AcceptLanguage != "" || len(c.opts.Chrome.CustomHeaders) > 0 {
		headers := network.Headers{}
		if c.opts.Chrome.AcceptLanguage != "" {
			headers["Accept-Language"] = c.opts.Chrome.AcceptLanguage
		}
		for name, value := range c.opts.Chrome.CustomHeaders {
			headers[name] = value
		}
		tasks = append(tasks, network.SetExtraHTTPHeaders(headers))
	}

	// 设置Cookie
	if len(c.opts.Scan.Cookies) > 0 {
		for _, cookie := range c.opts.Scan.Cookies {
			cookieParam := network.SetCookie(cookie.Name, cookie.Value)
			if cookie.Domain != "" {
				cookieParam = cookieParam.WithDomain(cookie.Domain)
			}
			if cookie.Path != "" {
				cookieParam = cookieParam.WithPath(cookie.Path)
			}
			cookieParam = cookieParam.WithSecure(cookie.Secure)
			cookieParam = cookieParam.WithHTTPOnly(cookie.HttpOnly)
			if cookie.Expires > 0 {
				ts := cdp.TimeSinceEpoch(time.Unix(cookie.Expires, 0))
				cookieParam = cookieParam.WithExpires(&ts)
			}
			if cookie.SameSite != "" {
				cookieParam = cookieParam.WithSameSite(parseSameSite(cookie.SameSite))
			}
			tasks = append(tasks, cookieParam)
		}
	}

	// 加载前执行JavaScript
	if c.opts.Scan.RunJSBefore && c.opts.Scan.JavaScript != "" {
		tasks = append(tasks, addScriptToEvaluateOnNewDocument(c.opts.Scan.JavaScript))
	}

	// 添加指纹伪装脚本
	if fingerprintJS := buildFingerprintScript(c.opts); fingerprintJS != "" {
		tasks = append(tasks, addScriptToEvaluateOnNewDocument(fingerprintJS))
	}

	// 页面导航
	tasks = append(tasks, chromedp.Navigate(target))

	// 添加延迟
	if c.opts.Chrome.Delay > 0 {
		tasks = append(tasks, chromedp.Sleep(time.Duration(c.opts.Chrome.Delay)*time.Second))
	}

	tasks = append(tasks, buildInteractionActions(c.opts.Scan.Actions)...)

	// 表单填充
	if len(c.opts.Scan.Form.Fields) > 0 {
		// 填充每个字段
		for _, field := range c.opts.Scan.Form.Fields {
			var sel interface{}
			var by chromedp.QueryOption

			if field.Selector != "" {
				sel = field.Selector
				by = chromedp.ByQuery
			} else if field.XPath != "" {
				sel = field.XPath
				by = chromedp.BySearch
			} else {
				continue
			}

			// 根据字段类型处理
			switch field.Type {
			case "checkbox", "radio":
				tasks = append(tasks, chromedp.Click(sel, by))
			case "select":
				tasks = append(tasks, chromedp.SendKeys(sel, field.Value, by))
			default: // input
				// 先清空字段内容
				tasks = append(tasks, chromedp.Clear(sel, by))
				// 再填充新值
				tasks = append(tasks, chromedp.SendKeys(sel, field.Value, by))
			}
		}

		// 提交表单
		if c.opts.Scan.Form.SubmitSelector != "" || c.opts.Scan.Form.SubmitXPath != "" {
			var submitSel interface{}
			var by chromedp.QueryOption

			if c.opts.Scan.Form.SubmitSelector != "" {
				submitSel = c.opts.Scan.Form.SubmitSelector
				by = chromedp.ByQuery
			} else {
				submitSel = c.opts.Scan.Form.SubmitXPath
				by = chromedp.BySearch
			}

			// 点击提交按钮
			tasks = append(tasks, chromedp.Click(submitSel, by))

			// 提交后等待
			if c.opts.Scan.Form.WaitAfterSubmit > 0 {
				tasks = append(tasks, chromedp.Sleep(time.Duration(c.opts.Scan.Form.WaitAfterSubmit)*time.Millisecond))
			} else {
				// 默认等待1秒
				tasks = append(tasks, chromedp.Sleep(1*time.Second))
			}
		}
	}

	// 加载后执行JavaScript
	if c.opts.Scan.RunJSAfter && c.opts.Scan.JavaScript != "" {
		tasks = append(tasks, chromedp.Evaluate(c.opts.Scan.JavaScript, nil))
	}

	// 获取页面信息
	var buf []byte
	var htmlContent string
	var title string
	var responseCode int
	var cookies []*network.Cookie

	tasks = append(tasks,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 获取响应码 — 优先精确匹配，其次模糊匹配，最后取第一个非零状态码
			var statusCode int
			for _, nl := range networkEvents {
				if nl.URL == target {
					statusCode = nl.StatusCode
					break
				}
			}
			// 精确匹配失败，尝试后缀匹配
			if statusCode == 0 {
				for _, nl := range networkEvents {
					if strings.HasSuffix(nl.URL, target) || strings.HasSuffix(target, nl.URL) {
						statusCode = nl.StatusCode
						break
					}
				}
			}
			// 仍然为0，取第一个有非零状态码的响应
			if statusCode == 0 {
				for _, nl := range networkEvents {
					if nl.StatusCode > 0 {
						statusCode = nl.StatusCode
						break
					}
				}
			}
			responseCode = statusCode
			return nil
		}),
		chromedp.Title(&title),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 获取HTML内容
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			htmlContent, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 获取Cookies
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}),
	)

	// 根据不同的选择方式截图
	if c.opts.Scan.Selector != "" {
		// 使用CSS选择器截图
		tasks = append(tasks, chromedp.Screenshot(c.opts.Scan.Selector, &buf, chromedp.ByQuery))
	} else if c.opts.Scan.XPath != "" {
		// 使用XPath截图
		tasks = append(tasks, chromedp.Screenshot(c.opts.Scan.XPath, &buf, chromedp.BySearch))
	} else if c.opts.Scan.CaptureFullPage {
		// 捕获完整页面（包括滚动部分）
		tasks = append(tasks, chromedp.FullScreenshot(&buf, 100))
	} else {
		// 默认捕获可视区域
		tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
	}

	// 执行任务
	err := chromedp.Run(c.ctx, tasks...)
	if err != nil {
		result.Failed = true
		result.FailedReason = err.Error()
		return result, err
	}
	buf, err = encodeScreenshot(buf, c.opts.Scan.ScreenshotFormat, c.opts.Scan.ScreenshotQuality)
	if err != nil {
		result.Failed = true
		result.FailedReason = err.Error()
		return result, err
	}

	// 填充结果
	result.Title = title
	result.ResponseCode = responseCode
	result.HTML = htmlContent

	// 填充从CDP事件收集的额外信息
	result.FinalURL = finalURL
	result.ResponseReason = responseReason
	result.Protocol = protocol
	result.ContentLength = contentLength
	result.IsPDF = isPDF
	if c.opts.Scan.ReturnScreenshotBytes {
		result.ScreenshotBytes = append([]byte(nil), buf...)
	}

	// 保存截图
	if !c.opts.Scan.ScreenshotSkipSave {
		filename := fmt.Sprintf("%s_%s.%s",
			strings.ReplaceAll(target, "/", "_"),
			time.Now().Format("20060102150405"),
			c.opts.Scan.ScreenshotFormat)
		screenshotFilepath := filepath.Join(c.opts.Scan.ScreenshotPath, filename)

		err = os.WriteFile(screenshotFilepath, buf, 0644)
		if err != nil {
			log.Error("保存截图失败", "error", err)
		} else {
			// 确保返回绝对路径
			absPath, _ := filepath.Abs(screenshotFilepath)
			result.Filename = absPath
			result.Screenshot = absPath
		}
	}

	// 计算感知哈希（用于截图去重和相似度检测）
	if len(buf) > 0 {
		if hashResult, err := phash.ComputeHash(buf); err == nil {
			result.PerceptionHash = hashResult.Hash
		} else {
			log.Debug("计算感知哈希失败", "error", err)
		}
	}

	// 保存响应头
	if c.opts.Scan.SaveHeaders && len(responseHeaders) > 0 {
		result.Headers = responseHeaders
	}

	// 保存TLS信息
	if responseTLS.Version != "" || responseTLS.Issuer != "" {
		result.TLS = responseTLS
	}

	// 保存控制台日志
	if c.opts.Scan.SaveConsole && len(consoleLogs) > 0 {
		result.Console = consoleLogs
	}

	// 保存Cookies
	if (c.opts.Scan.SaveCookies || c.opts.Scan.CookieWriteBack) && cookies != nil {
		for _, cookie := range cookies {
			result.Cookies = append(result.Cookies, models.Cookie{
				Name:   cookie.Name,
				Value:  cookie.Value,
				Domain: cookie.Domain,
				Path:   cookie.Path,
			})
		}
	}

	// 保存网络日志
	if c.opts.Scan.SaveNetwork {
		for _, nl := range networkEvents {
			result.Network = append(result.Network, *nl)
		}
	}

	// 技术指纹识别
	detector := techdetect.NewDetector()
	techs := detector.DetectFromResult(result)
	if len(techs) > 0 {
		result.Technologies = techdetect.ToModelsTechnologies(techs)
	}

	return result, nil
}

func buildInteractionActions(actions []InteractionAction) []chromedp.Action {
	if len(actions) == 0 {
		return nil
	}

	tasks := make([]chromedp.Action, 0, len(actions))
	for _, action := range actions {
		if action.Type == "wait" && !action.WaitVisible {
			waitTime := 1000
			if action.WaitTime > 0 {
				waitTime = action.WaitTime
			}
			tasks = append(tasks, chromedp.Sleep(time.Duration(waitTime)*time.Millisecond))
			continue
		}

		var sel interface{}
		var by chromedp.QueryOption

		if action.Selector != "" {
			sel = action.Selector
			by = chromedp.ByQuery
		} else if action.XPath != "" {
			sel = action.XPath
			by = chromedp.BySearch
		} else {
			continue
		}

		switch action.Type {
		case "click":
			tasks = append(tasks, chromedp.Click(sel, by))
		case "type":
			tasks = append(tasks, chromedp.SendKeys(sel, action.Value, by))
		case "scroll":
			if action.Selector == "" {
				continue
			}
			scrollJS := fmt.Sprintf(`
				const el = document.querySelector("%s");
				if(el) { el.scrollBy(0, %s); }
			`, action.Selector, action.Value)
			tasks = append(tasks, chromedp.Evaluate(scrollJS, nil))
		case "wait":
			tasks = append(tasks, chromedp.WaitVisible(sel, by))
		case "hover":
			if action.Selector != "" {
				hoverJS := fmt.Sprintf(`
					(function() {
						const element = document.querySelector("%s");
						if (element) {
							const mouseoverEvent = new MouseEvent('mouseover', {
								bubbles: true,
								cancelable: true,
								view: window
							});
							element.dispatchEvent(mouseoverEvent);
							return true;
						}
						return false;
					})()`, action.Selector)
				tasks = append(tasks, chromedp.Evaluate(hoverJS, nil))
			} else if action.XPath != "" {
				hoverJS := fmt.Sprintf(`
					(function() {
						const result = document.evaluate("%s", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
						const element = result.singleNodeValue;
						if (element) {
							const mouseoverEvent = new MouseEvent('mouseover', {
								bubbles: true,
								cancelable: true,
								view: window
							});
							element.dispatchEvent(mouseoverEvent);
							return true;
						}
						return false;
					})()`, action.XPath)
				tasks = append(tasks, chromedp.Evaluate(hoverJS, nil))
			}
		}
	}

	return tasks
}

// Close implements the Driver interface
func (c *ChromeDP) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

// parseSameSite 将字符串转为 network.CookieSameSite
func parseSameSite(s string) network.CookieSameSite {
	switch strings.ToLower(s) {
	case "strict":
		return network.CookieSameSiteStrict
	case "lax":
		return network.CookieSameSiteLax
	case "none":
		return network.CookieSameSiteNone
	default:
		return network.CookieSameSiteLax
	}
}

func buildFingerprintScript(opts *Options) string {
	if opts == nil {
		return ""
	}

	if opts.Chrome.Platform == "" && opts.Chrome.Vendor == "" &&
		len(opts.Chrome.Plugins) == 0 && opts.Chrome.WebGLVendor == "" &&
		opts.Chrome.WebGLRenderer == "" && !opts.Chrome.SpoofScreenSize &&
		!opts.Chrome.DisableWebRTC {
		return ""
	}

	var b strings.Builder
	b.WriteString("(() => {")

	if opts.Chrome.Platform != "" {
		b.WriteString(`
			Object.defineProperty(navigator, 'platform', {
				get: function() { return `)
		b.WriteString(jsStringLiteral(opts.Chrome.Platform))
		b.WriteString(`; }
			});`)
	}

	if opts.Chrome.Vendor != "" {
		b.WriteString(`
			Object.defineProperty(navigator, 'vendor', {
				get: function() { return `)
		b.WriteString(jsStringLiteral(opts.Chrome.Vendor))
		b.WriteString(`; }
			});`)
	}

	if len(opts.Chrome.Plugins) > 0 {
		pluginsJSON, _ := json.Marshal(opts.Chrome.Plugins)
		b.WriteString(`
			Object.defineProperty(navigator, 'plugins', {
				get: function() { return `)
		b.Write(pluginsJSON)
		b.WriteString(`; }
			});`)
	}

	if opts.Chrome.WebGLVendor != "" || opts.Chrome.WebGLRenderer != "" {
		b.WriteString(`
			const getParameter = WebGLRenderingContext.prototype.getParameter;
			WebGLRenderingContext.prototype.getParameter = function(parameter) {`)

		if opts.Chrome.WebGLVendor != "" {
			b.WriteString(`
				if (parameter === 37445) {
					return `)
			b.WriteString(jsStringLiteral(opts.Chrome.WebGLVendor))
			b.WriteString(`;
				}`)
		}

		if opts.Chrome.WebGLRenderer != "" {
			b.WriteString(`
				if (parameter === 37446) {
					return `)
			b.WriteString(jsStringLiteral(opts.Chrome.WebGLRenderer))
			b.WriteString(`;
				}`)
		}

		b.WriteString(`
				return getParameter.call(this, parameter);
			};`)
	}

	if opts.Chrome.SpoofScreenSize && opts.Chrome.ScreenWidth > 0 && opts.Chrome.ScreenHeight > 0 {
		b.WriteString(fmt.Sprintf(`
			Object.defineProperty(window, 'screen', {
				get: function() {
					return {
						width: %d,
						height: %d,
						availWidth: %d,
						availHeight: %d,
						colorDepth: 24,
						pixelDepth: 24
					};
				}
			});`, opts.Chrome.ScreenWidth, opts.Chrome.ScreenHeight,
			opts.Chrome.ScreenWidth, opts.Chrome.ScreenHeight))
	}

	if opts.Chrome.DisableWebRTC {
		b.WriteString(`
			Object.defineProperty(window, 'RTCPeerConnection', {
				value: undefined
			});
			Object.defineProperty(window, 'webkitRTCPeerConnection', {
				value: undefined
			});`)
	}

	b.WriteString("})();")
	return b.String()
}

func buildDeviceEmulationActions(opts *Options) []chromedp.Action {
	if opts == nil {
		return nil
	}

	width := opts.Chrome.WindowX
	height := opts.Chrome.WindowY
	if opts.Chrome.ScreenWidth > 0 {
		width = opts.Chrome.ScreenWidth
	}
	if opts.Chrome.ScreenHeight > 0 {
		height = opts.Chrome.ScreenHeight
	}
	if width <= 0 || height <= 0 {
		return nil
	}

	scaleFactor := opts.Chrome.DeviceScaleFactor
	if scaleFactor == 0 && (opts.Chrome.IsMobile || opts.Chrome.HasTouch || opts.Chrome.SpoofScreenSize) {
		scaleFactor = 1
	}
	if scaleFactor == 0 && !opts.Chrome.IsMobile && !opts.Chrome.HasTouch && !opts.Chrome.SpoofScreenSize {
		return nil
	}

	metrics := emulation.SetDeviceMetricsOverride(
		int64(width),
		int64(height),
		scaleFactor,
		opts.Chrome.IsMobile,
	).WithScreenWidth(int64(width)).WithScreenHeight(int64(height))

	actions := []chromedp.Action{metrics}
	if opts.Chrome.HasTouch {
		actions = append(actions, emulation.SetTouchEmulationEnabled(true).WithMaxTouchPoints(1))
	}
	return actions
}

func jsStringLiteral(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

func addScriptToEvaluateOnNewDocument(script string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
		return err
	})
}

func encodeScreenshot(buf []byte, format string, quality int) ([]byte, error) {
	format = strings.ToLower(format)
	if format == "" {
		format = "png"
	}

	if format != "jpeg" && format != "jpg" && format != "png" {
		return nil, fmt.Errorf("unsupported screenshot format: %s", format)
	}

	if format == "png" && bytes.HasPrefix(buf, []byte{0x89, 'P', 'N', 'G'}) {
		return buf, nil
	}
	if (format == "jpeg" || format == "jpg") && bytes.HasPrefix(buf, []byte{0xff, 0xd8}) {
		return buf, nil
	}

	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}

	var out bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&out, img); err != nil {
			return nil, fmt.Errorf("encode png screenshot: %w", err)
		}
	default:
		if quality <= 0 || quality > 100 {
			quality = 90
		}
		opaque := image.NewRGBA(img.Bounds())
		draw.Draw(opaque, opaque.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
		draw.Draw(opaque, opaque.Bounds(), img, img.Bounds().Min, draw.Over)
		if err := jpeg.Encode(&out, opaque, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode jpeg screenshot: %w", err)
		}
	}

	return out.Bytes(), nil
}
