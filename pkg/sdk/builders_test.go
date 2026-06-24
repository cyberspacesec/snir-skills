package sdk

import (
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestNewScreenshotOptions(t *testing.T) {
	cookie := runner.CustomCookie{Name: "session", Value: "abc", Domain: "example.com"}
	action := runner.InteractionAction{Type: "click", Selector: "#submit"}
	form := runner.Form{SubmitSelector: "#login"}

	opts := NewScreenshotOptions(
		WithTimeout(11*time.Second),
		WithDelay(2*time.Second),
		WithViewport(390, 844),
		WithUserAgent("agent"),
		WithProxy("http://127.0.0.1:8080"),
		WithDevice("iphone-15"),
		WithIgnoreCertErrors(),
		WithFullPage(),
		WithElement("#main"),
		WithXPath("//main"),
		WithFormat("jpeg", 80),
		WithSkipSave(),
		WithJSBefore("window.test = true"),
		WithJSFile("preload.js", true),
		WithEvidence(),
		WithCustomHeaders(map[string]string{"X-Test": "1"}),
		WithAcceptLanguage("zh-CN"),
		WithFingerprint("Win32", "Google Inc.", "Intel", "Iris"),
		WithPlugins("PDF Viewer"),
		WithDisableWebRTC(),
		WithSpoofedScreen(1920, 1080),
		WithInjectedCookies(cookie),
		WithActions(action),
		WithForm(form),
		WithMaxRetries(3),
	)

	if opts.Timeout != 11*time.Second || opts.Delay != 2*time.Second {
		t.Fatalf("timeout/delay = %v/%v", opts.Timeout, opts.Delay)
	}
	if opts.WindowWidth != 390 || opts.WindowHeight != 844 {
		t.Fatalf("viewport = %dx%d", opts.WindowWidth, opts.WindowHeight)
	}
	if opts.UserAgent != "agent" || opts.Proxy != "http://127.0.0.1:8080" || opts.Device != "iphone-15" {
		t.Fatalf("browser overrides not set: %+v", opts)
	}
	if !opts.IgnoreCertErrors || !opts.CaptureFullPage || !opts.SkipSave {
		t.Fatalf("bool options not set: %+v", opts)
	}
	if opts.Selector != "#main" || opts.XPath != "//main" {
		t.Fatalf("element selectors not set: %+v", opts)
	}
	if opts.ScreenshotFormat != "jpeg" || opts.ScreenshotQuality != 80 {
		t.Fatalf("format = %s/%d", opts.ScreenshotFormat, opts.ScreenshotQuality)
	}
	if opts.JavaScript != "window.test = true" || opts.JavaScriptFile != "preload.js" ||
		!opts.RunJSBefore || opts.RunJSAfter {
		t.Fatalf("js options not set: %+v", opts)
	}
	if !opts.SaveHTML || !opts.SaveHeaders || !opts.SaveConsole || !opts.SaveCookies || !opts.SaveNetwork {
		t.Fatalf("evidence options not set: %+v", opts)
	}
	if opts.CustomHeaders["X-Test"] != "1" || opts.AcceptLanguage != "zh-CN" {
		t.Fatalf("headers/language not set: %+v", opts)
	}
	if opts.Platform != "Win32" || opts.Vendor != "Google Inc." || opts.WebGLVendor != "Intel" || opts.WebGLRenderer != "Iris" {
		t.Fatalf("fingerprint not set: %+v", opts)
	}
	if len(opts.Plugins) != 1 || opts.Plugins[0] != "PDF Viewer" {
		t.Fatalf("plugins = %v", opts.Plugins)
	}
	if !opts.DisableWebRTC || !opts.SpoofScreenSize || opts.ScreenWidth != 1920 || opts.ScreenHeight != 1080 {
		t.Fatalf("privacy/screen options not set: %+v", opts)
	}
	if len(opts.Cookies) != 1 || opts.Cookies[0].Name != "session" {
		t.Fatalf("cookies = %+v", opts.Cookies)
	}
	if len(opts.Actions) != 1 || opts.Actions[0].Selector != "#submit" {
		t.Fatalf("actions = %+v", opts.Actions)
	}
	if opts.Form.SubmitSelector != "#login" {
		t.Fatalf("form = %+v", opts.Form)
	}
	if opts.MaxRetries != 3 {
		t.Fatalf("MaxRetries = %d", opts.MaxRetries)
	}

	afterOpts := NewScreenshotOptions(WithJS("window.after = true"))
	if afterOpts.JavaScript != "window.after = true" || afterOpts.RunJSBefore || !afterOpts.RunJSAfter {
		t.Fatalf("WithJS timing = %+v", afterOpts)
	}
	afterOpts = NewScreenshotOptions(WithJSAfter("window.after = true"))
	if afterOpts.JavaScript != "window.after = true" || afterOpts.RunJSBefore || !afterOpts.RunJSAfter {
		t.Fatalf("WithJSAfter timing = %+v", afterOpts)
	}
	fileAfterOpts := NewScreenshotOptions(WithJSFile("after.js", false))
	if fileAfterOpts.JavaScriptFile != "after.js" || fileAfterOpts.RunJSBefore || !fileAfterOpts.RunJSAfter {
		t.Fatalf("WithJSFile after timing = %+v", fileAfterOpts)
	}
}

func TestCloneScreenshotOptions(t *testing.T) {
	if CloneScreenshotOptions(nil) == nil {
		t.Fatal("CloneScreenshotOptions(nil) returned nil")
	}

	opts := &ScreenshotOptions{
		UserAgent:     "agent",
		Plugins:       []string{"PDF Viewer"},
		CustomHeaders: map[string]string{"X-Test": "1"},
		Cookies:       []runner.CustomCookie{{Name: "session", Value: "abc"}},
		Actions:       []runner.InteractionAction{{Type: "click", Selector: "#submit"}},
	}
	cloned := CloneScreenshotOptions(opts)
	if cloned == opts {
		t.Fatal("CloneScreenshotOptions returned the same pointer")
	}
	if cloned.UserAgent != "agent" {
		t.Fatalf("UserAgent = %q", cloned.UserAgent)
	}
	cloned.Plugins[0] = "Changed"
	cloned.CustomHeaders["X-Test"] = "2"
	cloned.Cookies[0].Value = "changed"
	cloned.Actions[0].Selector = "#changed"
	if opts.Plugins[0] != "PDF Viewer" || opts.CustomHeaders["X-Test"] != "1" ||
		opts.Cookies[0].Value != "abc" || opts.Actions[0].Selector != "#submit" {
		t.Fatalf("CloneScreenshotOptions shared mutable fields: original=%+v cloned=%+v", opts, cloned)
	}
}
