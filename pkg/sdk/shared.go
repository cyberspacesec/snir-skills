package sdk

import (
	"context"
	"fmt"
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
