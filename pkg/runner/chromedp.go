package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/models"
)

// ChromeDP implements the Driver interface using chromedp
type ChromeDP struct {
	ctx    context.Context
	cancel context.CancelFunc
	opts   *Options
}

// NewChromeDP creates a new ChromeDP driver
func NewChromeDP(opts *Options) (*ChromeDP, error) {
	// 设置Chrome选项
	chromedpOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
	}

	// 根据配置设置无头模式
	if opts.Chrome.Headless {
		chromedpOpts = append(chromedpOpts, chromedp.Headless)
	}

	// 设置窗口大小
	chromedpOpts = append(chromedpOpts, chromedp.WindowSize(opts.Chrome.WindowX, opts.Chrome.WindowY))

	// 设置自定义User-Agent
	if opts.Chrome.UserAgent != "" {
		chromedpOpts = append(chromedpOpts, chromedp.UserAgent(opts.Chrome.UserAgent))
	}

	// 设置代理
	if opts.Chrome.Proxy != "" {
		chromedpOpts = append(chromedpOpts, chromedp.ProxyServer(opts.Chrome.Proxy))
	}

	// 设置Chrome路径
	if opts.Chrome.Path != "" {
		chromedpOpts = append(chromedpOpts, chromedp.ExecPath(opts.Chrome.Path))
	}

	// 忽略证书错误
	if opts.Chrome.IgnoreCertErrors {
		chromedpOpts = append(chromedpOpts, chromedp.Flag("ignore-certificate-errors", true))
	}

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
	result := &models.Result{
		URL:      target,
		ProbedAt: time.Now(),
	}

	// 创建网络事件监听器
	networkEvents := make(map[string]*models.NetworkLog)
	chromedp.ListenTarget(c.ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			networkEvents[e.RequestID.String()] = &models.NetworkLog{
				Type:   models.HTTP,
				URL:    e.Request.URL,
				Method: e.Request.Method,
			}
		case *network.EventResponseReceived:
			if nl, ok := networkEvents[e.RequestID.String()]; ok {
				nl.StatusCode = int(e.Response.Status)
				nl.ContentType = e.Response.MimeType
			}
		}
	})

	// 准备任务序列
	tasks := []chromedp.Action{
		network.Enable(),
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
			tasks = append(tasks, cookieParam)
		}
	}

	// 加载前执行JavaScript
	if c.opts.Scan.RunJSBefore && c.opts.Scan.JavaScript != "" {
		tasks = append(tasks, chromedp.Evaluate(c.opts.Scan.JavaScript, nil))
	}

	// 添加指纹伪装脚本
	if c.opts.Chrome.Platform != "" || c.opts.Chrome.Vendor != "" ||
		len(c.opts.Chrome.Plugins) > 0 || c.opts.Chrome.WebGLVendor != "" ||
		c.opts.Chrome.WebGLRenderer != "" || c.opts.Chrome.SpoofScreenSize {

		// 构建指纹伪装脚本
		fingerprintJS := "(() => {"

		// 修改navigator.platform
		if c.opts.Chrome.Platform != "" {
			fingerprintJS += fmt.Sprintf(`
				Object.defineProperty(navigator, 'platform', {
					get: function() { return '%s'; }
				});`, c.opts.Chrome.Platform)
		}

		// 修改vendor
		if c.opts.Chrome.Vendor != "" {
			fingerprintJS += fmt.Sprintf(`
				Object.defineProperty(navigator, 'vendor', {
					get: function() { return '%s'; }
				});`, c.opts.Chrome.Vendor)
		}

		// 修改plugins
		if len(c.opts.Chrome.Plugins) > 0 {
			pluginsJSON, _ := json.Marshal(c.opts.Chrome.Plugins)
			fingerprintJS += fmt.Sprintf(`
				Object.defineProperty(navigator, 'plugins', {
					get: function() { return %s; }
				});`, string(pluginsJSON))
		}

		// WebGL相关
		if c.opts.Chrome.WebGLVendor != "" || c.opts.Chrome.WebGLRenderer != "" {
			fingerprintJS += `
				const getParameter = WebGLRenderingContext.prototype.getParameter;
				WebGLRenderingContext.prototype.getParameter = function(parameter) {
			`

			if c.opts.Chrome.WebGLVendor != "" {
				fingerprintJS += fmt.Sprintf(`
					if (parameter === 37445) {
						return '%s';
					}`, c.opts.Chrome.WebGLVendor)
			}

			if c.opts.Chrome.WebGLRenderer != "" {
				fingerprintJS += fmt.Sprintf(`
					if (parameter === 37446) {
						return '%s';
					}`, c.opts.Chrome.WebGLRenderer)
			}

			fingerprintJS += `
					return getParameter.call(this, parameter);
				};`
		}

		// 屏幕尺寸
		if c.opts.Chrome.SpoofScreenSize && c.opts.Chrome.ScreenWidth > 0 && c.opts.Chrome.ScreenHeight > 0 {
			fingerprintJS += fmt.Sprintf(`
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
				});`, c.opts.Chrome.ScreenWidth, c.opts.Chrome.ScreenHeight,
				c.opts.Chrome.ScreenWidth, c.opts.Chrome.ScreenHeight)
		}

		// WebRTC
		if c.opts.Chrome.DisableWebRTC {
			fingerprintJS += `
				// 禁用WebRTC
				Object.defineProperty(window, 'RTCPeerConnection', {
					value: undefined
				});
				Object.defineProperty(window, 'webkitRTCPeerConnection', {
					value: undefined
				});`
		}

		fingerprintJS += "})();"

		// 执行指纹伪装脚本
		tasks = append(tasks, chromedp.Evaluate(fingerprintJS, nil))
	}

	// 页面导航
	tasks = append(tasks, chromedp.Navigate(target))

	// 添加延迟
	if c.opts.Chrome.Delay > 0 {
		tasks = append(tasks, chromedp.Sleep(time.Duration(c.opts.Chrome.Delay)*time.Second))
	}

	// 处理交互操作
	if len(c.opts.Scan.Actions) > 0 {
		for _, action := range c.opts.Scan.Actions {
			// 确定选择方式
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

			// 根据操作类型执行不同动作
			switch action.Type {
			case "click":
				tasks = append(tasks, chromedp.Click(sel, by))
			case "type":
				tasks = append(tasks, chromedp.SendKeys(sel, action.Value, by))
			case "scroll":
				scrollJS := fmt.Sprintf(`
					const el = document.querySelector("%s");
					if(el) { el.scrollBy(0, %s); }
				`, action.Selector, action.Value)
				tasks = append(tasks, chromedp.Evaluate(scrollJS, nil))
			case "wait":
				if action.WaitVisible {
					tasks = append(tasks, chromedp.WaitVisible(sel, by))
				} else {
					waitTime := 1000
					if action.WaitTime > 0 {
						waitTime = action.WaitTime
					}
					tasks = append(tasks, chromedp.Sleep(time.Duration(waitTime)*time.Millisecond))
				}
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
	}

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
			// 获取响应码
			var statusCode int
			for _, nl := range networkEvents {
				if nl.URL == target || strings.HasSuffix(target, nl.URL) {
					statusCode = nl.StatusCode
					break
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

	// 填充结果
	result.Title = title
	result.ResponseCode = responseCode
	result.HTML = htmlContent

	// 保存截图
	if !c.opts.Scan.ScreenshotSkipSave {
		filename := fmt.Sprintf("%s_%s.%s",
			strings.ReplaceAll(target, "/", "_"),
			time.Now().Format("20060102150405"),
			c.opts.Scan.ScreenshotFormat)
		filepath := filepath.Join(c.opts.Scan.ScreenshotPath, filename)

		err = ioutil.WriteFile(filepath, buf, 0644)
		if err != nil {
			log.Error("保存截图失败", "error", err)
		} else {
			result.Filename = filepath
			result.Screenshot = filepath
		}
	}

	// 保存Cookies
	if c.opts.Scan.SaveCookies && cookies != nil {
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

	return result, nil
}

// Close implements the Driver interface
func (c *ChromeDP) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}
