// Package sdk 提供 go-snir 截图能力的 Go API，供其他项目直接 import 调用
//
// 使用示例:
//
//	client, _ := sdk.NewClient(sdk.DefaultClientOptions())
//	defer client.Close()
//
//	// 基本截图
//	result, _ := client.Screenshot("https://example.com", nil)
//	fmt.Println(result.Title, result.Filename)
//
//	// 组合复杂场景
//	result, _ = client.Capture("https://example.com",
//	    sdk.WithFullPage(),
//	    sdk.WithEvidence(),
//	    sdk.WithDevice("iphone-15"),
//	)
//
//	// 获取截图字节数据（不写磁盘）
//	imgBytes, result, _ := client.ScreenshotBytes("https://example.com", nil)
//
//	// 批量截图
//	results := client.BatchScreenshot([]string{"https://a.com", "https://b.com"}, nil)
//
//	// 流式批量截图（每完成一个立即返回）
//	ch := client.BatchScreenshotStreaming(ctx, urls, nil)
//	for result := range ch {
//	    fmt.Println(result.URL, result.Result.Title)
//	}
//
//	// 带取消的截图
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	result, _ := client.ScreenshotWithContext(ctx, "https://example.com", nil)
package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// Client 是 go-snir 截图 SDK 的主入口
// 其他 Go 项目通过 import 此包来复用截图能力
// 内部持有 DriverPool，多个调用方共享同一个 Chrome 浏览器进程
type Client struct {
	pool      driverPool
	opts      ClientOptions
	cookieJar *runner.CookieJar // Cookie 持久化存储
}

type driverPool interface {
	ScreenshotWithContext(context.Context, string, *runner.Options) (*models.Result, error)
	Stats() runner.PoolStats
	SetIdleTimeout(time.Duration)
	On(runner.PoolEventHandler)
	ActiveCount() int
	Close()
}

var (
	newDriverPool = func(opts *runner.Options, maxConcurrent int) (driverPool, error) {
		return runner.NewDriverPool(opts, maxConcurrent)
	}
	newCookieJar = runner.NewCookieJar
)

// NewClient 创建一个新的截图客户端
// 内部初始化 Chrome 浏览器进程池，多个截图请求复用同一浏览器进程
func NewClient(opts ClientOptions) (*Client, error) {
	runnerOpts := toRunnerOptions(opts)
	pool, err := newDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("初始化截图客户端失败: %v", err)
	}

	client := &Client{
		pool: pool,
		opts: opts,
	}

	// 加载 Cookie 持久化存储
	if opts.CookieFile != "" {
		jar, err := newCookieJar(opts.CookieFile)
		if err != nil {
			log.Warn("加载 Cookie 文件失败", "file", opts.CookieFile, "error", err)
		} else {
			client.cookieJar = jar
			log.Info("Cookie 持久化存储已加载", "file", opts.CookieFile)
		}
	}

	mode := "本地"
	if opts.WSSURL != "" {
		mode = "远程"
	}
	log.Info("截图SDK客户端已创建", "mode", mode, "max_concurrent", opts.MaxConcurrent)
	return client, nil
}

// NewRemoteClient 创建一个连接到远程 Chrome 的截图客户端
// wsURL: 远程 Chrome 的 WebSocket URL（如 ws://hostname:9222/devtools/browser/xxxx）
// maxConcurrent: 最大并发截图数
// 其他选项使用默认值
func NewRemoteClient(wsURL string, maxConcurrent int) (*Client, error) {
	opts := DefaultClientOptions()
	opts.WSSURL = wsURL
	if maxConcurrent > 0 {
		opts.MaxConcurrent = maxConcurrent
	}

	runnerOpts := toRunnerOptions(opts)
	pool, err := newDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("连接远程浏览器失败: %v", err)
	}

	log.Info("远程截图SDK客户端已创建", "ws_url", wsURL, "max_concurrent", opts.MaxConcurrent)
	return &Client{
		pool: pool,
		opts: opts,
	}, nil
}

// ---------------------------------------------------------------------------
// 截图方法
// ---------------------------------------------------------------------------

// Capture 使用函数式选项执行截图。
//
// 示例:
//
//	result, err := client.Capture(
//	    "https://example.com",
//	    sdk.WithFullPage(),
//	    sdk.WithEvidence(),
//	    sdk.WithDevice("iphone-15"),
//	)
func (c *Client) Capture(url string, options ...ScreenshotOption) (*models.Result, error) {
	return c.CaptureWithContext(context.Background(), url, options...)
}

// CaptureWithContext 使用函数式选项执行可取消的截图。
func (c *Client) CaptureWithContext(ctx context.Context, url string, options ...ScreenshotOption) (*models.Result, error) {
	return c.ScreenshotWithContext(ctx, url, NewScreenshotOptions(options...))
}

// Screenshot 对指定 URL 执行截图
// url: 目标网页 URL
// screenshotOpts: 单次截图的可选配置，可覆盖客户端默认配置，传 nil 使用默认配置
// 返回截图结果，包含页面标题、截图文件路径、状态码等信息
func (c *Client) Screenshot(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	return c.ScreenshotWithContext(context.Background(), url, screenshotOpts)
}

// ScreenshotWithContext 支持取消的截图
// ctx 可用于取消长时间运行的截图任务
func (c *Client) ScreenshotWithContext(ctx context.Context, url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	runnerOpts := c.runnerOptionsForScreenshot(url, screenshotOpts)
	if result, err := rejectBlacklistedTarget(url, &runnerOpts); err != nil {
		return result, err
	}

	result, err := c.pool.ScreenshotWithContext(ctx, url, &runnerOpts)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	c.handleResultCookies(url, result, &runnerOpts)

	if result.Failed {
		return result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}

	return result, nil
}

// ScreenshotBytes 对指定 URL 执行截图，返回截图的原始字节数据
// 适合在内存中直接使用截图数据（如上传到 S3、写入 HTTP response 等）
// 返回 PNG/JPEG 字节数据、截图元信息、错误
func (c *Client) ScreenshotBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	return c.ScreenshotBytesWithContext(context.Background(), url, screenshotOpts)
}

// CaptureBytes 使用函数式选项执行截图并返回图片字节。
func (c *Client) CaptureBytes(url string, options ...ScreenshotOption) ([]byte, *models.Result, error) {
	return c.CaptureBytesWithContext(context.Background(), url, options...)
}

// CaptureBytesWithContext 使用函数式选项执行可取消的截图并返回图片字节。
func (c *Client) CaptureBytesWithContext(ctx context.Context, url string, options ...ScreenshotOption) ([]byte, *models.Result, error) {
	return c.ScreenshotBytesWithContext(ctx, url, NewScreenshotOptions(options...))
}

// ScreenshotBytesWithContext 支持取消的截图字节数据获取
func (c *Client) ScreenshotBytesWithContext(ctx context.Context, url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	runnerOpts := c.runnerOptionsForScreenshot(url, screenshotOpts)
	runnerOpts.Scan.ReturnScreenshotBytes = true
	runnerOpts.Scan.ScreenshotSkipSave = true
	if result, err := rejectBlacklistedTarget(url, &runnerOpts); err != nil {
		return nil, result, err
	}

	result, err := c.pool.ScreenshotWithContext(ctx, url, &runnerOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("截图失败: %v", err)
	}

	c.handleResultCookies(url, result, &runnerOpts)

	if result.Failed {
		return nil, result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}

	data, err := screenshotBytesFromResult(result)
	if err != nil {
		return nil, result, err
	}

	return data, result, nil
}

func screenshotBytesFromResult(result *models.Result) ([]byte, error) {
	if len(result.ScreenshotBytes) > 0 {
		return result.ScreenshotBytes, nil
	}

	if result.Screenshot == "" {
		return nil, fmt.Errorf("截图文件路径为空")
	}

	data, err := os.ReadFile(result.Screenshot)
	if err != nil {
		return nil, fmt.Errorf("读取截图文件失败: %v", err)
	}
	return data, nil
}

// ScreenshotHTML 截图并同时获取页面 HTML 源码
// 等价于设置 SaveHTML=true 后截图，便捷方法
func (c *Client) ScreenshotHTML(url string, screenshotOpts *ScreenshotOptions) (string, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveHTML = true

	result, err := c.Screenshot(url, opts)
	if err != nil {
		return "", nil, err
	}
	return result.HTML, result, nil
}

// ScreenshotHeaders 截图并收集 HTTP 头。
func (c *Client) ScreenshotHeaders(url string, screenshotOpts *ScreenshotOptions) ([]models.Header, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveHeaders = true

	result, err := c.Screenshot(url, opts)
	if err != nil {
		return nil, nil, err
	}
	return result.Headers, result, nil
}

// ScreenshotHeadersBytes 截图、收集 HTTP 头，并返回图片字节。
func (c *Client) ScreenshotHeadersBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveHeaders = true
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotCookies 截图并收集浏览器 Cookie。
func (c *Client) ScreenshotCookies(url string, screenshotOpts *ScreenshotOptions) ([]models.Cookie, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveCookies = true

	result, err := c.Screenshot(url, opts)
	if err != nil {
		return nil, nil, err
	}
	return result.Cookies, result, nil
}

// ScreenshotCookiesBytes 截图、收集浏览器 Cookie，并返回图片字节。
func (c *Client) ScreenshotCookiesBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveCookies = true
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotConsole 截图并收集浏览器控制台日志。
func (c *Client) ScreenshotConsole(url string, screenshotOpts *ScreenshotOptions) ([]models.ConsoleLog, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveConsole = true

	result, err := c.Screenshot(url, opts)
	if err != nil {
		return nil, nil, err
	}
	return result.Console, result, nil
}

// ScreenshotConsoleBytes 截图、收集浏览器控制台日志，并返回图片字节。
func (c *Client) ScreenshotConsoleBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveConsole = true
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotNetwork 截图并收集网络请求日志。
func (c *Client) ScreenshotNetwork(url string, screenshotOpts *ScreenshotOptions) ([]models.NetworkLog, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveNetwork = true

	result, err := c.Screenshot(url, opts)
	if err != nil {
		return nil, nil, err
	}
	return result.Network, result, nil
}

// ScreenshotNetworkBytes 截图、收集网络请求日志，并返回图片字节。
func (c *Client) ScreenshotNetworkBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveNetwork = true
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithFormat 使用指定图片格式和质量截图。
func (c *Client) ScreenshotWithFormat(url string, format string, quality int, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.ScreenshotFormat = format
	opts.ScreenshotQuality = quality
	return c.Screenshot(url, opts)
}

// ScreenshotWithFormatBytes 使用指定图片格式和质量截图，并返回图片字节。
func (c *Client) ScreenshotWithFormatBytes(url string, format string, quality int, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.ScreenshotFormat = format
	opts.ScreenshotQuality = quality
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotToPath 使用指定输出目录或文件路径保存截图。
func (c *Client) ScreenshotToPath(url string, path string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.ScreenshotPath = path
	return c.Screenshot(url, opts)
}

// ScreenshotWithDelay 等待指定时间后截图。
func (c *Client) ScreenshotWithDelay(url string, delay time.Duration, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Delay = delay
	return c.Screenshot(url, opts)
}

// ScreenshotWithDelayBytes 等待指定时间后截图，并返回图片字节。
func (c *Client) ScreenshotWithDelayBytes(url string, delay time.Duration, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Delay = delay
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithTimeout 使用指定页面加载超时截图。
func (c *Client) ScreenshotWithTimeout(url string, timeout time.Duration, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Timeout = timeout
	return c.Screenshot(url, opts)
}

// ScreenshotWithTimeoutBytes 使用指定页面加载超时截图，并返回图片字节。
func (c *Client) ScreenshotWithTimeoutBytes(url string, timeout time.Duration, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Timeout = timeout
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithProxy 使用指定代理截图。
func (c *Client) ScreenshotWithProxy(url string, proxy string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxy(proxy)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithProxyBytes 使用指定代理截图，并返回图片字节。
func (c *Client) ScreenshotWithProxyBytes(url string, proxy string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxy(proxy)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithProxyList 使用代理列表和轮换策略截图。
func (c *Client) ScreenshotWithProxyList(url string, strategy runner.ProxyStrategy, proxies []string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxyList(strategy, proxies...)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithProxyListBytes 使用代理列表和轮换策略截图，并返回图片字节。
func (c *Client) ScreenshotWithProxyListBytes(url string, strategy runner.ProxyStrategy, proxies []string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxyList(strategy, proxies...)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithProxyFile 使用代理文件和轮换策略截图。
func (c *Client) ScreenshotWithProxyFile(url string, path string, strategy runner.ProxyStrategy, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxyFile(path, strategy)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithProxyFileBytes 使用代理文件和轮换策略截图，并返回图片字节。
func (c *Client) ScreenshotWithProxyFileBytes(url string, path string, strategy runner.ProxyStrategy, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxyFile(path, strategy)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithProxyURL 使用动态代理 API 和轮换策略截图。
func (c *Client) ScreenshotWithProxyURL(url string, proxyURL string, strategy runner.ProxyStrategy, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxyURL(proxyURL, strategy)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithProxyURLBytes 使用动态代理 API 和轮换策略截图，并返回图片字节。
func (c *Client) ScreenshotWithProxyURLBytes(url string, proxyURL string, strategy runner.ProxyStrategy, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithProxyURL(proxyURL, strategy)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithCustomHeaders 使用自定义请求头截图。
func (c *Client) ScreenshotWithCustomHeaders(url string, headers map[string]string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCustomHeaders(headers)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithCustomHeadersBytes 使用自定义请求头截图，并返回图片字节。
func (c *Client) ScreenshotWithCustomHeadersBytes(url string, headers map[string]string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCustomHeaders(headers)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithUserAgent 使用指定 User-Agent 截图。
func (c *Client) ScreenshotWithUserAgent(url string, userAgent string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithUserAgent(userAgent)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithUserAgentBytes 使用指定 User-Agent 截图，并返回图片字节。
func (c *Client) ScreenshotWithUserAgentBytes(url string, userAgent string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithUserAgent(userAgent)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithAcceptLanguage 使用指定 Accept-Language 截图。
func (c *Client) ScreenshotWithAcceptLanguage(url string, language string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithAcceptLanguage(language)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithAcceptLanguageBytes 使用指定 Accept-Language 截图，并返回图片字节。
func (c *Client) ScreenshotWithAcceptLanguageBytes(url string, language string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithAcceptLanguage(language)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithFingerprint 使用浏览器指纹覆盖截图。
func (c *Client) ScreenshotWithFingerprint(url string, platform, vendor, webGLVendor, webGLRenderer string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithFingerprint(platform, vendor, webGLVendor, webGLRenderer)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithFingerprintBytes 使用浏览器指纹覆盖截图，并返回图片字节。
func (c *Client) ScreenshotWithFingerprintBytes(url string, platform, vendor, webGLVendor, webGLRenderer string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithFingerprint(platform, vendor, webGLVendor, webGLRenderer)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithDeviceEmulation 使用自定义设备参数截图。
func (c *Client) ScreenshotWithDeviceEmulation(url string, width, height int, scaleFactor float64, isMobile, hasTouch bool, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithDeviceEmulation(width, height, scaleFactor, isMobile, hasTouch)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithDeviceEmulationBytes 使用自定义设备参数截图，并返回图片字节。
func (c *Client) ScreenshotWithDeviceEmulationBytes(url string, width, height int, scaleFactor float64, isMobile, hasTouch bool, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithDeviceEmulation(width, height, scaleFactor, isMobile, hasTouch)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithMobileEmulation 使用移动端和触摸仿真截图。
func (c *Client) ScreenshotWithMobileEmulation(url string, scaleFactor float64, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithMobileEmulation(scaleFactor)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithMobileEmulationBytes 使用移动端和触摸仿真截图，并返回图片字节。
func (c *Client) ScreenshotWithMobileEmulationBytes(url string, scaleFactor float64, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithMobileEmulation(scaleFactor)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithTouchEmulation 使用指定触摸仿真状态截图。
func (c *Client) ScreenshotWithTouchEmulation(url string, enabled bool, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithTouchEmulation(enabled)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithTouchEmulationBytes 使用指定触摸仿真状态截图，并返回图片字节。
func (c *Client) ScreenshotWithTouchEmulationBytes(url string, enabled bool, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithTouchEmulation(enabled)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithIgnoreCertErrors 忽略证书错误后截图。
func (c *Client) ScreenshotWithIgnoreCertErrors(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithIgnoreCertErrors()(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithIgnoreCertErrorsBytes 忽略证书错误后截图，并返回图片字节。
func (c *Client) ScreenshotWithIgnoreCertErrorsBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithIgnoreCertErrors()(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithPlugins 使用指定 navigator.plugins 指纹截图。
func (c *Client) ScreenshotWithPlugins(url string, plugins []string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithPlugins(plugins...)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithPluginsBytes 使用指定 navigator.plugins 指纹截图，并返回图片字节。
func (c *Client) ScreenshotWithPluginsBytes(url string, plugins []string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithPlugins(plugins...)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithDisabledWebRTC 禁用 WebRTC API 后截图。
func (c *Client) ScreenshotWithDisabledWebRTC(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithDisableWebRTC()(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithDisabledWebRTCBytes 禁用 WebRTC API 后截图，并返回图片字节。
func (c *Client) ScreenshotWithDisabledWebRTCBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithDisableWebRTC()(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithSpoofedScreen 使用伪造屏幕尺寸截图。
func (c *Client) ScreenshotWithSpoofedScreen(url string, width, height int, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithSpoofedScreen(width, height)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithSpoofedScreenBytes 使用伪造屏幕尺寸截图，并返回图片字节。
func (c *Client) ScreenshotWithSpoofedScreenBytes(url string, width, height int, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithSpoofedScreen(width, height)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithCookieHeader 使用 Cookie header 注入认证状态后截图。
func (c *Client) ScreenshotWithCookieHeader(url string, header string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieHeader(header)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithCookieHeaderBytes 使用 Cookie header 注入认证状态后截图，并返回图片字节。
func (c *Client) ScreenshotWithCookieHeaderBytes(url string, header string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieHeader(header)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithCookieStrings 注入多个 Cookie header 后截图。
func (c *Client) ScreenshotWithCookieStrings(url string, headers []string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieStrings(headers...)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithCookieStringsBytes 注入多个 Cookie header 后截图，并返回图片字节。
func (c *Client) ScreenshotWithCookieStringsBytes(url string, headers []string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieStrings(headers...)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithCookieFile 使用持久化 JSON CookieJar 截图。
func (c *Client) ScreenshotWithCookieFile(url string, path string, writeBack bool, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieFile(path)(opts)
	opts.CookieWriteBack = writeBack
	return c.Screenshot(url, opts)
}

// ScreenshotWithCookieFileBytes 使用持久化 JSON CookieJar 截图，并返回图片字节。
func (c *Client) ScreenshotWithCookieFileBytes(url string, path string, writeBack bool, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieFile(path)(opts)
	opts.CookieWriteBack = writeBack
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithCookieImport 导入 Netscape/Mozilla Cookie 文件后截图。
func (c *Client) ScreenshotWithCookieImport(url string, path string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieImport(path)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithCookieImportBytes 导入 Netscape/Mozilla Cookie 文件后截图，并返回图片字节。
func (c *Client) ScreenshotWithCookieImportBytes(url string, path string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieImport(path)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithCookieExport 截图后导出 Cookie 到 Netscape/Mozilla Cookie 文件。
func (c *Client) ScreenshotWithCookieExport(url string, path string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieExport(path)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithCookieExportBytes 截图后导出 Cookie 到 Netscape/Mozilla Cookie 文件，并返回图片字节。
func (c *Client) ScreenshotWithCookieExportBytes(url string, path string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithCookieExport(path)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithBlacklist 使用自定义 URL 黑名单规则截图。
func (c *Client) ScreenshotWithBlacklist(url string, patterns []string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithBlacklist(patterns...)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithBlacklistBytes 使用自定义 URL 黑名单规则截图，并返回图片字节。
func (c *Client) ScreenshotWithBlacklistBytes(url string, patterns []string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithBlacklist(patterns...)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithBlacklistFile 使用 URL 黑名单文件截图。
func (c *Client) ScreenshotWithBlacklistFile(url string, path string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithBlacklistFile(path)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithBlacklistFileBytes 使用 URL 黑名单文件截图，并返回图片字节。
func (c *Client) ScreenshotWithBlacklistFileBytes(url string, path string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithBlacklistFile(path)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithoutBlacklist 禁用 URL 黑名单后截图。
func (c *Client) ScreenshotWithoutBlacklist(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithNoBlacklist()(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithoutBlacklistBytes 禁用 URL 黑名单后截图，并返回图片字节。
func (c *Client) ScreenshotWithoutBlacklistBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithNoBlacklist()(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithDefaultBlacklist 使用内置 URL 黑名单规则截图。
func (c *Client) ScreenshotWithDefaultBlacklist(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithDefaultBlacklist()(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithDefaultBlacklistBytes 使用内置 URL 黑名单规则截图，并返回图片字节。
func (c *Client) ScreenshotWithDefaultBlacklistBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithDefaultBlacklist()(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithRetries 使用指定最大重试次数截图。
func (c *Client) ScreenshotWithRetries(url string, maxRetries int, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithMaxRetries(maxRetries)(opts)
	return c.Screenshot(url, opts)
}

// ScreenshotWithRetriesBytes 使用指定最大重试次数截图，并返回图片字节。
func (c *Client) ScreenshotWithRetriesBytes(url string, maxRetries int, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithMaxRetries(maxRetries)(opts)
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithActions 截图前执行交互动作序列
// 适用于需要点击按钮、填写表单、滚动页面等交互后再截图的场景
//
// 示例:
//
//	actions := []runner.InteractionAction{
//	    sdk.ActionType("#search", "go-snir"),
//	    sdk.ActionClick("#search-btn"),
//	    sdk.ActionWait(2 * time.Second),
//	}
//	result, _ := client.ScreenshotWithActions("https://example.com", actions, nil)
func (c *Client) ScreenshotWithActions(url string, actions []runner.InteractionAction, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Actions = actions
	return c.Screenshot(url, opts)
}

// ScreenshotWithActionsBytes 截图前执行交互动作序列并返回图片字节。
func (c *Client) ScreenshotWithActionsBytes(url string, actions []runner.InteractionAction, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Actions = actions
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithForm 截图前填写并提交表单
// 适用于登录页面、搜索框等需要表单交互的场景
//
// 示例:
//
//	form := sdk.FormWithSubmit("#login-btn", 3*time.Second,
//	    sdk.FormInput("#username", "admin"),
//	    sdk.FormInput("#password", "pass123"),
//	)
//	result, _ := client.ScreenshotWithForm("https://example.com/login", form, nil)
func (c *Client) ScreenshotWithForm(url string, form runner.Form, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Form = form
	return c.Screenshot(url, opts)
}

// ScreenshotWithFormBytes 截图前填写并提交表单并返回图片字节。
func (c *Client) ScreenshotWithFormBytes(url string, form runner.Form, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Form = form
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotEvidence 截图并收集 HTML、HTTP 头、Cookie、控制台日志和网络请求。
func (c *Client) ScreenshotEvidence(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	return c.ScreenshotEvidenceWithContext(context.Background(), url, screenshotOpts)
}

// ScreenshotEvidenceWithContext 支持取消的全证据截图。
func (c *Client) ScreenshotEvidenceWithContext(ctx context.Context, url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithEvidence()(opts)
	return c.ScreenshotWithContext(ctx, url, opts)
}

// ScreenshotEvidenceBytes 截图、收集全部证据，并返回图片字节。
func (c *Client) ScreenshotEvidenceBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	return c.ScreenshotEvidenceBytesWithContext(context.Background(), url, screenshotOpts)
}

// ScreenshotEvidenceBytesWithContext 支持取消的全证据字节截图。
func (c *Client) ScreenshotEvidenceBytesWithContext(ctx context.Context, url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithEvidence()(opts)
	return c.ScreenshotBytesWithContext(ctx, url, opts)
}

// CaptureEvidenceBundle 使用函数式选项采集全证据并写入证据包目录。
func (c *Client) CaptureEvidenceBundle(url string, dir string, options ...ScreenshotOption) (*EvidenceBundle, *models.Result, error) {
	return c.CaptureEvidenceBundleWithContext(context.Background(), url, dir, options...)
}

// CaptureEvidenceBundleWithContext 使用函数式选项执行可取消的全证据采集并写入证据包目录。
func (c *Client) CaptureEvidenceBundleWithContext(ctx context.Context, url string, dir string, options ...ScreenshotOption) (*EvidenceBundle, *models.Result, error) {
	return c.ScreenshotEvidenceBundleWithContext(ctx, url, dir, NewScreenshotOptions(options...))
}

// ScreenshotEvidenceBundle 截图、收集全部证据，并写入证据包目录。
func (c *Client) ScreenshotEvidenceBundle(url string, dir string, screenshotOpts *ScreenshotOptions) (*EvidenceBundle, *models.Result, error) {
	return c.ScreenshotEvidenceBundleWithContext(context.Background(), url, dir, screenshotOpts)
}

// ScreenshotEvidenceBundleWithContext 支持取消的全证据采集和证据包导出。
func (c *Client) ScreenshotEvidenceBundleWithContext(ctx context.Context, url string, dir string, screenshotOpts *ScreenshotOptions) (*EvidenceBundle, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	WithEvidence()(opts)

	_, result, err := c.ScreenshotBytesWithContext(ctx, url, opts)
	if err != nil {
		return nil, result, err
	}

	bundle, err := WrapResult(result).SaveEvidenceBundle(dir)
	if err != nil {
		return nil, result, err
	}
	return bundle, result, nil
}

// ScreenshotWithCookies 截图前注入自定义 Cookie
// 适用于需要认证状态的页面
//
// 示例:
//
//	cookies := []runner.CustomCookie{
//	    {Name: "session", Value: "abc123", Domain: "example.com"},
//	}
//	result, _ := client.ScreenshotWithCookies("https://example.com/dashboard", cookies, nil)
func (c *Client) ScreenshotWithCookies(url string, cookies []runner.CustomCookie, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Cookies = cookies
	return c.Screenshot(url, opts)
}

// ScreenshotWithCookiesBytes 截图前注入自定义 Cookie 并返回图片字节。
func (c *Client) ScreenshotWithCookiesBytes(url string, cookies []runner.CustomCookie, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Cookies = cookies
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotElement 截取指定 CSS 选择器匹配的元素
// 便捷方法，等价于设置 Selector 后截图
func (c *Client) ScreenshotElement(url string, selector string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Selector = selector
	return c.Screenshot(url, opts)
}

// ScreenshotElementBytes 截取指定 CSS 选择器匹配的元素并返回图片字节。
func (c *Client) ScreenshotElementBytes(url string, selector string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Selector = selector
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotXPath 截取指定 XPath 匹配的元素。
func (c *Client) ScreenshotXPath(url string, xpath string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.XPath = xpath
	return c.Screenshot(url, opts)
}

// ScreenshotXPathBytes 截取指定 XPath 匹配的元素并返回图片字节。
func (c *Client) ScreenshotXPathBytes(url string, xpath string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.XPath = xpath
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotFullPage 截取完整页面（含滚动区域）
// 便捷方法，等价于设置 CaptureFullPage=true 后截图
func (c *Client) ScreenshotFullPage(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.CaptureFullPage = true
	return c.Screenshot(url, opts)
}

// ScreenshotFullPageBytes 截取完整页面并返回图片字节。
func (c *Client) ScreenshotFullPageBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.CaptureFullPage = true
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotDevice 使用指定设备预设截图。
func (c *Client) ScreenshotDevice(url string, device string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Device = device
	return c.Screenshot(url, opts)
}

// ScreenshotDeviceBytes 使用指定设备预设截图并返回图片字节。
func (c *Client) ScreenshotDeviceBytes(url string, device string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Device = device
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotViewport 使用指定 viewport 截图。
func (c *Client) ScreenshotViewport(url string, width, height int, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.WindowWidth = width
	opts.WindowHeight = height
	return c.Screenshot(url, opts)
}

// ScreenshotViewportBytes 使用指定 viewport 截图并返回图片字节。
func (c *Client) ScreenshotViewportBytes(url string, width, height int, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.WindowWidth = width
	opts.WindowHeight = height
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithJS 截图前执行 JavaScript
// 便捷方法，适用于需要操作 DOM 后截图的场景
func (c *Client) ScreenshotWithJS(url string, js string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.JavaScript = js
	opts.RunJSAfter = true
	return c.Screenshot(url, opts)
}

// ScreenshotWithJSBytes 截图前执行 JavaScript 并返回图片字节。
func (c *Client) ScreenshotWithJSBytes(url string, js string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.JavaScript = js
	opts.RunJSAfter = true
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithJSBefore 页面加载前执行 JavaScript 后截图。
func (c *Client) ScreenshotWithJSBefore(url string, js string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.JavaScript = js
	opts.RunJSBefore = true
	opts.RunJSAfter = false
	return c.Screenshot(url, opts)
}

// ScreenshotWithJSBeforeBytes 页面加载前执行 JavaScript 后截图并返回图片字节。
func (c *Client) ScreenshotWithJSBeforeBytes(url string, js string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.JavaScript = js
	opts.RunJSBefore = true
	opts.RunJSAfter = false
	return c.ScreenshotBytes(url, opts)
}

// ScreenshotWithJSFile 执行 JavaScript 文件后截图。
func (c *Client) ScreenshotWithJSFile(url string, jsFile string, beforeLoad bool, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.JavaScriptFile = jsFile
	if beforeLoad {
		opts.RunJSBefore = true
		opts.RunJSAfter = false
	} else {
		opts.RunJSAfter = true
	}
	return c.Screenshot(url, opts)
}

// ScreenshotWithJSFileBytes 执行 JavaScript 文件后截图并返回图片字节。
func (c *Client) ScreenshotWithJSFileBytes(url string, jsFile string, beforeLoad bool, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.JavaScriptFile = jsFile
	if beforeLoad {
		opts.RunJSBefore = true
		opts.RunJSAfter = false
	} else {
		opts.RunJSAfter = true
	}
	return c.ScreenshotBytes(url, opts)
}

// ---------------------------------------------------------------------------
// 批量截图
// ---------------------------------------------------------------------------

// BatchScreenshot 批量截图，并发执行
// urls: 要截图的 URL 列表
// screenshotOpts: 所有 URL 共享的截图配置，传 nil 使用默认配置
// 返回每个 URL 的截图结果，失败的结果也会包含在列表中（检查 Error 字段）
func (c *Client) BatchScreenshot(urls []string, screenshotOpts *ScreenshotOptions) []BatchResult {
	return c.BatchScreenshotWithContext(context.Background(), urls, screenshotOpts)
}

// BatchScreenshotWithContext 支持取消的批量截图
func (c *Client) BatchScreenshotWithContext(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions) []BatchResult {
	results := make([]BatchResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()

			result, err := c.ScreenshotWithContext(ctx, target, screenshotOpts)
			results[idx] = BatchResult{
				URL:    target,
				Result: result,
				Error:  err,
			}
		}(i, url)
	}

	wg.Wait()
	return results
}

// BatchScreenshotBytes 批量截图，并返回每个目标的图片字节。
func (c *Client) BatchScreenshotBytes(urls []string, screenshotOpts *ScreenshotOptions) []BatchBytesResult {
	return c.BatchScreenshotBytesWithContext(context.Background(), urls, screenshotOpts)
}

// BatchScreenshotBytesWithContext 支持取消的批量截图字节数据获取。
func (c *Client) BatchScreenshotBytesWithContext(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions) []BatchBytesResult {
	results := make([]BatchBytesResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()

			data, result, err := c.ScreenshotBytesWithContext(ctx, target, screenshotOpts)
			results[idx] = BatchBytesResult{
				URL:    target,
				Data:   data,
				Result: result,
				Error:  err,
			}
		}(i, url)
	}

	wg.Wait()
	return results
}

// BatchScreenshotRequests captures each request with its own URL and screenshot options.
func (c *Client) BatchScreenshotRequests(requests []ScreenshotRequest) []BatchResult {
	return c.BatchScreenshotRequestsWithContext(context.Background(), requests)
}

// BatchScreenshotRequestsWithContext supports cancellation while capturing per-request screenshot options.
func (c *Client) BatchScreenshotRequestsWithContext(ctx context.Context, requests []ScreenshotRequest) []BatchResult {
	results := make([]BatchResult, len(requests))
	var wg sync.WaitGroup

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req ScreenshotRequest) {
			defer wg.Done()

			result, err := c.ScreenshotWithContext(ctx, req.URL, req.Options)
			results[idx] = BatchResult{
				Name:   req.Name,
				URL:    req.URL,
				Result: result,
				Error:  err,
			}
		}(i, request)
	}

	wg.Wait()
	return results
}

// BatchScreenshotRequestsBytes captures each request with its own options and returns image bytes.
func (c *Client) BatchScreenshotRequestsBytes(requests []ScreenshotRequest) []BatchBytesResult {
	return c.BatchScreenshotRequestsBytesWithContext(context.Background(), requests)
}

// BatchScreenshotRequestsBytesWithContext supports cancellation for per-request byte screenshots.
func (c *Client) BatchScreenshotRequestsBytesWithContext(ctx context.Context, requests []ScreenshotRequest) []BatchBytesResult {
	results := make([]BatchBytesResult, len(requests))
	var wg sync.WaitGroup

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req ScreenshotRequest) {
			defer wg.Done()

			data, result, err := c.ScreenshotBytesWithContext(ctx, req.URL, req.Options)
			results[idx] = BatchBytesResult{
				Name:   req.Name,
				URL:    req.URL,
				Data:   data,
				Result: result,
				Error:  err,
			}
		}(i, request)
	}

	wg.Wait()
	return results
}

// BatchScreenshotEvidenceBundles captures each URL with full evidence and writes one bundle directory per URL.
func (c *Client) BatchScreenshotEvidenceBundles(urls []string, dir string, screenshotOpts *ScreenshotOptions) []BatchEvidenceBundleResult {
	return c.BatchScreenshotEvidenceBundlesWithContext(context.Background(), urls, dir, screenshotOpts)
}

// BatchScreenshotEvidenceBundlesWithContext supports cancellation while writing one evidence bundle per URL.
func (c *Client) BatchScreenshotEvidenceBundlesWithContext(ctx context.Context, urls []string, dir string, screenshotOpts *ScreenshotOptions) []BatchEvidenceBundleResult {
	results := make([]BatchEvidenceBundleResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()

			bundleDir := batchEvidenceBundleDir(dir, idx, "", target)
			bundle, result, err := c.ScreenshotEvidenceBundleWithContext(ctx, target, bundleDir, screenshotOpts)
			results[idx] = BatchEvidenceBundleResult{
				URL:    target,
				Dir:    bundleDir,
				Bundle: bundle,
				Result: result,
				Error:  err,
			}
		}(i, url)
	}

	wg.Wait()
	return results
}

// BatchScreenshotRequestsEvidenceBundles captures each request with its own options and writes evidence bundles.
func (c *Client) BatchScreenshotRequestsEvidenceBundles(requests []ScreenshotRequest, dir string) []BatchEvidenceBundleResult {
	return c.BatchScreenshotRequestsEvidenceBundlesWithContext(context.Background(), requests, dir)
}

// BatchScreenshotRequestsEvidenceBundlesWithContext supports cancellation for per-request evidence bundle capture.
func (c *Client) BatchScreenshotRequestsEvidenceBundlesWithContext(ctx context.Context, requests []ScreenshotRequest, dir string) []BatchEvidenceBundleResult {
	results := make([]BatchEvidenceBundleResult, len(requests))
	var wg sync.WaitGroup

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req ScreenshotRequest) {
			defer wg.Done()

			bundleDir := batchEvidenceBundleDir(dir, idx, req.Name, req.URL)
			bundle, result, err := c.ScreenshotEvidenceBundleWithContext(ctx, req.URL, bundleDir, req.Options)
			results[idx] = BatchEvidenceBundleResult{
				Name:   req.Name,
				URL:    req.URL,
				Dir:    bundleDir,
				Bundle: bundle,
				Result: result,
				Error:  err,
			}
		}(i, request)
	}

	wg.Wait()
	return results
}

// BatchScreenshotTargets expands bare hosts/IPs by configured schemes and ports, then captures each URL.
func (c *Client) BatchScreenshotTargets(targets []string, screenshotOpts *ScreenshotOptions) []BatchResult {
	return c.BatchScreenshotTargetsWithContext(context.Background(), targets, screenshotOpts)
}

// BatchScreenshotTargetsWithContext supports cancellation while capturing expanded host/IP targets.
func (c *Client) BatchScreenshotTargetsWithContext(ctx context.Context, targets []string, screenshotOpts *ScreenshotOptions) []BatchResult {
	expanded := c.ExpandTargets(targets, screenshotOpts)
	return c.BatchScreenshotWithContext(ctx, expanded, screenshotOpts)
}

// BatchScreenshotTargetsBytes expands bare hosts/IPs, then captures each target as image bytes.
func (c *Client) BatchScreenshotTargetsBytes(targets []string, screenshotOpts *ScreenshotOptions) []BatchBytesResult {
	return c.BatchScreenshotTargetsBytesWithContext(context.Background(), targets, screenshotOpts)
}

// BatchScreenshotTargetsBytesWithContext supports cancellation while capturing expanded targets as image bytes.
func (c *Client) BatchScreenshotTargetsBytesWithContext(ctx context.Context, targets []string, screenshotOpts *ScreenshotOptions) []BatchBytesResult {
	expanded := c.ExpandTargets(targets, screenshotOpts)
	return c.BatchScreenshotBytesWithContext(ctx, expanded, screenshotOpts)
}

// BatchScreenshotTargetsEvidenceBundles expands bare hosts/IPs, then writes evidence bundles for each expanded URL.
func (c *Client) BatchScreenshotTargetsEvidenceBundles(targets []string, dir string, screenshotOpts *ScreenshotOptions) []BatchEvidenceBundleResult {
	return c.BatchScreenshotTargetsEvidenceBundlesWithContext(context.Background(), targets, dir, screenshotOpts)
}

// BatchScreenshotTargetsEvidenceBundlesWithContext supports cancellation while writing expanded target evidence bundles.
func (c *Client) BatchScreenshotTargetsEvidenceBundlesWithContext(ctx context.Context, targets []string, dir string, screenshotOpts *ScreenshotOptions) []BatchEvidenceBundleResult {
	expanded := c.ExpandTargets(targets, screenshotOpts)
	return c.BatchScreenshotEvidenceBundlesWithContext(ctx, expanded, dir, screenshotOpts)
}

// BatchScreenshotStreaming 流式批量截图
// 每完成一个截图立即通过 channel 返回，不用等全部完成
// 适用于大量 URL 截图、进度展示、实时处理等场景
//
// 示例:
//
//	ch := client.BatchScreenshotStreaming(ctx, urls, nil)
//	for result := range ch {
//	    if result.Error != nil {
//	        log.Printf("失败: %s - %v", result.URL, result.Error)
//	    } else {
//	        log.Printf("完成: %s - %s", result.URL, result.Result.Title)
//	    }
//	}
func (c *Client) BatchScreenshotStreaming(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions) <-chan BatchResult {
	ch := make(chan BatchResult, len(urls))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for _, url := range urls {
			// 检查 context 是否已取消
			select {
			case <-ctx.Done():
				ch <- BatchResult{URL: url, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(target string) {
				defer wg.Done()

				result, err := c.ScreenshotWithContext(ctx, target, screenshotOpts)
				ch <- BatchResult{
					URL:    target,
					Result: result,
					Error:  err,
				}
			}(url)
		}

		wg.Wait()
	}()

	return ch
}

// BatchScreenshotBytesStreaming streams byte screenshot results as each URL completes.
func (c *Client) BatchScreenshotBytesStreaming(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions) <-chan BatchBytesResult {
	ch := make(chan BatchBytesResult, len(urls))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for _, url := range urls {
			select {
			case <-ctx.Done():
				ch <- BatchBytesResult{URL: url, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(target string) {
				defer wg.Done()

				data, result, err := c.ScreenshotBytesWithContext(ctx, target, screenshotOpts)
				ch <- BatchBytesResult{
					URL:    target,
					Data:   data,
					Result: result,
					Error:  err,
				}
			}(url)
		}

		wg.Wait()
	}()

	return ch
}

// BatchScreenshotEvidenceBundlesStreaming streams evidence bundle results as each URL completes.
func (c *Client) BatchScreenshotEvidenceBundlesStreaming(ctx context.Context, urls []string, dir string, screenshotOpts *ScreenshotOptions) <-chan BatchEvidenceBundleResult {
	ch := make(chan BatchEvidenceBundleResult, len(urls))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for i, url := range urls {
			bundleDir := batchEvidenceBundleDir(dir, i, "", url)
			select {
			case <-ctx.Done():
				ch <- BatchEvidenceBundleResult{URL: url, Dir: bundleDir, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(target string, targetDir string) {
				defer wg.Done()

				bundle, result, err := c.ScreenshotEvidenceBundleWithContext(ctx, target, targetDir, screenshotOpts)
				ch <- BatchEvidenceBundleResult{
					URL:    target,
					Dir:    targetDir,
					Bundle: bundle,
					Result: result,
					Error:  err,
				}
			}(url, bundleDir)
		}

		wg.Wait()
	}()

	return ch
}

// BatchScreenshotRequestsStreaming streams per-request screenshot results as each one completes.
func (c *Client) BatchScreenshotRequestsStreaming(ctx context.Context, requests []ScreenshotRequest) <-chan BatchResult {
	ch := make(chan BatchResult, len(requests))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for _, request := range requests {
			select {
			case <-ctx.Done():
				ch <- BatchResult{Name: request.Name, URL: request.URL, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(req ScreenshotRequest) {
				defer wg.Done()

				result, err := c.ScreenshotWithContext(ctx, req.URL, req.Options)
				ch <- BatchResult{
					Name:   req.Name,
					URL:    req.URL,
					Result: result,
					Error:  err,
				}
			}(request)
		}

		wg.Wait()
	}()

	return ch
}

// BatchScreenshotRequestsBytesStreaming streams per-request byte screenshot results as each one completes.
func (c *Client) BatchScreenshotRequestsBytesStreaming(ctx context.Context, requests []ScreenshotRequest) <-chan BatchBytesResult {
	ch := make(chan BatchBytesResult, len(requests))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for _, request := range requests {
			select {
			case <-ctx.Done():
				ch <- BatchBytesResult{Name: request.Name, URL: request.URL, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(req ScreenshotRequest) {
				defer wg.Done()

				data, result, err := c.ScreenshotBytesWithContext(ctx, req.URL, req.Options)
				ch <- BatchBytesResult{
					Name:   req.Name,
					URL:    req.URL,
					Data:   data,
					Result: result,
					Error:  err,
				}
			}(request)
		}

		wg.Wait()
	}()

	return ch
}

// BatchScreenshotRequestsEvidenceBundlesStreaming streams per-request evidence bundle results as each one completes.
func (c *Client) BatchScreenshotRequestsEvidenceBundlesStreaming(ctx context.Context, requests []ScreenshotRequest, dir string) <-chan BatchEvidenceBundleResult {
	ch := make(chan BatchEvidenceBundleResult, len(requests))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for i, request := range requests {
			bundleDir := batchEvidenceBundleDir(dir, i, request.Name, request.URL)
			select {
			case <-ctx.Done():
				ch <- BatchEvidenceBundleResult{Name: request.Name, URL: request.URL, Dir: bundleDir, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(req ScreenshotRequest, targetDir string) {
				defer wg.Done()

				bundle, result, err := c.ScreenshotEvidenceBundleWithContext(ctx, req.URL, targetDir, req.Options)
				ch <- BatchEvidenceBundleResult{
					Name:   req.Name,
					URL:    req.URL,
					Dir:    targetDir,
					Bundle: bundle,
					Result: result,
					Error:  err,
				}
			}(request, bundleDir)
		}

		wg.Wait()
	}()

	return ch
}

// BatchScreenshotTargetsStreaming expands bare hosts/IPs, then streams each screenshot result as it completes.
func (c *Client) BatchScreenshotTargetsStreaming(ctx context.Context, targets []string, screenshotOpts *ScreenshotOptions) <-chan BatchResult {
	expanded := c.ExpandTargets(targets, screenshotOpts)
	return c.BatchScreenshotStreaming(ctx, expanded, screenshotOpts)
}

// BatchScreenshotTargetsBytesStreaming expands bare hosts/IPs, then streams byte screenshot results.
func (c *Client) BatchScreenshotTargetsBytesStreaming(ctx context.Context, targets []string, screenshotOpts *ScreenshotOptions) <-chan BatchBytesResult {
	expanded := c.ExpandTargets(targets, screenshotOpts)
	return c.BatchScreenshotBytesStreaming(ctx, expanded, screenshotOpts)
}

// BatchScreenshotTargetsEvidenceBundlesStreaming expands bare hosts/IPs, then streams evidence bundle results.
func (c *Client) BatchScreenshotTargetsEvidenceBundlesStreaming(ctx context.Context, targets []string, dir string, screenshotOpts *ScreenshotOptions) <-chan BatchEvidenceBundleResult {
	expanded := c.ExpandTargets(targets, screenshotOpts)
	return c.BatchScreenshotEvidenceBundlesStreaming(ctx, expanded, dir, screenshotOpts)
}

// BatchScreenshotTargetsCallback expands bare hosts/IPs, then calls callback for each completed result.
func (c *Client) BatchScreenshotTargetsCallback(ctx context.Context, targets []string, screenshotOpts *ScreenshotOptions, callback func(BatchResult)) {
	ch := c.BatchScreenshotTargetsStreaming(ctx, targets, screenshotOpts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotTargetsBytesCallback expands bare hosts/IPs, then calls callback for each completed byte result.
func (c *Client) BatchScreenshotTargetsBytesCallback(ctx context.Context, targets []string, screenshotOpts *ScreenshotOptions, callback func(BatchBytesResult)) {
	ch := c.BatchScreenshotTargetsBytesStreaming(ctx, targets, screenshotOpts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotTargetsEvidenceBundlesCallback expands bare hosts/IPs, then calls callback for each evidence bundle result.
func (c *Client) BatchScreenshotTargetsEvidenceBundlesCallback(ctx context.Context, targets []string, dir string, screenshotOpts *ScreenshotOptions, callback func(BatchEvidenceBundleResult)) {
	ch := c.BatchScreenshotTargetsEvidenceBundlesStreaming(ctx, targets, dir, screenshotOpts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotCallback 批量截图，每完成一个调用回调函数
// 适用于需要逐个处理结果的场景
// callback 在截图完成时同步调用，可用于进度展示、结果处理等
func (c *Client) BatchScreenshotCallback(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions, callback func(BatchResult)) {
	ch := c.BatchScreenshotStreaming(ctx, urls, screenshotOpts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotBytesCallback calls callback for each completed byte screenshot result.
func (c *Client) BatchScreenshotBytesCallback(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions, callback func(BatchBytesResult)) {
	ch := c.BatchScreenshotBytesStreaming(ctx, urls, screenshotOpts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotEvidenceBundlesCallback calls callback for each completed evidence bundle result.
func (c *Client) BatchScreenshotEvidenceBundlesCallback(ctx context.Context, urls []string, dir string, screenshotOpts *ScreenshotOptions, callback func(BatchEvidenceBundleResult)) {
	ch := c.BatchScreenshotEvidenceBundlesStreaming(ctx, urls, dir, screenshotOpts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotRequestsCallback calls callback for each completed per-request result.
func (c *Client) BatchScreenshotRequestsCallback(ctx context.Context, requests []ScreenshotRequest, callback func(BatchResult)) {
	ch := c.BatchScreenshotRequestsStreaming(ctx, requests)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotRequestsBytesCallback calls callback for each completed per-request byte result.
func (c *Client) BatchScreenshotRequestsBytesCallback(ctx context.Context, requests []ScreenshotRequest, callback func(BatchBytesResult)) {
	ch := c.BatchScreenshotRequestsBytesStreaming(ctx, requests)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// BatchScreenshotRequestsEvidenceBundlesCallback calls callback for each completed per-request evidence bundle result.
func (c *Client) BatchScreenshotRequestsEvidenceBundlesCallback(ctx context.Context, requests []ScreenshotRequest, dir string, callback func(BatchEvidenceBundleResult)) {
	ch := c.BatchScreenshotRequestsEvidenceBundlesStreaming(ctx, requests, dir)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// ---------------------------------------------------------------------------
// 池管理
// ---------------------------------------------------------------------------

// Stats 返回连接池统计信息
func (c *Client) Stats() runner.PoolStats {
	return c.pool.Stats()
}

// SetIdleTimeout 设置空闲超时
// 当客户端空闲超过此时间后，自动关闭浏览器进程释放资源
// 下次截图时会自动重启浏览器进程
// 设为 0 表示不自动关闭（默认行为）
func (c *Client) SetIdleTimeout(timeout time.Duration) {
	c.pool.SetIdleTimeout(timeout)
}

// OnEvent 注册池事件监听器
// 事件类型:
//   - screenshot_start: 截图开始
//   - screenshot_complete: 截图完成（含耗时和结果）
//   - screenshot_failed: 截图失败（含错误信息）
//   - reconnect: 浏览器进程重新连接
//   - idle_close: 空闲超时关闭浏览器
//   - pool_closed: 连接池关闭
//
// 回调是异步执行的，不会阻塞截图流程
func (c *Client) OnEvent(handler runner.PoolEventHandler) {
	c.pool.On(handler)
}

// ActiveCount 返回当前正在执行的截图数
func (c *Client) ActiveCount() int {
	return c.pool.ActiveCount()
}

// SetCookieJar 设置 Cookie 持久化存储
// 设置后，截图时自动从 CookieJar 加载对应域名的 Cookie
// 一次性 Cookie 获取后自动移除，持久化 Cookie 保留
func (c *Client) SetCookieJar(jar *runner.CookieJar) {
	c.cookieJar = jar
}

// CookieJar 返回当前的 Cookie 持久化存储
func (c *Client) CookieJar() *runner.CookieJar {
	return c.cookieJar
}

// AddCookie 添加 Cookie 到持久化存储
// 如果没有 CookieJar，会自动创建一个内存中的 CookieJar
func (c *Client) AddCookie(cookie runner.PersistentCookie) error {
	if c.cookieJar == nil {
		jar, err := newCookieJar("")
		if err != nil {
			return err
		}
		c.cookieJar = jar
	}
	return c.cookieJar.AddCookie(cookie)
}

// AddPersistentCookie 添加持久化 Cookie
func (c *Client) AddPersistentCookie(name, value, domain string) error {
	return c.AddCookie(runner.PersistentCookie{
		Name:       name,
		Value:      value,
		Domain:     domain,
		Persistent: true,
	})
}

// AddOneTimeCookie 添加一次性 Cookie（获取后自动移除）
func (c *Client) AddOneTimeCookie(name, value, domain string) error {
	return c.AddCookie(runner.PersistentCookie{
		Name:       name,
		Value:      value,
		Domain:     domain,
		Persistent: false,
	})
}

// Close 关闭客户端，释放浏览器进程
// 调用后客户端不可再使用
func (c *Client) Close() {
	c.pool.Close()
	log.Info("截图SDK客户端已关闭")
}

// ---------------------------------------------------------------------------
// 类型定义
// ---------------------------------------------------------------------------

// BatchResult 批量截图中的单个结果
type BatchResult struct {
	Name   string         `json:"name,omitempty"`
	URL    string         `json:"url"`
	Result *models.Result `json:"result,omitempty"`
	Error  error          `json:"error,omitempty"`
}

// BatchBytesResult 批量截图中的单个字节结果
type BatchBytesResult struct {
	Name   string         `json:"name,omitempty"`
	URL    string         `json:"url"`
	Data   []byte         `json:"data,omitempty"`
	Result *models.Result `json:"result,omitempty"`
	Error  error          `json:"error,omitempty"`
}

// BatchEvidenceBundleResult 批量证据包采集中的单个结果。
type BatchEvidenceBundleResult struct {
	Name   string          `json:"name,omitempty"`
	URL    string          `json:"url"`
	Dir    string          `json:"dir"`
	Bundle *EvidenceBundle `json:"bundle,omitempty"`
	Result *models.Result  `json:"result,omitempty"`
	Error  error           `json:"error,omitempty"`
}

// ScreenshotRequest describes one batch item with independent screenshot options.
type ScreenshotRequest struct {
	Name    string             `json:"name,omitempty"`
	URL     string             `json:"url"`
	Options *ScreenshotOptions `json:"options,omitempty"`
}

// ---------------------------------------------------------------------------
// 内部辅助
// ---------------------------------------------------------------------------

// ensureScreenshotOptions 确保 ScreenshotOptions 不为 nil
// 如果传入 nil，返回一个新的零值 ScreenshotOptions
func (c *Client) ensureScreenshotOptions(opts *ScreenshotOptions) *ScreenshotOptions {
	if opts != nil {
		return opts
	}
	return &ScreenshotOptions{}
}

func batchEvidenceBundleDir(root string, idx int, name string, target string) string {
	if root == "" {
		return ""
	}
	label := name
	if label == "" {
		label = target
	}
	label = islazy.SanitizeFilename(label)
	runes := []rune(label)
	if len(runes) > 96 {
		label = string(runes[:96])
	}
	return filepath.Join(root, fmt.Sprintf("%03d_%s", idx+1, label))
}

func (c *Client) runnerOptionsForScreenshot(target string, screenshotOpts *ScreenshotOptions) runner.Options {
	runnerOpts := toRunnerOptions(c.opts)
	runnerOpts = mergeWithScreenshotOptions(runnerOpts, screenshotOpts)
	appendCookieSources(target, &runnerOpts)
	c.mergeCookieJar(target, &runnerOpts)
	return runnerOpts
}

func rejectBlacklistedTarget(target string, opts *runner.Options) (*models.Result, error) {
	result, err := blacklistedResult(target, opts)
	if err != nil {
		return nil, fmt.Errorf("初始化URL黑名单失败: %v", err)
	}
	if result != nil {
		return result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}
	return nil, nil
}

func blacklistedResult(target string, opts *runner.Options) (*models.Result, error) {
	blacklist, err := runner.NewURLBlacklist(opts)
	if err != nil {
		return nil, err
	}
	if isBlacklisted, reason := blacklist.IsBlacklisted(target); isBlacklisted {
		result := &models.Result{
			URL:          target,
			ProbedAt:     time.Now(),
			Failed:       true,
			FailedReason: fmt.Sprintf("URL在黑名单中: %s", reason),
		}
		models.EnrichEndpoint(result)
		return result, nil
	}
	return nil, nil
}

func (c *Client) mergeCookieJar(target string, opts *runner.Options) {
	jar := c.cookieJarForOptions(opts)
	if jar == nil {
		return
	}
	jarCookies := jar.GetCookies(extractDomain(target))
	if len(jarCookies) == 0 {
		return
	}

	// CookieJar 中的 Cookie 在前，调用参数中的 Cookie 在后，便于单次调用覆盖。
	allCookies := append(jarCookies, opts.Scan.Cookies...)
	opts.Scan.Cookies = allCookies
}

func (c *Client) cookieJarForOptions(opts *runner.Options) *runner.CookieJar {
	if opts == nil || opts.Scan.CookiesFile == "" {
		return c.cookieJar
	}

	if opts.Scan.CookiesFile == c.opts.CookieFile {
		if c.cookieJar != nil {
			return c.cookieJar
		}
		jar, err := newCookieJar(opts.Scan.CookiesFile)
		if err != nil {
			log.Warn("SDK: 加载 Cookie 持久化文件失败", "file", opts.Scan.CookiesFile, "error", err)
			return nil
		}
		c.cookieJar = jar
		return jar
	}

	jar, err := newCookieJar(opts.Scan.CookiesFile)
	if err != nil {
		log.Warn("SDK: 加载单次 Cookie 持久化文件失败", "file", opts.Scan.CookiesFile, "error", err)
		return nil
	}
	return jar
}

func appendCookieSources(target string, opts *runner.Options) {
	defaultDomain := extractDomain(target)
	for _, header := range opts.Scan.CookieStrings {
		parsed := runner.ParseCookieHeader(header, defaultDomain)
		opts.Scan.Cookies = append(opts.Scan.Cookies, parsed...)
	}

	if opts.Scan.CookieImport == "" {
		return
	}
	imported, err := runner.LoadNetscapeCookieFile(opts.Scan.CookieImport)
	if err != nil {
		log.Warn("SDK: 导入 Netscape Cookie 失败", "file", opts.Scan.CookieImport, "error", err)
		return
	}
	opts.Scan.Cookies = append(opts.Scan.Cookies, imported...)
}

func (c *Client) handleResultCookies(target string, result *models.Result, opts *runner.Options) {
	if result == nil || len(result.Cookies) == 0 {
		return
	}

	if opts.Scan.CookieWriteBack {
		c.writeBackResultCookies(target, result.Cookies, c.cookieJarForOptions(opts))
	}

	if opts.Scan.CookieExport != "" {
		if err := runner.ExportResultCookiesToNetscape(opts.Scan.CookieExport, result.Cookies, target); err != nil {
			log.Warn("SDK: 导出 Netscape Cookie 失败", "file", opts.Scan.CookieExport, "error", err)
		}
	}
}

func (c *Client) writeBackResultCookies(target string, cookies []models.Cookie, jar *runner.CookieJar) {
	if jar == nil {
		var err error
		jar, err = newCookieJar("")
		if err != nil {
			log.Warn("SDK: 创建内存 CookieJar 失败", "error", err)
			return
		}
		c.cookieJar = jar
	}

	defaultDomain := extractDomain(target)
	for _, cookie := range cookies {
		cookieDomain := cookie.Domain
		if cookieDomain == "" {
			cookieDomain = defaultDomain
		}
		if err := jar.AddCookie(runner.PersistentCookie{
			Name:       cookie.Name,
			Value:      cookie.Value,
			Domain:     cookieDomain,
			Path:       cookie.Path,
			Persistent: true,
			Source:     "session",
		}); err != nil {
			log.Warn("SDK: 写回 Cookie 失败", "domain", cookieDomain, "name", cookie.Name, "error", err)
		}
	}
}

// extractDomain 从 URL 中提取域名
func extractDomain(rawURL string) string {
	// 简单提取：去掉协议和路径
	u := rawURL
	// 去掉协议
	for _, prefix := range []string{"http://", "https://"} {
		if len(u) > len(prefix) && u[:len(prefix)] == prefix {
			u = u[len(prefix):]
			break
		}
	}
	// 去掉路径
	for i, c := range u {
		if c == '/' || c == ':' || c == '?' || c == '#' {
			u = u[:i]
			break
		}
	}
	return u
}
