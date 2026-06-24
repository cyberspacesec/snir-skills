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
	if opts.CustomHeaders != nil {
		cloned.CustomHeaders = make(map[string]string, len(opts.CustomHeaders))
		for name, value := range opts.CustomHeaders {
			cloned.CustomHeaders[name] = value
		}
	}
	if opts.Cookies != nil {
		cloned.Cookies = append([]runner.CustomCookie(nil), opts.Cookies...)
	}
	if opts.Actions != nil {
		cloned.Actions = append([]runner.InteractionAction(nil), opts.Actions...)
	}
	return &cloned
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
	}
}

// WithDevice applies a named device preset.
func WithDevice(device string) ScreenshotOption {
	return func(opts *ScreenshotOptions) {
		opts.Device = device
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
