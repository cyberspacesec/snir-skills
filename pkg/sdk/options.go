package sdk

import (
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// ClientOptions SDK 客户端配置
// 控制浏览器行为、截图参数、数据收集等
type ClientOptions struct {
	// Chrome 浏览器配置
	ChromePath       string // Chrome 可执行文件路径
	Headless         bool   // 无头模式（默认 true）
	WindowWidth      int    // 窗口宽度（默认 1280）
	WindowHeight     int    // 窗口高度（默认 800）
	UserAgent        string // 自定义 User-Agent
	Proxy            string // 代理服务器地址
	Device           string // 设备预设名称
	WSSURL           string // 远程 Chrome WebSocket URL
	IgnoreCertErrors bool   // 忽略证书错误

	// 浏览器指纹配置（反检测）
	AcceptLanguage  string            // Accept-Language 头（如 "zh-CN,zh;q=0.9"）
	Platform        string            // 平台标识（如 "Win32"）
	Vendor          string            // 浏览器厂商（如 "Google Inc."）
	Plugins         []string          // 浏览器插件列表
	WebGLVendor     string            // WebGL 厂商（如 "Intel Inc."）
	WebGLRenderer   string            // WebGL 渲染器（如 "Intel Iris"）
	CustomHeaders   map[string]string // 自定义 HTTP 头
	DisableWebRTC   bool              // 禁用 WebRTC
	SpoofScreenSize bool              // 伪造屏幕尺寸
	ScreenWidth     int               // 伪造屏幕宽度
	ScreenHeight    int               // 伪造屏幕高度

	// 截图配置
	MaxConcurrent     int    // 最大并发截图数（默认 2）
	ScreenshotPath    string // 截图保存路径
	ScreenshotFormat  string // 截图格式 png/jpeg
	ScreenshotQuality int    // JPEG 截图质量（1-100，默认 90）
	SkipSave          bool   // 跳过保存到磁盘
	CaptureFullPage   bool   // 全页截图（含滚动区域）
	Selector          string // CSS 选择器截图
	XPath             string // XPath 截图

	// 超时配置
	Timeout time.Duration // 页面加载超时
	Delay   time.Duration // 截图前等待

	// JavaScript 执行
	JavaScript     string // 在页面上执行的 JavaScript
	JavaScriptFile string // JavaScript 文件路径
	RunJSBefore    bool   // 在页面加载前执行 JS

	// 数据收集
	SaveHTML    bool // 保存 HTML 源码
	SaveHeaders bool // 保存 HTTP 头
	SaveConsole bool // 保存控制台日志
	SaveCookies bool // 保存 Cookie
	SaveNetwork bool // 保存网络请求日志

	// 重试配置
	MaxRetries int // 最大重试次数（默认 1）

	// 自定义 Cookie
	Cookies []runner.CustomCookie // 注入自定义 Cookie

	// 浏览器交互
	Actions []runner.InteractionAction // 交互动作序列
	Form    runner.Form                // 表单填写配置

	// 黑名单
	EnableBlacklist   bool     // 启用 URL 黑名单
	DefaultBlacklist  bool     // 使用默认黑名单
	BlacklistPatterns []string // 自定义黑名单规则
	BlacklistFile     string   // 黑名单文件路径

	// Cookie 持久化
	CookieFile string // Cookie 持久化文件路径 (JSON 格式)
}

// DefaultClientOptions 返回默认客户端配置
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Headless:          true,
		WindowWidth:       1280,
		WindowHeight:      800,
		Timeout:           30 * time.Second,
		MaxConcurrent:     2,
		ScreenshotPath:    "screenshots",
		ScreenshotFormat:  "png",
		ScreenshotQuality: 90,
		SkipSave:          false,
		IgnoreCertErrors:  false,
		CaptureFullPage:   false,
		MaxRetries:        1,
		EnableBlacklist:   true,
		DefaultBlacklist:  true,
	}
}

// ScreenshotOptions 单次截图的覆盖配置
// 非零值会覆盖 ClientOptions 中的对应配置
type ScreenshotOptions struct {
	// 超时覆盖
	Timeout time.Duration // 页面加载超时（覆盖 ClientOptions）
	Delay   time.Duration // 截图前等待（覆盖 ClientOptions）

	// 浏览器覆盖
	UserAgent string // User-Agent（覆盖 ClientOptions）
	Proxy     string // 代理（覆盖 ClientOptions）
	Device    string // 设备预设名称（覆盖 ClientOptions）

	// 截图覆盖
	Selector          string // CSS 选择器
	XPath             string // XPath
	CaptureFullPage   bool   // 全页截图
	ScreenshotFormat  string // 截图格式 png/jpeg
	ScreenshotQuality int    // JPEG 质量

	// JavaScript
	JavaScript     string // 在页面上执行的 JavaScript
	JavaScriptFile string // JavaScript 文件路径
	RunJSBefore    bool   // 在页面加载前执行 JS

	// 数据收集覆盖
	SaveHTML    bool // 保存 HTML
	SaveHeaders bool // 保存 HTTP 头
	SaveConsole bool // 保存控制台
	SaveCookies bool // 保存 Cookie
	SaveNetwork bool // 保存网络请求
	SkipSave    bool // 跳过保存

	// 自定义 Cookie（注入）
	Cookies []runner.CustomCookie

	// 浏览器交互
	Actions []runner.InteractionAction // 交互动作序列
	Form    runner.Form                // 表单填写

	// 重试覆盖
	MaxRetries int // 最大重试次数
}

// toRunnerOptions 将 ClientOptions 转换为 runner.Options
func toRunnerOptions(co ClientOptions) runner.Options {
	opts := runner.Options{}

	// Chrome 配置
	opts.Chrome.Path = co.ChromePath
	opts.Chrome.Headless = co.Headless
	opts.Chrome.WindowX = co.WindowWidth
	opts.Chrome.WindowY = co.WindowHeight
	opts.Chrome.UserAgent = co.UserAgent
	opts.Chrome.Proxy = co.Proxy
	opts.Chrome.IgnoreCertErrors = co.IgnoreCertErrors
	opts.Chrome.WSS = co.WSSURL
	opts.Chrome.Timeout = int(co.Timeout.Seconds())
	opts.Chrome.Delay = int(co.Delay.Seconds())

	applyDevicePreset(co.Device, &opts)

	// 浏览器指纹：只用非零字段覆盖，避免空值清掉设备预设。
	if co.UserAgent != "" {
		opts.Chrome.UserAgent = co.UserAgent
	}
	if co.AcceptLanguage != "" {
		opts.Chrome.AcceptLanguage = co.AcceptLanguage
	}
	if co.Platform != "" {
		opts.Chrome.Platform = co.Platform
	}
	if co.Vendor != "" {
		opts.Chrome.Vendor = co.Vendor
	}
	if len(co.Plugins) > 0 {
		opts.Chrome.Plugins = co.Plugins
	}
	if co.WebGLVendor != "" {
		opts.Chrome.WebGLVendor = co.WebGLVendor
	}
	if co.WebGLRenderer != "" {
		opts.Chrome.WebGLRenderer = co.WebGLRenderer
	}
	if len(co.CustomHeaders) > 0 {
		opts.Chrome.CustomHeaders = co.CustomHeaders
	}
	if co.DisableWebRTC {
		opts.Chrome.DisableWebRTC = true
	}
	if co.SpoofScreenSize {
		opts.Chrome.SpoofScreenSize = true
	}
	if co.ScreenWidth > 0 {
		opts.Chrome.ScreenWidth = co.ScreenWidth
	}
	if co.ScreenHeight > 0 {
		opts.Chrome.ScreenHeight = co.ScreenHeight
	}

	// Scan 配置
	opts.Scan.ScreenshotPath = co.ScreenshotPath
	opts.Scan.ScreenshotFormat = co.ScreenshotFormat
	opts.Scan.ScreenshotQuality = co.ScreenshotQuality
	opts.Scan.ScreenshotSkipSave = co.SkipSave
	opts.Scan.Selector = co.Selector
	opts.Scan.XPath = co.XPath
	opts.Scan.CaptureFullPage = co.CaptureFullPage
	opts.Scan.HTTP = true
	opts.Scan.HTTPS = true
	opts.Scan.MaxRetries = co.MaxRetries

	// JavaScript
	opts.Scan.JavaScript = co.JavaScript
	opts.Scan.JavaScriptFile = co.JavaScriptFile
	opts.Scan.RunJSBefore = co.RunJSBefore
	if co.JavaScript != "" {
		opts.Scan.RunJSAfter = true
	}

	// 数据收集
	opts.Scan.SaveHTML = co.SaveHTML
	opts.Scan.SaveHeaders = co.SaveHeaders
	opts.Scan.SaveConsole = co.SaveConsole
	opts.Scan.SaveCookies = co.SaveCookies
	opts.Scan.SaveNetwork = co.SaveNetwork

	// Cookie
	opts.Scan.Cookies = co.Cookies
	opts.Scan.CookiesFile = co.CookieFile

	// 交互
	opts.Scan.Actions = co.Actions
	opts.Scan.Form = co.Form

	// 黑名单
	opts.Scan.EnableBlacklist = co.EnableBlacklist
	opts.Scan.DefaultBlacklist = co.DefaultBlacklist
	opts.Scan.BlacklistPatterns = co.BlacklistPatterns
	opts.Scan.BlacklistFile = co.BlacklistFile

	return opts
}

// mergeWithScreenshotOptions 用 ScreenshotOptions 覆盖 runner.Options
func mergeWithScreenshotOptions(base runner.Options, so *ScreenshotOptions) runner.Options {
	if so == nil {
		return base
	}

	// 超时覆盖
	if so.Timeout > 0 {
		base.Chrome.Timeout = int(so.Timeout.Seconds())
	}
	if so.Delay > 0 {
		base.Chrome.Delay = int(so.Delay.Seconds())
	}

	// 浏览器覆盖
	if so.UserAgent != "" {
		base.Chrome.UserAgent = so.UserAgent
	}
	if so.Proxy != "" {
		base.Chrome.Proxy = so.Proxy
	}
	if so.Device != "" {
		applyDevicePreset(so.Device, &base)
	}

	// 截图覆盖
	if so.Selector != "" {
		base.Scan.Selector = so.Selector
	}
	if so.XPath != "" {
		base.Scan.XPath = so.XPath
	}
	if so.CaptureFullPage {
		base.Scan.CaptureFullPage = true
	}
	if so.ScreenshotFormat != "" {
		base.Scan.ScreenshotFormat = so.ScreenshotFormat
	}
	if so.ScreenshotQuality > 0 {
		base.Scan.ScreenshotQuality = so.ScreenshotQuality
	}

	// JavaScript 覆盖
	if so.JavaScript != "" {
		base.Scan.JavaScript = so.JavaScript
		base.Scan.RunJSAfter = true
	}
	if so.JavaScriptFile != "" {
		base.Scan.JavaScriptFile = so.JavaScriptFile
	}
	if so.RunJSBefore {
		base.Scan.RunJSBefore = true
	}

	// 数据收集覆盖
	if so.SaveHTML {
		base.Scan.SaveHTML = true
	}
	if so.SaveHeaders {
		base.Scan.SaveHeaders = true
	}
	if so.SaveConsole {
		base.Scan.SaveConsole = true
	}
	if so.SaveCookies {
		base.Scan.SaveCookies = true
	}
	if so.SaveNetwork {
		base.Scan.SaveNetwork = true
	}
	if so.SkipSave {
		base.Scan.ScreenshotSkipSave = true
	}

	// Cookie（追加而非替换）
	if len(so.Cookies) > 0 {
		base.Scan.Cookies = append(base.Scan.Cookies, so.Cookies...)
	}

	// 交互
	if len(so.Actions) > 0 {
		base.Scan.Actions = so.Actions
	}
	if so.Form.Fields != nil {
		base.Scan.Form = so.Form
	}

	// 重试
	if so.MaxRetries > 0 {
		base.Scan.MaxRetries = so.MaxRetries
	}

	return base
}

func applyDevicePreset(device string, opts *runner.Options) {
	if device == "" {
		return
	}
	preset, err := runner.GetDevicePreset(device)
	if err != nil {
		return
	}
	preset.ApplyToOptions(opts)
}
