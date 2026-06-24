package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

var (
	sharedScreenshotWithContext = runner.SharedScreenshotWithContext
	sharedSetIdleTimeout        = runner.SharedSetIdleTimeout
	sharedPoolStats             = runner.SharedPoolStats
	closeSharedPool             = runner.CloseSharedPool
)

// SharedScreenshot 使用进程级共享池执行截图
// 多个包/模块调用此函数会自动复用同一个 Chrome 进程
// 首次调用时自动初始化池，后续调用直接复用
func SharedScreenshot(url string, opts *ScreenshotOptions) (*models.Result, error) {
	return SharedScreenshotWithContext(context.Background(), url, opts)
}

// SharedCapture 使用函数式选项通过进程级共享池执行截图。
func SharedCapture(url string, options ...ScreenshotOption) (*models.Result, error) {
	return SharedCaptureWithContext(context.Background(), url, options...)
}

// SharedCaptureWithContext 使用函数式选项通过进程级共享池执行可取消截图。
func SharedCaptureWithContext(ctx context.Context, url string, options ...ScreenshotOption) (*models.Result, error) {
	return SharedScreenshotWithContext(ctx, url, NewScreenshotOptions(options...))
}

// SharedScreenshotWithContext 使用共享池执行截图（支持取消）
func SharedScreenshotWithContext(ctx context.Context, url string, opts *ScreenshotOptions) (*models.Result, error) {
	runnerOpts := sharedRunnerOptionsForScreenshot(url, opts)
	if result, err := blacklistedResult(url, &runnerOpts); err != nil {
		return nil, err
	} else if result != nil {
		return result, nil
	}

	result, err := sharedScreenshotWithContext(ctx, url, &runnerOpts)
	if err != nil {
		return nil, err
	}

	sharedHandleResultCookies(url, result, &runnerOpts)

	if result.Failed {
		return result, nil
	}

	return result, nil
}

// SharedScreenshotBytes 使用共享池执行截图，返回截图原始字节数据。
func SharedScreenshotBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	return SharedScreenshotBytesWithContext(context.Background(), url, opts)
}

// SharedCaptureBytes 使用函数式选项通过共享池执行截图，返回截图原始字节数据。
func SharedCaptureBytes(url string, options ...ScreenshotOption) ([]byte, *models.Result, error) {
	return SharedCaptureBytesWithContext(context.Background(), url, options...)
}

// SharedCaptureBytesWithContext 使用函数式选项通过共享池执行可取消截图，返回截图原始字节数据。
func SharedCaptureBytesWithContext(ctx context.Context, url string, options ...ScreenshotOption) ([]byte, *models.Result, error) {
	return SharedScreenshotBytesWithContext(ctx, url, NewScreenshotOptions(options...))
}

// SharedScreenshotBytesWithContext 使用共享池执行可取消截图，返回截图原始字节数据。
func SharedScreenshotBytesWithContext(ctx context.Context, url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	runnerOpts := sharedRunnerOptionsForScreenshot(url, opts)
	runnerOpts.Scan.ReturnScreenshotBytes = true
	runnerOpts.Scan.ScreenshotSkipSave = true
	if result, err := rejectBlacklistedTarget(url, &runnerOpts); err != nil {
		return nil, result, err
	}

	result, err := sharedScreenshotWithContext(ctx, url, &runnerOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("截图失败: %v", err)
	}

	sharedHandleResultCookies(url, result, &runnerOpts)

	if result.Failed {
		return nil, result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}

	data, err := screenshotBytesFromResult(result)
	if err != nil {
		return nil, result, err
	}

	return data, result, nil
}

// SharedScreenshotHTML 使用共享池截图并返回页面 HTML 源码。
func SharedScreenshotHTML(url string, opts *ScreenshotOptions) (string, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.SaveHTML = true

	result, err := SharedScreenshot(url, screenshotOpts)
	if err != nil {
		return "", nil, err
	}
	return result.HTML, result, nil
}

// SharedScreenshotHeaders 使用共享池截图并收集 HTTP 头。
func SharedScreenshotHeaders(url string, opts *ScreenshotOptions) ([]models.Header, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.SaveHeaders = true

	result, err := SharedScreenshot(url, screenshotOpts)
	if err != nil {
		return nil, nil, err
	}
	return result.Headers, result, nil
}

// SharedScreenshotHeadersBytes 使用共享池截图、收集 HTTP 头，并返回图片字节。
func SharedScreenshotHeadersBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.SaveHeaders = true
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotConsole 使用共享池截图并收集浏览器控制台日志。
func SharedScreenshotConsole(url string, opts *ScreenshotOptions) ([]models.ConsoleLog, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.SaveConsole = true

	result, err := SharedScreenshot(url, screenshotOpts)
	if err != nil {
		return nil, nil, err
	}
	return result.Console, result, nil
}

// SharedScreenshotConsoleBytes 使用共享池截图、收集浏览器控制台日志，并返回图片字节。
func SharedScreenshotConsoleBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.SaveConsole = true
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotNetwork 使用共享池截图并收集网络请求日志。
func SharedScreenshotNetwork(url string, opts *ScreenshotOptions) ([]models.NetworkLog, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.SaveNetwork = true

	result, err := SharedScreenshot(url, screenshotOpts)
	if err != nil {
		return nil, nil, err
	}
	return result.Network, result, nil
}

// SharedScreenshotNetworkBytes 使用共享池截图、收集网络请求日志，并返回图片字节。
func SharedScreenshotNetworkBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.SaveNetwork = true
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithFormat 使用共享池按指定图片格式和质量截图。
func SharedScreenshotWithFormat(url string, format string, quality int, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.ScreenshotFormat = format
	screenshotOpts.ScreenshotQuality = quality
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithFormatBytes 使用共享池按指定图片格式和质量截图，并返回图片字节。
func SharedScreenshotWithFormatBytes(url string, format string, quality int, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.ScreenshotFormat = format
	screenshotOpts.ScreenshotQuality = quality
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotToPath 使用共享池将截图保存到指定输出目录或文件路径。
func SharedScreenshotToPath(url string, path string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.ScreenshotPath = path
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithDelay 使用共享池等待指定时间后截图。
func SharedScreenshotWithDelay(url string, delay time.Duration, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Delay = delay
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithDelayBytes 使用共享池等待指定时间后截图，并返回图片字节。
func SharedScreenshotWithDelayBytes(url string, delay time.Duration, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Delay = delay
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithTimeout 使用共享池按指定页面加载超时截图。
func SharedScreenshotWithTimeout(url string, timeout time.Duration, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Timeout = timeout
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithTimeoutBytes 使用共享池按指定页面加载超时截图，并返回图片字节。
func SharedScreenshotWithTimeoutBytes(url string, timeout time.Duration, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Timeout = timeout
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithProxy 使用共享池通过指定代理截图。
func SharedScreenshotWithProxy(url string, proxy string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxy(proxy)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithProxyBytes 使用共享池通过指定代理截图，并返回图片字节。
func SharedScreenshotWithProxyBytes(url string, proxy string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxy(proxy)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithProxyList 使用共享池按代理列表和轮换策略截图。
func SharedScreenshotWithProxyList(url string, strategy runner.ProxyStrategy, proxies []string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxyList(strategy, proxies...)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithProxyListBytes 使用共享池按代理列表和轮换策略截图，并返回图片字节。
func SharedScreenshotWithProxyListBytes(url string, strategy runner.ProxyStrategy, proxies []string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxyList(strategy, proxies...)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithProxyFile 使用共享池按代理文件和轮换策略截图。
func SharedScreenshotWithProxyFile(url string, path string, strategy runner.ProxyStrategy, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxyFile(path, strategy)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithProxyFileBytes 使用共享池按代理文件和轮换策略截图，并返回图片字节。
func SharedScreenshotWithProxyFileBytes(url string, path string, strategy runner.ProxyStrategy, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxyFile(path, strategy)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithProxyURL 使用共享池按动态代理 API 和轮换策略截图。
func SharedScreenshotWithProxyURL(url string, proxyURL string, strategy runner.ProxyStrategy, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxyURL(proxyURL, strategy)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithProxyURLBytes 使用共享池按动态代理 API 和轮换策略截图，并返回图片字节。
func SharedScreenshotWithProxyURLBytes(url string, proxyURL string, strategy runner.ProxyStrategy, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithProxyURL(proxyURL, strategy)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithCustomHeaders 使用共享池按自定义请求头截图。
func SharedScreenshotWithCustomHeaders(url string, headers map[string]string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCustomHeaders(headers)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithCustomHeadersBytes 使用共享池按自定义请求头截图，并返回图片字节。
func SharedScreenshotWithCustomHeadersBytes(url string, headers map[string]string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCustomHeaders(headers)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithUserAgent 使用共享池按指定 User-Agent 截图。
func SharedScreenshotWithUserAgent(url string, userAgent string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithUserAgent(userAgent)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithUserAgentBytes 使用共享池按指定 User-Agent 截图，并返回图片字节。
func SharedScreenshotWithUserAgentBytes(url string, userAgent string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithUserAgent(userAgent)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithAcceptLanguage 使用共享池按指定 Accept-Language 截图。
func SharedScreenshotWithAcceptLanguage(url string, language string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithAcceptLanguage(language)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithAcceptLanguageBytes 使用共享池按指定 Accept-Language 截图，并返回图片字节。
func SharedScreenshotWithAcceptLanguageBytes(url string, language string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithAcceptLanguage(language)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithFingerprint 使用共享池按浏览器指纹覆盖截图。
func SharedScreenshotWithFingerprint(url string, platform, vendor, webGLVendor, webGLRenderer string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithFingerprint(platform, vendor, webGLVendor, webGLRenderer)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithFingerprintBytes 使用共享池按浏览器指纹覆盖截图，并返回图片字节。
func SharedScreenshotWithFingerprintBytes(url string, platform, vendor, webGLVendor, webGLRenderer string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithFingerprint(platform, vendor, webGLVendor, webGLRenderer)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithDeviceEmulation 使用共享池按自定义设备参数截图。
func SharedScreenshotWithDeviceEmulation(url string, width, height int, scaleFactor float64, isMobile, hasTouch bool, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithDeviceEmulation(width, height, scaleFactor, isMobile, hasTouch)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithDeviceEmulationBytes 使用共享池按自定义设备参数截图，并返回图片字节。
func SharedScreenshotWithDeviceEmulationBytes(url string, width, height int, scaleFactor float64, isMobile, hasTouch bool, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithDeviceEmulation(width, height, scaleFactor, isMobile, hasTouch)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithMobileEmulation 使用共享池按移动端和触摸仿真截图。
func SharedScreenshotWithMobileEmulation(url string, scaleFactor float64, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithMobileEmulation(scaleFactor)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithMobileEmulationBytes 使用共享池按移动端和触摸仿真截图，并返回图片字节。
func SharedScreenshotWithMobileEmulationBytes(url string, scaleFactor float64, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithMobileEmulation(scaleFactor)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithTouchEmulation 使用共享池按指定触摸仿真状态截图。
func SharedScreenshotWithTouchEmulation(url string, enabled bool, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithTouchEmulation(enabled)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithTouchEmulationBytes 使用共享池按指定触摸仿真状态截图，并返回图片字节。
func SharedScreenshotWithTouchEmulationBytes(url string, enabled bool, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithTouchEmulation(enabled)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithIgnoreCertErrors 使用共享池忽略证书错误后截图。
func SharedScreenshotWithIgnoreCertErrors(url string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithIgnoreCertErrors()(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithIgnoreCertErrorsBytes 使用共享池忽略证书错误后截图，并返回图片字节。
func SharedScreenshotWithIgnoreCertErrorsBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithIgnoreCertErrors()(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithPlugins 使用共享池按指定 navigator.plugins 指纹截图。
func SharedScreenshotWithPlugins(url string, plugins []string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithPlugins(plugins...)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithPluginsBytes 使用共享池按指定 navigator.plugins 指纹截图，并返回图片字节。
func SharedScreenshotWithPluginsBytes(url string, plugins []string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithPlugins(plugins...)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithDisabledWebRTC 使用共享池禁用 WebRTC API 后截图。
func SharedScreenshotWithDisabledWebRTC(url string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithDisableWebRTC()(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithDisabledWebRTCBytes 使用共享池禁用 WebRTC API 后截图，并返回图片字节。
func SharedScreenshotWithDisabledWebRTCBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithDisableWebRTC()(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithSpoofedScreen 使用共享池按伪造屏幕尺寸截图。
func SharedScreenshotWithSpoofedScreen(url string, width, height int, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithSpoofedScreen(width, height)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithSpoofedScreenBytes 使用共享池按伪造屏幕尺寸截图，并返回图片字节。
func SharedScreenshotWithSpoofedScreenBytes(url string, width, height int, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithSpoofedScreen(width, height)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithCookieHeader 使用共享池注入 Cookie header 后截图。
func SharedScreenshotWithCookieHeader(url string, header string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieHeader(header)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithCookieHeaderBytes 使用共享池注入 Cookie header 后截图，并返回图片字节。
func SharedScreenshotWithCookieHeaderBytes(url string, header string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieHeader(header)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithCookieStrings 使用共享池注入多个 Cookie header 后截图。
func SharedScreenshotWithCookieStrings(url string, headers []string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieStrings(headers...)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithCookieStringsBytes 使用共享池注入多个 Cookie header 后截图，并返回图片字节。
func SharedScreenshotWithCookieStringsBytes(url string, headers []string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieStrings(headers...)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithCookieFile 使用共享池按持久化 JSON CookieJar 截图。
func SharedScreenshotWithCookieFile(url string, path string, writeBack bool, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieFile(path)(screenshotOpts)
	screenshotOpts.CookieWriteBack = writeBack
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithCookieFileBytes 使用共享池按持久化 JSON CookieJar 截图，并返回图片字节。
func SharedScreenshotWithCookieFileBytes(url string, path string, writeBack bool, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieFile(path)(screenshotOpts)
	screenshotOpts.CookieWriteBack = writeBack
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithCookieImport 使用共享池导入 Netscape/Mozilla Cookie 文件后截图。
func SharedScreenshotWithCookieImport(url string, path string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieImport(path)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithCookieImportBytes 使用共享池导入 Netscape/Mozilla Cookie 文件后截图，并返回图片字节。
func SharedScreenshotWithCookieImportBytes(url string, path string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieImport(path)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithCookieExport 使用共享池截图后导出 Cookie 到 Netscape/Mozilla Cookie 文件。
func SharedScreenshotWithCookieExport(url string, path string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieExport(path)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithCookieExportBytes 使用共享池截图后导出 Cookie 到 Netscape/Mozilla Cookie 文件，并返回图片字节。
func SharedScreenshotWithCookieExportBytes(url string, path string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithCookieExport(path)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithBlacklist 使用共享池按自定义 URL 黑名单规则截图。
func SharedScreenshotWithBlacklist(url string, patterns []string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithBlacklist(patterns...)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithBlacklistBytes 使用共享池按自定义 URL 黑名单规则截图，并返回图片字节。
func SharedScreenshotWithBlacklistBytes(url string, patterns []string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithBlacklist(patterns...)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithBlacklistFile 使用共享池按 URL 黑名单文件截图。
func SharedScreenshotWithBlacklistFile(url string, path string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithBlacklistFile(path)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithBlacklistFileBytes 使用共享池按 URL 黑名单文件截图，并返回图片字节。
func SharedScreenshotWithBlacklistFileBytes(url string, path string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithBlacklistFile(path)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithoutBlacklist 使用共享池禁用 URL 黑名单后截图。
func SharedScreenshotWithoutBlacklist(url string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithNoBlacklist()(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithoutBlacklistBytes 使用共享池禁用 URL 黑名单后截图，并返回图片字节。
func SharedScreenshotWithoutBlacklistBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithNoBlacklist()(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithDefaultBlacklist 使用共享池按内置 URL 黑名单规则截图。
func SharedScreenshotWithDefaultBlacklist(url string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithDefaultBlacklist()(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithDefaultBlacklistBytes 使用共享池按内置 URL 黑名单规则截图，并返回图片字节。
func SharedScreenshotWithDefaultBlacklistBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithDefaultBlacklist()(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithRetries 使用共享池按指定最大重试次数截图。
func SharedScreenshotWithRetries(url string, maxRetries int, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithMaxRetries(maxRetries)(screenshotOpts)
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithRetriesBytes 使用共享池按指定最大重试次数截图，并返回图片字节。
func SharedScreenshotWithRetriesBytes(url string, maxRetries int, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithMaxRetries(maxRetries)(screenshotOpts)
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithActions 使用共享池在截图前执行交互动作序列。
func SharedScreenshotWithActions(url string, actions []runner.InteractionAction, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Actions = actions
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithActionsBytes 使用共享池在截图前执行交互动作序列，并返回图片字节。
func SharedScreenshotWithActionsBytes(url string, actions []runner.InteractionAction, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Actions = actions
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithForm 使用共享池在截图前填写并提交表单。
func SharedScreenshotWithForm(url string, form runner.Form, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Form = form
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithFormBytes 使用共享池在截图前填写并提交表单，并返回图片字节。
func SharedScreenshotWithFormBytes(url string, form runner.Form, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Form = form
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithCookies 使用共享池在截图前注入自定义 Cookie。
func SharedScreenshotWithCookies(url string, cookies []runner.CustomCookie, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Cookies = cookies
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithCookiesBytes 使用共享池在截图前注入自定义 Cookie，并返回图片字节。
func SharedScreenshotWithCookiesBytes(url string, cookies []runner.CustomCookie, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Cookies = cookies
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotElement 使用共享池截取指定 CSS 选择器匹配的元素。
func SharedScreenshotElement(url string, selector string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Selector = selector
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotElementBytes 使用共享池截取指定 CSS 选择器匹配的元素，并返回图片字节。
func SharedScreenshotElementBytes(url string, selector string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Selector = selector
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotXPath 使用共享池截取指定 XPath 匹配的元素。
func SharedScreenshotXPath(url string, xpath string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.XPath = xpath
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotXPathBytes 使用共享池截取指定 XPath 匹配的元素，并返回图片字节。
func SharedScreenshotXPathBytes(url string, xpath string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.XPath = xpath
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotFullPage 使用共享池截取完整页面。
func SharedScreenshotFullPage(url string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.CaptureFullPage = true
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotFullPageBytes 使用共享池截取完整页面，并返回图片字节。
func SharedScreenshotFullPageBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.CaptureFullPage = true
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotDevice 使用共享池按指定设备预设截图。
func SharedScreenshotDevice(url string, device string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Device = device
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotDeviceBytes 使用共享池按指定设备预设截图，并返回图片字节。
func SharedScreenshotDeviceBytes(url string, device string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.Device = device
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotViewport 使用共享池按指定 viewport 截图。
func SharedScreenshotViewport(url string, width, height int, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.WindowWidth = width
	screenshotOpts.WindowHeight = height
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotViewportBytes 使用共享池按指定 viewport 截图，并返回图片字节。
func SharedScreenshotViewportBytes(url string, width, height int, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.WindowWidth = width
	screenshotOpts.WindowHeight = height
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithJS 使用共享池在页面加载后执行 JavaScript 再截图。
func SharedScreenshotWithJS(url string, js string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.JavaScript = js
	screenshotOpts.RunJSAfter = true
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithJSBytes 使用共享池在页面加载后执行 JavaScript 再截图，并返回图片字节。
func SharedScreenshotWithJSBytes(url string, js string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.JavaScript = js
	screenshotOpts.RunJSAfter = true
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithJSBefore 使用共享池在页面加载前执行 JavaScript 再截图。
func SharedScreenshotWithJSBefore(url string, js string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.JavaScript = js
	screenshotOpts.RunJSBefore = true
	screenshotOpts.RunJSAfter = false
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithJSBeforeBytes 使用共享池在页面加载前执行 JavaScript 再截图，并返回图片字节。
func SharedScreenshotWithJSBeforeBytes(url string, js string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.JavaScript = js
	screenshotOpts.RunJSBefore = true
	screenshotOpts.RunJSAfter = false
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotWithJSFile 使用共享池执行 JavaScript 文件后截图。
func SharedScreenshotWithJSFile(url string, jsFile string, beforeLoad bool, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.JavaScriptFile = jsFile
	if beforeLoad {
		screenshotOpts.RunJSBefore = true
		screenshotOpts.RunJSAfter = false
	} else {
		screenshotOpts.RunJSAfter = true
	}
	return SharedScreenshot(url, screenshotOpts)
}

// SharedScreenshotWithJSFileBytes 使用共享池执行 JavaScript 文件后截图，并返回图片字节。
func SharedScreenshotWithJSFileBytes(url string, jsFile string, beforeLoad bool, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	screenshotOpts.JavaScriptFile = jsFile
	if beforeLoad {
		screenshotOpts.RunJSBefore = true
		screenshotOpts.RunJSAfter = false
	} else {
		screenshotOpts.RunJSAfter = true
	}
	return SharedScreenshotBytes(url, screenshotOpts)
}

// SharedScreenshotEvidence 使用共享池截图并收集 HTML、HTTP 头、Cookie、控制台日志和网络请求。
func SharedScreenshotEvidence(url string, opts *ScreenshotOptions) (*models.Result, error) {
	return SharedScreenshotEvidenceWithContext(context.Background(), url, opts)
}

// SharedScreenshotEvidenceWithContext 使用共享池执行可取消的全证据截图。
func SharedScreenshotEvidenceWithContext(ctx context.Context, url string, opts *ScreenshotOptions) (*models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithEvidence()(screenshotOpts)
	return SharedScreenshotWithContext(ctx, url, screenshotOpts)
}

// SharedScreenshotEvidenceBytes 使用共享池截图、收集全部证据，并返回图片字节。
func SharedScreenshotEvidenceBytes(url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	return SharedScreenshotEvidenceBytesWithContext(context.Background(), url, opts)
}

// SharedScreenshotEvidenceBytesWithContext 使用共享池执行可取消的全证据字节截图。
func SharedScreenshotEvidenceBytesWithContext(ctx context.Context, url string, opts *ScreenshotOptions) ([]byte, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithEvidence()(screenshotOpts)
	return SharedScreenshotBytesWithContext(ctx, url, screenshotOpts)
}

// SharedCaptureEvidenceBundle 使用函数式选项通过共享池采集全证据并写入证据包目录。
func SharedCaptureEvidenceBundle(url string, dir string, options ...ScreenshotOption) (*EvidenceBundle, *models.Result, error) {
	return SharedCaptureEvidenceBundleWithContext(context.Background(), url, dir, options...)
}

// SharedCaptureEvidenceBundleWithContext 使用函数式选项通过共享池执行可取消的全证据采集并写入证据包目录。
func SharedCaptureEvidenceBundleWithContext(ctx context.Context, url string, dir string, options ...ScreenshotOption) (*EvidenceBundle, *models.Result, error) {
	return SharedScreenshotEvidenceBundleWithContext(ctx, url, dir, NewScreenshotOptions(options...))
}

// SharedScreenshotEvidenceBundle 使用共享池截图、收集全部证据，并写入证据包目录。
func SharedScreenshotEvidenceBundle(url string, dir string, opts *ScreenshotOptions) (*EvidenceBundle, *models.Result, error) {
	return SharedScreenshotEvidenceBundleWithContext(context.Background(), url, dir, opts)
}

// SharedScreenshotEvidenceBundleWithContext 使用共享池执行可取消的全证据采集和证据包导出。
func SharedScreenshotEvidenceBundleWithContext(ctx context.Context, url string, dir string, opts *ScreenshotOptions) (*EvidenceBundle, *models.Result, error) {
	screenshotOpts := ensureSharedScreenshotOptions(opts)
	WithEvidence()(screenshotOpts)

	_, result, err := SharedScreenshotBytesWithContext(ctx, url, screenshotOpts)
	if err != nil {
		return nil, result, err
	}

	bundle, err := WrapResult(result).SaveEvidenceBundle(dir)
	if err != nil {
		return nil, result, err
	}
	return bundle, result, nil
}

// SharedBatchScreenshot 使用共享池批量截图。
func SharedBatchScreenshot(urls []string, opts *ScreenshotOptions) []BatchResult {
	return SharedBatchScreenshotWithContext(context.Background(), urls, opts)
}

// SharedBatchScreenshotWithContext 使用共享池执行可取消的批量截图。
func SharedBatchScreenshotWithContext(ctx context.Context, urls []string, opts *ScreenshotOptions) []BatchResult {
	results := make([]BatchResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()

			result, err := SharedScreenshotWithContext(ctx, target, opts)
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

// SharedBatchScreenshotBytes 使用共享池批量截图，并返回每个目标的图片字节。
func SharedBatchScreenshotBytes(urls []string, opts *ScreenshotOptions) []BatchBytesResult {
	return SharedBatchScreenshotBytesWithContext(context.Background(), urls, opts)
}

// SharedBatchScreenshotBytesWithContext 使用共享池执行可取消的批量字节截图。
func SharedBatchScreenshotBytesWithContext(ctx context.Context, urls []string, opts *ScreenshotOptions) []BatchBytesResult {
	results := make([]BatchBytesResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()

			data, result, err := SharedScreenshotBytesWithContext(ctx, target, opts)
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

// SharedBatchScreenshotRequests 使用共享池批量截图，每个请求可携带独立配置。
func SharedBatchScreenshotRequests(requests []ScreenshotRequest) []BatchResult {
	return SharedBatchScreenshotRequestsWithContext(context.Background(), requests)
}

// SharedBatchScreenshotRequestsWithContext 使用共享池执行可取消的 per-request 批量截图。
func SharedBatchScreenshotRequestsWithContext(ctx context.Context, requests []ScreenshotRequest) []BatchResult {
	results := make([]BatchResult, len(requests))
	var wg sync.WaitGroup

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req ScreenshotRequest) {
			defer wg.Done()

			result, err := SharedScreenshotWithContext(ctx, req.URL, req.Options)
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

// SharedBatchScreenshotRequestsBytes 使用共享池批量截图，每个请求可携带独立配置，并返回图片字节。
func SharedBatchScreenshotRequestsBytes(requests []ScreenshotRequest) []BatchBytesResult {
	return SharedBatchScreenshotRequestsBytesWithContext(context.Background(), requests)
}

// SharedBatchScreenshotRequestsBytesWithContext 使用共享池执行可取消的 per-request 批量字节截图。
func SharedBatchScreenshotRequestsBytesWithContext(ctx context.Context, requests []ScreenshotRequest) []BatchBytesResult {
	results := make([]BatchBytesResult, len(requests))
	var wg sync.WaitGroup

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req ScreenshotRequest) {
			defer wg.Done()

			data, result, err := SharedScreenshotBytesWithContext(ctx, req.URL, req.Options)
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

// SharedBatchScreenshotEvidenceBundles 使用共享池批量采集全证据，并为每个 URL 写入证据包目录。
func SharedBatchScreenshotEvidenceBundles(urls []string, dir string, opts *ScreenshotOptions) []BatchEvidenceBundleResult {
	return SharedBatchScreenshotEvidenceBundlesWithContext(context.Background(), urls, dir, opts)
}

// SharedBatchScreenshotEvidenceBundlesWithContext 使用共享池执行可取消的批量证据包采集。
func SharedBatchScreenshotEvidenceBundlesWithContext(ctx context.Context, urls []string, dir string, opts *ScreenshotOptions) []BatchEvidenceBundleResult {
	results := make([]BatchEvidenceBundleResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()

			bundleDir := batchEvidenceBundleDir(dir, idx, "", target)
			bundle, result, err := SharedScreenshotEvidenceBundleWithContext(ctx, target, bundleDir, opts)
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

// SharedBatchScreenshotRequestsEvidenceBundles 使用共享池按请求独立配置批量采集证据包。
func SharedBatchScreenshotRequestsEvidenceBundles(requests []ScreenshotRequest, dir string) []BatchEvidenceBundleResult {
	return SharedBatchScreenshotRequestsEvidenceBundlesWithContext(context.Background(), requests, dir)
}

// SharedBatchScreenshotRequestsEvidenceBundlesWithContext 使用共享池执行可取消的 per-request 证据包采集。
func SharedBatchScreenshotRequestsEvidenceBundlesWithContext(ctx context.Context, requests []ScreenshotRequest, dir string) []BatchEvidenceBundleResult {
	results := make([]BatchEvidenceBundleResult, len(requests))
	var wg sync.WaitGroup

	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req ScreenshotRequest) {
			defer wg.Done()

			bundleDir := batchEvidenceBundleDir(dir, idx, req.Name, req.URL)
			bundle, result, err := SharedScreenshotEvidenceBundleWithContext(ctx, req.URL, bundleDir, req.Options)
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

// SharedBatchScreenshotTargets 展开裸 host/IP 目标后使用共享池批量截图。
func SharedBatchScreenshotTargets(targets []string, opts *ScreenshotOptions) []BatchResult {
	return SharedBatchScreenshotTargetsWithContext(context.Background(), targets, opts)
}

// SharedBatchScreenshotTargetsWithContext 展开裸 host/IP 目标后使用共享池执行可取消的批量截图。
func SharedBatchScreenshotTargetsWithContext(ctx context.Context, targets []string, opts *ScreenshotOptions) []BatchResult {
	expanded := ExpandTargets(targets, opts)
	return SharedBatchScreenshotWithContext(ctx, expanded, opts)
}

// SharedBatchScreenshotTargetsBytes 展开裸 host/IP 目标后使用共享池批量截图，并返回图片字节。
func SharedBatchScreenshotTargetsBytes(targets []string, opts *ScreenshotOptions) []BatchBytesResult {
	return SharedBatchScreenshotTargetsBytesWithContext(context.Background(), targets, opts)
}

// SharedBatchScreenshotTargetsBytesWithContext 展开裸 host/IP 目标后使用共享池执行可取消的批量字节截图。
func SharedBatchScreenshotTargetsBytesWithContext(ctx context.Context, targets []string, opts *ScreenshotOptions) []BatchBytesResult {
	expanded := ExpandTargets(targets, opts)
	return SharedBatchScreenshotBytesWithContext(ctx, expanded, opts)
}

// SharedBatchScreenshotTargetsEvidenceBundles 展开裸 host/IP 目标后使用共享池批量采集证据包。
func SharedBatchScreenshotTargetsEvidenceBundles(targets []string, dir string, opts *ScreenshotOptions) []BatchEvidenceBundleResult {
	return SharedBatchScreenshotTargetsEvidenceBundlesWithContext(context.Background(), targets, dir, opts)
}

// SharedBatchScreenshotTargetsEvidenceBundlesWithContext 展开裸 host/IP 目标后使用共享池执行可取消的批量证据包采集。
func SharedBatchScreenshotTargetsEvidenceBundlesWithContext(ctx context.Context, targets []string, dir string, opts *ScreenshotOptions) []BatchEvidenceBundleResult {
	expanded := ExpandTargets(targets, opts)
	return SharedBatchScreenshotEvidenceBundlesWithContext(ctx, expanded, dir, opts)
}

// SharedBatchScreenshotStreaming 使用共享池流式返回批量截图结果。
func SharedBatchScreenshotStreaming(ctx context.Context, urls []string, opts *ScreenshotOptions) <-chan BatchResult {
	ch := make(chan BatchResult, len(urls))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for _, url := range urls {
			select {
			case <-ctx.Done():
				ch <- BatchResult{URL: url, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(target string) {
				defer wg.Done()

				result, err := SharedScreenshotWithContext(ctx, target, opts)
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

// SharedBatchScreenshotBytesStreaming 使用共享池流式返回批量字节截图结果。
func SharedBatchScreenshotBytesStreaming(ctx context.Context, urls []string, opts *ScreenshotOptions) <-chan BatchBytesResult {
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

				data, result, err := SharedScreenshotBytesWithContext(ctx, target, opts)
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

// SharedBatchScreenshotEvidenceBundlesStreaming 使用共享池流式返回批量证据包结果。
func SharedBatchScreenshotEvidenceBundlesStreaming(ctx context.Context, urls []string, dir string, opts *ScreenshotOptions) <-chan BatchEvidenceBundleResult {
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

				bundle, result, err := SharedScreenshotEvidenceBundleWithContext(ctx, target, targetDir, opts)
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

// SharedBatchScreenshotRequestsStreaming 使用共享池流式返回 per-request 批量截图结果。
func SharedBatchScreenshotRequestsStreaming(ctx context.Context, requests []ScreenshotRequest) <-chan BatchResult {
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

				result, err := SharedScreenshotWithContext(ctx, req.URL, req.Options)
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

// SharedBatchScreenshotRequestsBytesStreaming 使用共享池流式返回 per-request 批量字节截图结果。
func SharedBatchScreenshotRequestsBytesStreaming(ctx context.Context, requests []ScreenshotRequest) <-chan BatchBytesResult {
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

				data, result, err := SharedScreenshotBytesWithContext(ctx, req.URL, req.Options)
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

// SharedBatchScreenshotRequestsEvidenceBundlesStreaming 使用共享池流式返回 per-request 批量证据包结果。
func SharedBatchScreenshotRequestsEvidenceBundlesStreaming(ctx context.Context, requests []ScreenshotRequest, dir string) <-chan BatchEvidenceBundleResult {
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

				bundle, result, err := SharedScreenshotEvidenceBundleWithContext(ctx, req.URL, targetDir, req.Options)
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

// SharedBatchScreenshotTargetsStreaming 展开裸 host/IP 目标后使用共享池流式返回截图结果。
func SharedBatchScreenshotTargetsStreaming(ctx context.Context, targets []string, opts *ScreenshotOptions) <-chan BatchResult {
	expanded := ExpandTargets(targets, opts)
	return SharedBatchScreenshotStreaming(ctx, expanded, opts)
}

// SharedBatchScreenshotTargetsBytesStreaming 展开裸 host/IP 目标后使用共享池流式返回字节截图结果。
func SharedBatchScreenshotTargetsBytesStreaming(ctx context.Context, targets []string, opts *ScreenshotOptions) <-chan BatchBytesResult {
	expanded := ExpandTargets(targets, opts)
	return SharedBatchScreenshotBytesStreaming(ctx, expanded, opts)
}

// SharedBatchScreenshotTargetsEvidenceBundlesStreaming 展开裸 host/IP 目标后使用共享池流式返回证据包结果。
func SharedBatchScreenshotTargetsEvidenceBundlesStreaming(ctx context.Context, targets []string, dir string, opts *ScreenshotOptions) <-chan BatchEvidenceBundleResult {
	expanded := ExpandTargets(targets, opts)
	return SharedBatchScreenshotEvidenceBundlesStreaming(ctx, expanded, dir, opts)
}

// SharedBatchScreenshotCallback 使用共享池批量截图，每完成一个调用 callback。
func SharedBatchScreenshotCallback(ctx context.Context, urls []string, opts *ScreenshotOptions, callback func(BatchResult)) {
	ch := SharedBatchScreenshotStreaming(ctx, urls, opts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotBytesCallback 使用共享池批量字节截图，每完成一个调用 callback。
func SharedBatchScreenshotBytesCallback(ctx context.Context, urls []string, opts *ScreenshotOptions, callback func(BatchBytesResult)) {
	ch := SharedBatchScreenshotBytesStreaming(ctx, urls, opts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotEvidenceBundlesCallback 使用共享池批量采集证据包，每完成一个调用 callback。
func SharedBatchScreenshotEvidenceBundlesCallback(ctx context.Context, urls []string, dir string, opts *ScreenshotOptions, callback func(BatchEvidenceBundleResult)) {
	ch := SharedBatchScreenshotEvidenceBundlesStreaming(ctx, urls, dir, opts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotRequestsCallback 使用共享池 per-request 批量截图，每完成一个调用 callback。
func SharedBatchScreenshotRequestsCallback(ctx context.Context, requests []ScreenshotRequest, callback func(BatchResult)) {
	ch := SharedBatchScreenshotRequestsStreaming(ctx, requests)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotRequestsBytesCallback 使用共享池 per-request 批量字节截图，每完成一个调用 callback。
func SharedBatchScreenshotRequestsBytesCallback(ctx context.Context, requests []ScreenshotRequest, callback func(BatchBytesResult)) {
	ch := SharedBatchScreenshotRequestsBytesStreaming(ctx, requests)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotRequestsEvidenceBundlesCallback 使用共享池 per-request 批量采集证据包，每完成一个调用 callback。
func SharedBatchScreenshotRequestsEvidenceBundlesCallback(ctx context.Context, requests []ScreenshotRequest, dir string, callback func(BatchEvidenceBundleResult)) {
	ch := SharedBatchScreenshotRequestsEvidenceBundlesStreaming(ctx, requests, dir)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotTargetsCallback 展开裸 host/IP 目标后使用共享池批量截图，每完成一个调用 callback。
func SharedBatchScreenshotTargetsCallback(ctx context.Context, targets []string, opts *ScreenshotOptions, callback func(BatchResult)) {
	ch := SharedBatchScreenshotTargetsStreaming(ctx, targets, opts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotTargetsBytesCallback 展开裸 host/IP 目标后使用共享池批量字节截图，每完成一个调用 callback。
func SharedBatchScreenshotTargetsBytesCallback(ctx context.Context, targets []string, opts *ScreenshotOptions, callback func(BatchBytesResult)) {
	ch := SharedBatchScreenshotTargetsBytesStreaming(ctx, targets, opts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedBatchScreenshotTargetsEvidenceBundlesCallback 展开裸 host/IP 目标后使用共享池批量采集证据包，每完成一个调用 callback。
func SharedBatchScreenshotTargetsEvidenceBundlesCallback(ctx context.Context, targets []string, dir string, opts *ScreenshotOptions, callback func(BatchEvidenceBundleResult)) {
	ch := SharedBatchScreenshotTargetsEvidenceBundlesStreaming(ctx, targets, dir, opts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// SharedSetIdleTimeout 设置共享池的空闲超时
func SharedSetIdleTimeout(timeout time.Duration) {
	if err := sharedSetIdleTimeout(timeout); err != nil {
		log.Error("设置共享池空闲超时失败", "error", err)
	}
}

// SharedStats 返回共享池统计信息
func SharedStats() (runner.PoolStats, error) {
	return sharedPoolStats()
}

// CloseSharedPool 关闭共享池
// 通常在程序退出时调用（可用 defer）
func CloseSharedPool() {
	closeSharedPool()
}

// defaultRunnerOptions 返回默认 runner 配置
func defaultRunnerOptions() runner.Options {
	opts := runner.Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = "screenshots"
	opts.Scan.ScreenshotFormat = "png"
	return opts
}

func ensureSharedScreenshotOptions(opts *ScreenshotOptions) *ScreenshotOptions {
	if opts != nil {
		return opts
	}
	return &ScreenshotOptions{}
}

func sharedRunnerOptionsForScreenshot(target string, opts *ScreenshotOptions) runner.Options {
	runnerOpts := defaultRunnerOptions()
	runnerOpts = mergeWithScreenshotOptions(runnerOpts, opts)
	appendCookieSources(target, &runnerOpts)
	return runnerOpts
}

func sharedHandleResultCookies(target string, result *models.Result, opts *runner.Options) {
	if result == nil || len(result.Cookies) == 0 || opts.Scan.CookieExport == "" {
		return
	}
	if err := runner.ExportResultCookiesToNetscape(opts.Scan.CookieExport, result.Cookies, target); err != nil {
		log.Warn("SDK: 导出 Netscape Cookie 失败", "file", opts.Scan.CookieExport, "error", err)
	}
}
