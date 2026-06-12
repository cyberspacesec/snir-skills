package sdk

import (
	"time"

	"github.com/cyberspacesec/go-snir/pkg/runner"
)

// ClientOptions SDK 客户端配置选项
// 外部项目通过设置这些选项来初始化截图客户端
type ClientOptions struct {
	// Chrome 浏览器相关配置
	ChromePath    string        // Chrome 可执行文件路径（留空则自动查找）
	Headless      bool          // 是否使用无头模式（默认 true）
	WindowWidth   int           // 窗口宽度（默认 1280）
	WindowHeight  int           // 窗口高度（默认 800）
	Timeout       time.Duration // 页面加载超时（默认 30s）
	Delay         time.Duration // 截图前等待时间
	UserAgent     string        // 自定义 User-Agent
	Proxy         string        // 代理服务器地址

	// 连接池配置
	MaxConcurrent int // 最大并发截图数（默认 2）

	// 截图输出配置
	ScreenshotPath   string // 截图保存目录（默认 "screenshots"）
	ScreenshotFormat string // 截图格式: "png" 或 "jpeg"（默认 "png"）
	SkipSave         bool   // 是否跳过保存截图到磁盘（仅返回内存数据）

	// 高级选项
	IgnoreCertErrors bool  // 忽略证书错误
	CaptureFullPage  bool  // 截取完整页面
	Selector         string // CSS 选择器截图
	XPath            string // XPath 截图
}

// DefaultClientOptions 返回默认的客户端配置
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Headless:         true,
		WindowWidth:      1280,
		WindowHeight:     800,
		Timeout:          30 * time.Second,
		MaxConcurrent:    2,
		ScreenshotPath:   "screenshots",
		ScreenshotFormat: "png",
		SkipSave:         false,
		IgnoreCertErrors: false,
		CaptureFullPage:  false,
	}
}

// ScreenshotOptions 单次截图的可选配置
// 可以覆盖 ClientOptions 中的部分设置
type ScreenshotOptions struct {
	Timeout         time.Duration // 本次截图超时
	Delay           time.Duration // 本次截图延迟
	UserAgent       string        // 本次截图 User-Agent
	Selector        string        // 本次截图 CSS 选择器
	XPath           string        // 本次截图 XPath
	CaptureFullPage bool          // 本次截图是否全页
	JavaScript      string        // 要执行的 JavaScript
	SkipSave        bool          // 本次截图是否跳过保存到磁盘
}

// toRunnerOptions 将 ClientOptions 转换为 runner.Options
func toRunnerOptions(co ClientOptions) runner.Options {
	opts := runner.Options{}
	opts.Chrome.Path = co.ChromePath
	opts.Chrome.Headless = co.Headless
	opts.Chrome.WindowX = co.WindowWidth
	opts.Chrome.WindowY = co.WindowHeight
	opts.Chrome.Timeout = int(co.Timeout.Seconds())
	opts.Chrome.Delay = int(co.Delay.Seconds())
	opts.Chrome.UserAgent = co.UserAgent
	opts.Chrome.Proxy = co.Proxy
	opts.Chrome.IgnoreCertErrors = co.IgnoreCertErrors
	opts.Scan.ScreenshotPath = co.ScreenshotPath
	opts.Scan.ScreenshotFormat = co.ScreenshotFormat
	opts.Scan.ScreenshotSkipSave = co.SkipSave
	opts.Scan.Selector = co.Selector
	opts.Scan.XPath = co.XPath
	opts.Scan.CaptureFullPage = co.CaptureFullPage
	opts.Scan.HTTP = true
	opts.Scan.HTTPS = true
	return opts
}

// mergeWithScreenshotOptions 将 ScreenshotOptions 合并到 runner.Options
func mergeWithScreenshotOptions(base runner.Options, so *ScreenshotOptions) runner.Options {
	if so == nil {
		return base
	}

	if so.Timeout > 0 {
		base.Chrome.Timeout = int(so.Timeout.Seconds())
	}
	if so.Delay > 0 {
		base.Chrome.Delay = int(so.Delay.Seconds())
	}
	if so.UserAgent != "" {
		base.Chrome.UserAgent = so.UserAgent
	}
	if so.Selector != "" {
		base.Scan.Selector = so.Selector
	}
	if so.XPath != "" {
		base.Scan.XPath = so.XPath
	}
	if so.CaptureFullPage {
		base.Scan.CaptureFullPage = true
	}
	if so.JavaScript != "" {
		base.Scan.JavaScript = so.JavaScript
		base.Scan.RunJSAfter = true
	}
	if so.SkipSave {
		base.Scan.ScreenshotSkipSave = true
	}

	return base
}