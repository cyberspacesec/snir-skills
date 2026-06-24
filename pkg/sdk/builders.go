package sdk

import (
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// ScreenshotOption mutates ScreenshotOptions for fluent per-request setup.
type ScreenshotOption func(*ScreenshotOptions)

// NewScreenshotOptions builds ScreenshotOptions from functional options.
//
// Example:
//
//	opts := sdk.NewScreenshotOptions(
//	    sdk.WithFullPage(),
//	    sdk.WithEvidence(),
//	    sdk.WithDevice("iphone-15"),
//	)
func NewScreenshotOptions(options ...ScreenshotOption) *ScreenshotOptions {
	opts := &ScreenshotOptions{}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}
	return opts
}

// CloneScreenshotOptions returns a copy of opts, or a new empty options value.
func CloneScreenshotOptions(opts *ScreenshotOptions) *ScreenshotOptions {
	if opts == nil {
		return &ScreenshotOptions{}
	}
	cloned := *opts
	if opts.Plugins != nil {
		cloned.Plugins = append([]string(nil), opts.Plugins...)
	}
	if opts.ProxyList != nil {
		cloned.ProxyList = append([]string(nil), opts.ProxyList...)
	}
	if opts.Ports != nil {
		cloned.Ports = append([]int(nil), opts.Ports...)
	}
	if opts.CustomHeaders != nil {
		cloned.CustomHeaders = make(map[string]string, len(opts.CustomHeaders))
		for name, value := range opts.CustomHeaders {
			cloned.CustomHeaders[name] = value
		}
	}
	if opts.Cookies != nil {
		cloned.Cookies = append([]runner.CustomCookie(nil), opts.Cookies...)
	}
	if opts.CookieStrings != nil {
		cloned.CookieStrings = append([]string(nil), opts.CookieStrings...)
	}
	if opts.Actions != nil {
		cloned.Actions = append([]runner.InteractionAction(nil), opts.Actions...)
	}
	if opts.BlacklistPatterns != nil {
		cloned.BlacklistPatterns = append(make([]string, 0, len(opts.BlacklistPatterns)), opts.BlacklistPatterns...)
	}
	if opts.IsMobile != nil {
		cloned.IsMobile = boolPtr(*opts.IsMobile)
	}
	if opts.HasTouch != nil {
		cloned.HasTouch = boolPtr(*opts.HasTouch)
	}
	if opts.EnableBlacklist != nil {
		cloned.EnableBlacklist = boolPtr(*opts.EnableBlacklist)
	}
	if opts.DefaultBlacklist != nil {
		cloned.DefaultBlacklist = boolPtr(*opts.DefaultBlacklist)
	}
	return &cloned
}

func boolPtr(value bool) *bool {
	return &value
}

// WithTimeout sets the page load timeout for this screenshot.
func WithTimeout(timeout time.Duration) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Timeout = timeout
	}
}

// WithDelay waits before taking the screenshot.
func WithDelay(delay time.Duration) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Delay = delay
	}
}

// WithViewport overrides the browser viewport for this screenshot.
func WithViewport(width, height int) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.WindowWidth = width
		opts.WindowHeight = height
	}
}

// WithUserAgent overrides the browser User-Agent.
func WithUserAgent(userAgent string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.UserAgent = userAgent
	}
}

// WithProxy routes this screenshot through proxy.
func WithProxy(proxy string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Proxy = proxy
		opts.ProxyList = nil
		opts.ProxyFile = ""
		opts.ProxyURL = ""
	}
}

// WithProxyList rotates this screenshot through the provided proxy list.
func WithProxyList(strategy runner.ProxyStrategy, proxies ...string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Proxy = ""
		opts.ProxyList = proxies
		opts.ProxyFile = ""
		opts.ProxyURL = ""
		opts.ProxyStrategy = strategy
	}
}

// WithProxyFile loads rotating proxies from a file.
func WithProxyFile(path string, strategy runner.ProxyStrategy) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Proxy = ""
		opts.ProxyList = nil
		opts.ProxyFile = path
		opts.ProxyURL = ""
		opts.ProxyStrategy = strategy
	}
}

// WithProxyURL gets proxies from a dynamic proxy API.
func WithProxyURL(url string, strategy runner.ProxyStrategy) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Proxy = ""
		opts.ProxyList = nil
		opts.ProxyFile = ""
		opts.ProxyURL = url
		opts.ProxyStrategy = strategy
	}
}

// WithProxyStrategy sets the proxy rotation strategy.
func WithProxyStrategy(strategy runner.ProxyStrategy) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.ProxyStrategy = strategy
	}
}

// WithDevice applies a named device preset.
func WithDevice(device string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Device = device
	}
}

// WithDeviceEmulation sets viewport, device scale factor, and touch/mobile emulation.
func WithDeviceEmulation(width, height int, scaleFactor float64, isMobile, hasTouch bool) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.WindowWidth = width
		opts.WindowHeight = height
		opts.DeviceScaleFactor = scaleFactor
		opts.IsMobile = boolPtr(isMobile)
		opts.HasTouch = boolPtr(hasTouch)
	}
}

// WithMobileEmulation enables mobile and touch emulation, optionally with a device scale factor.
func WithMobileEmulation(scaleFactor float64) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		if scaleFactor > 0 {
			opts.DeviceScaleFactor = scaleFactor
		}
		opts.IsMobile = boolPtr(true)
		opts.HasTouch = boolPtr(true)
	}
}

// WithTouchEmulation enables or disables touch emulation for this screenshot.
func WithTouchEmulation(enabled bool) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.HasTouch = boolPtr(enabled)
	}
}

// WithIgnoreCertErrors ignores certificate errors for this screenshot.
func WithIgnoreCertErrors() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.IgnoreCertErrors = true
	}
}

// WithFullPage captures the full scrollable page.
func WithFullPage() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.CaptureFullPage = true
	}
}

// WithElement captures the element matching selector.
func WithElement(selector string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Selector = selector
	}
}

// WithXPath captures the element matching xpath.
func WithXPath(xpath string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.XPath = xpath
	}
}

// WithFormat sets screenshot format and optional JPEG quality.
func WithFormat(format string, quality int) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.ScreenshotFormat = format
		opts.ScreenshotQuality = quality
	}
}

// WithPorts sets the scan ports for target expansion workflows.
func WithPorts(ports ...int) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Ports = ports
	}
}

// WithSkipSave keeps screenshot output in memory when the selected API supports it.
func WithSkipSave() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SkipSave = true
	}
}

// WithJS runs JavaScript after page load.
func WithJS(js string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.JavaScript = js
		opts.RunJSBefore = false
		opts.RunJSAfter = true
	}
}

// WithJSBefore runs JavaScript before page load.
func WithJSBefore(js string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.JavaScript = js
		opts.RunJSBefore = true
		opts.RunJSAfter = false
	}
}

// WithJSAfter runs JavaScript after page load.
func WithJSAfter(js string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.JavaScript = js
		opts.RunJSBefore = false
		opts.RunJSAfter = true
	}
}

// WithJSFile runs JavaScript from a file.
func WithJSFile(path string, beforeLoad bool) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.JavaScriptFile = path
		if beforeLoad {
			opts.RunJSBefore = true
			opts.RunJSAfter = false
		} else {
			opts.RunJSBefore = false
			opts.RunJSAfter = true
		}
	}
}

// WithHTML collects HTML source.
func WithHTML() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SaveHTML = true
	}
}

// WithHeaders collects response headers.
func WithHeaders() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SaveHeaders = true
	}
}

// WithConsole collects browser console logs.
func WithConsole() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SaveConsole = true
	}
}

// WithCookies collects browser cookies.
func WithCookies() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SaveCookies = true
	}
}

// WithNetwork collects network request logs.
func WithNetwork() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SaveNetwork = true
	}
}

// WithEvidence collects all supported page evidence.
func WithEvidence() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SaveHTML = true
		opts.SaveHeaders = true
		opts.SaveConsole = true
		opts.SaveCookies = true
		opts.SaveNetwork = true
	}
}

// WithCustomHeaders applies additional request headers.
func WithCustomHeaders(headers map[string]string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.CustomHeaders = headers
	}
}

// WithAcceptLanguage overrides Accept-Language.
func WithAcceptLanguage(language string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.AcceptLanguage = language
	}
}

// WithFingerprint applies common browser fingerprint overrides.
func WithFingerprint(platform, vendor, webGLVendor, webGLRenderer string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Platform = platform
		opts.Vendor = vendor
		opts.WebGLVendor = webGLVendor
		opts.WebGLRenderer = webGLRenderer
	}
}

// WithPlugins overrides navigator.plugins values used by the fingerprint layer.
func WithPlugins(plugins ...string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Plugins = plugins
	}
}

// WithDisableWebRTC disables WebRTC APIs for this screenshot.
func WithDisableWebRTC() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.DisableWebRTC = true
	}
}

// WithSpoofedScreen sets spoofed screen dimensions.
func WithSpoofedScreen(width, height int) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.SpoofScreenSize = true
		opts.ScreenWidth = width
		opts.ScreenHeight = height
	}
}

// WithInjectedCookies appends cookies to inject before capture.
func WithInjectedCookies(cookies ...runner.CustomCookie) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Cookies = append(opts.Cookies, cookies...)
	}
}

// WithCookieHeader parses and injects cookies from a Cookie header value.
func WithCookieHeader(header string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.CookieHeader = header
	}
}

// WithCookieStrings parses and injects multiple Cookie header values.
func WithCookieStrings(headers ...string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.CookieStrings = append(opts.CookieStrings, headers...)
	}
}

// WithCookieImport imports cookies from a Netscape/Mozilla cookie file.
func WithCookieImport(path string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.CookieImport = path
	}
}

// WithCookieExport exports result cookies to a Netscape/Mozilla cookie file.
func WithCookieExport(path string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.CookieExport = path
		opts.SaveCookies = true
	}
}

// WithCookieWriteBack stores result cookies back into the SDK CookieJar.
func WithCookieWriteBack() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.CookieWriteBack = true
	}
}

// WithBlacklist enables URL blacklist checks using only the provided custom patterns.
func WithBlacklist(patterns ...string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.EnableBlacklist = boolPtr(true)
		opts.DefaultBlacklist = boolPtr(false)
		opts.BlacklistPatterns = append([]string{}, patterns...)
		opts.BlacklistFile = ""
		opts.clearBlacklistFile = true
	}
}

// WithDefaultBlacklist enables the built-in URL blacklist for this screenshot.
func WithDefaultBlacklist() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.EnableBlacklist = boolPtr(true)
		opts.DefaultBlacklist = boolPtr(true)
	}
}

// WithBlacklistFile enables URL blacklist checks from a pattern file.
func WithBlacklistFile(path string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.EnableBlacklist = boolPtr(true)
		opts.BlacklistFile = path
		opts.clearBlacklistFile = false
	}
}

// WithNoBlacklist disables URL blacklist checks for this screenshot.
func WithNoBlacklist() ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.EnableBlacklist = boolPtr(false)
		opts.DefaultBlacklist = boolPtr(false)
		opts.BlacklistPatterns = []string{}
		opts.BlacklistFile = ""
		opts.clearBlacklistFile = true
	}
}

// WithActions sets the interaction sequence to run before capture.
func WithActions(actions ...runner.InteractionAction) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Actions = actions
	}
}

// WithForm configures form fill and submit before capture.
func WithForm(form runner.Form) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Form = form
	}
}

// WithMaxRetries overrides retry count for this screenshot.
func WithMaxRetries(maxRetries int) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.MaxRetries = maxRetries
	}
}
