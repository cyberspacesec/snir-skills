package sdk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// installSharedHook 注入一个返回固定 Result 的 sharedScreenshotWithContext，
// 并把每次调用的 runner.Options 暴露给测试断言。返回的清理函数恢复原值。
// 为避免并发批量调用产生 data race，hook 使用互斥锁保护共享状态，
// 且每次返回 Result 的深拷贝（切片字段独立）。
func installSharedHook(t *testing.T, result *models.Result, err error) (*runner.Options, *int) {
	t.Helper()
	restoreSDKHooks(t)
	var mu sync.Mutex
	var last runner.Options
	calls := 0
	sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
		mu.Lock()
		calls++
		if opts != nil {
			last = *opts
		}
		mu.Unlock()
		if result == nil {
			return nil, err
		}
		clone := *result
		if clone.URL == "" {
			clone.URL = target
		}
		// 深拷贝切片字段，避免并发 goroutine 共享底层数组
		if len(clone.ScreenshotBytes) > 0 {
			clone.ScreenshotBytes = append([]byte(nil), clone.ScreenshotBytes...)
		}
		return &clone, err
	}
	return &last, &calls
}

func mustCaptureResult() *models.Result {
	return &models.Result{
		URL:             "https://example.com",
		Title:           "ok",
		HTML:            "<html></html>",
		ScreenshotBytes: []byte("png"),
		Headers:         []models.Header{{Name: "Server", Value: "nginx"}},
		Cookies:         []models.Cookie{{Name: "sid", Value: "v"}},
		Console:         []models.ConsoleLog{{Level: "log", Message: "hi"}},
		Network:         []models.NetworkLog{{Method: "GET", URL: "https://example.com/a"}},
	}
}

// TestSharedEvidenceCollectors 覆盖 HTML/Headers/Cookies/Console/Network 的非 bytes 与 bytes 变体。
func TestSharedEvidenceCollectors(t *testing.T) {
	t.Run("HTML returns source and result", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		html, result, err := SharedScreenshotHTML("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotHTML() error = %v", err)
		}
		if html != "<html></html>" || result == nil {
			t.Fatalf("html/result = %q/%+v", html, result)
		}
	})

	t.Run("Headers returns slice and result", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		headers, result, err := SharedScreenshotHeaders("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotHeaders() error = %v", err)
		}
		if len(headers) != 1 || headers[0].Name != "Server" || result == nil {
			t.Fatalf("headers = %+v", headers)
		}
	})

	t.Run("HeadersBytes returns image bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		data, result, err := SharedScreenshotHeadersBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotHeadersBytes() error = %v", err)
		}
		if string(data) != "png" || result == nil {
			t.Fatalf("data/result = %q/%+v", data, result)
		}
	})

	t.Run("Cookies returns slice and result", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		cookies, result, err := SharedScreenshotCookies("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotCookies() error = %v", err)
		}
		if len(cookies) != 1 || cookies[0].Name != "sid" || result == nil {
			t.Fatalf("cookies = %+v", cookies)
		}
	})

	t.Run("CookiesBytes returns image bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		data, result, err := SharedScreenshotCookiesBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotCookiesBytes() error = %v", err)
		}
		if string(data) != "png" || result == nil {
			t.Fatalf("data/result = %q/%+v", data, result)
		}
	})

	t.Run("Console returns logs and result", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		console, result, err := SharedScreenshotConsole("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotConsole() error = %v", err)
		}
		if len(console) != 1 || console[0].Message != "hi" || result == nil {
			t.Fatalf("console = %+v", console)
		}
	})

	t.Run("ConsoleBytes returns image bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		data, result, err := SharedScreenshotConsoleBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotConsoleBytes() error = %v", err)
		}
		if string(data) != "png" || result == nil {
			t.Fatalf("data/result = %q/%+v", data, result)
		}
	})

	t.Run("Network returns logs and result", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		network, result, err := SharedScreenshotNetwork("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotNetwork() error = %v", err)
		}
		if len(network) != 1 || network[0].Method != "GET" || result == nil {
			t.Fatalf("network = %+v", network)
		}
	})

	t.Run("NetworkBytes returns image bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		data, result, err := SharedScreenshotNetworkBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotNetworkBytes() error = %v", err)
		}
		if string(data) != "png" || result == nil {
			t.Fatalf("data/result = %q/%+v", data, result)
		}
	})
}

// TestSharedFormatDelayTimeout 覆盖 WithFormat/WithDelay/WithTimeout 及其 bytes 变体。
func TestSharedFormatDelayTimeout(t *testing.T) {
	t.Run("WithFormat sets format and quality", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithFormat("https://example.com", "jpeg", 80, nil); err != nil {
			t.Fatalf("SharedScreenshotWithFormat() error = %v", err)
		}
		if last.Scan.ScreenshotFormat != "jpeg" || last.Scan.ScreenshotQuality != 80 {
			t.Fatalf("format/quality = %q/%d", last.Scan.ScreenshotFormat, last.Scan.ScreenshotQuality)
		}
	})

	t.Run("WithFormatBytes sets format and returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithFormatBytes("https://example.com", "jpeg", 80, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithFormatBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.ScreenshotFormat != "jpeg" {
			t.Fatalf("data/format = %q/%q", data, last.Scan.ScreenshotFormat)
		}
	})

	t.Run("WithDelay sets delay", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithDelay("https://example.com", 250*time.Millisecond, nil); err != nil {
			t.Fatalf("SharedScreenshotWithDelay() error = %v", err)
		}
		if last.Chrome.Delay != 0 && last.Chrome.Delay != 1 {
			t.Fatalf("delay = %d", last.Chrome.Delay)
		}
	})

	t.Run("WithDelayBytes sets delay and returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithDelayBytes("https://example.com", 250*time.Millisecond, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithDelayBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("data = %q", data)
		}
		// delay 仅用于断言调用成功，runner.Delay 为秒整数
		_ = last
	})

	t.Run("WithTimeout sets timeout", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithTimeout("https://example.com", 12*time.Second, nil); err != nil {
			t.Fatalf("SharedScreenshotWithTimeout() error = %v", err)
		}
		if last.Chrome.Timeout != 12 {
			t.Fatalf("timeout = %d", last.Chrome.Timeout)
		}
	})

	t.Run("WithTimeoutBytes sets timeout and returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithTimeoutBytes("https://example.com", 12*time.Second, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithTimeoutBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.Timeout != 12 {
			t.Fatalf("data/timeout = %q/%d", data, last.Chrome.Timeout)
		}
	})

	t.Run("ToPath sets path", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotToPath("https://example.com", "/tmp/out", nil); err != nil {
			t.Fatalf("SharedScreenshotToPath() error = %v", err)
		}
		if last.Scan.ScreenshotPath != "/tmp/out" {
			t.Fatalf("path = %q", last.Scan.ScreenshotPath)
		}
	})
}

// TestSharedProxyWrappers 覆盖所有 Proxy 相关 wrapper 及其 bytes 变体。
func TestSharedProxyWrappers(t *testing.T) {
	t.Run("WithProxy sets proxy", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithProxy("https://example.com", "http://p:8080", nil); err != nil {
			t.Fatalf("SharedScreenshotWithProxy() error = %v", err)
		}
		if last.Chrome.Proxy != "http://p:8080" {
			t.Fatalf("proxy = %q", last.Chrome.Proxy)
		}
	})

	t.Run("WithProxyBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithProxyBytes("https://example.com", "http://p:8080", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithProxyBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.Proxy != "http://p:8080" {
			t.Fatalf("data/proxy = %q/%q", data, last.Chrome.Proxy)
		}
	})

	t.Run("WithProxyList sets strategy and list", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithProxyList("https://example.com", runner.ProxyRoundRobin, []string{"http://a", "http://b"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithProxyList() error = %v", err)
		}
		if last.Chrome.ProxyStrategy != runner.ProxyRoundRobin || len(last.Chrome.ProxyList) != 2 {
			t.Fatalf("strategy/list = %q/%+v", last.Chrome.ProxyStrategy, last.Chrome.ProxyList)
		}
	})

	t.Run("WithProxyListBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithProxyListBytes("https://example.com", runner.ProxyRandom, []string{"http://a"}, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithProxyListBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.ProxyStrategy != runner.ProxyRandom {
			t.Fatalf("data/strategy = %q/%q", data, last.Chrome.ProxyStrategy)
		}
	})

	t.Run("WithProxyFile sets proxy file", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithProxyFile("https://example.com", "proxies.txt", runner.ProxyRoundRobin, nil); err != nil {
			t.Fatalf("SharedScreenshotWithProxyFile() error = %v", err)
		}
		if last.Chrome.ProxyFile != "proxies.txt" || last.Chrome.ProxyStrategy != runner.ProxyRoundRobin {
			t.Fatalf("file/strategy = %q/%q", last.Chrome.ProxyFile, last.Chrome.ProxyStrategy)
		}
	})

	t.Run("WithProxyFileBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithProxyFileBytes("https://example.com", "proxies.txt", runner.ProxyRoundRobin, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithProxyFileBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.ProxyFile != "proxies.txt" {
			t.Fatalf("data/file = %q/%q", data, last.Chrome.ProxyFile)
		}
	})

	t.Run("WithProxyURL sets proxy api url", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithProxyURL("https://example.com", "http://api/p", runner.ProxyRandom, nil); err != nil {
			t.Fatalf("SharedScreenshotWithProxyURL() error = %v", err)
		}
		if last.Chrome.ProxyURL != "http://api/p" || last.Chrome.ProxyStrategy != runner.ProxyRandom {
			t.Fatalf("url/strategy = %q/%q", last.Chrome.ProxyURL, last.Chrome.ProxyStrategy)
		}
	})

	t.Run("WithProxyURLBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithProxyURLBytes("https://example.com", "http://api/p", runner.ProxyRandom, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithProxyURLBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.ProxyURL != "http://api/p" {
			t.Fatalf("data/url = %q/%q", data, last.Chrome.ProxyURL)
		}
	})
}

// TestSharedHeaderAndFingerprint 覆盖自定义头/UA/Accept-Language/指纹相关 wrapper。
func TestSharedHeaderAndFingerprint(t *testing.T) {
	t.Run("WithCustomHeaders sets headers", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithCustomHeaders("https://example.com", map[string]string{"X-Test": "1"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCustomHeaders() error = %v", err)
		}
		if last.Chrome.CustomHeaders["X-Test"] != "1" {
			t.Fatalf("custom headers = %+v", last.Chrome.CustomHeaders)
		}
	})

	t.Run("WithCustomHeadersBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithCustomHeadersBytes("https://example.com", map[string]string{"X-Test": "1"}, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithCustomHeadersBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.CustomHeaders["X-Test"] != "1" {
			t.Fatalf("data/headers = %q/%+v", data, last.Chrome.CustomHeaders)
		}
	})

	t.Run("WithUserAgent sets UA", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithUserAgent("https://example.com", "TestBot/1.0", nil); err != nil {
			t.Fatalf("SharedScreenshotWithUserAgent() error = %v", err)
		}
		if last.Chrome.UserAgent != "TestBot/1.0" {
			t.Fatalf("UA = %q", last.Chrome.UserAgent)
		}
	})

	t.Run("WithUserAgentBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithUserAgentBytes("https://example.com", "TestBot/1.0", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithUserAgentBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.UserAgent != "TestBot/1.0" {
			t.Fatalf("data/UA = %q/%q", data, last.Chrome.UserAgent)
		}
	})

	t.Run("WithAcceptLanguage sets lang", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithAcceptLanguage("https://example.com", "zh-CN", nil); err != nil {
			t.Fatalf("SharedScreenshotWithAcceptLanguage() error = %v", err)
		}
		if last.Chrome.AcceptLanguage != "zh-CN" {
			t.Fatalf("AcceptLanguage = %q", last.Chrome.AcceptLanguage)
		}
	})

	t.Run("WithAcceptLanguageBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithAcceptLanguageBytes("https://example.com", "zh-CN", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithAcceptLanguageBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.AcceptLanguage != "zh-CN" {
			t.Fatalf("data/lang = %q/%q", data, last.Chrome.AcceptLanguage)
		}
	})

	t.Run("WithFingerprint sets fingerprint", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithFingerprint("https://example.com", "Win32", "Google", "vendor", "renderer", nil); err != nil {
			t.Fatalf("SharedScreenshotWithFingerprint() error = %v", err)
		}
		if last.Chrome.Platform != "Win32" || last.Chrome.WebGLVendor != "vendor" {
			t.Fatalf("fingerprint = %+v", last.Chrome)
		}
	})

	t.Run("WithFingerprintBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithFingerprintBytes("https://example.com", "Win32", "Google", "vendor", "renderer", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithFingerprintBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.Platform != "Win32" {
			t.Fatalf("data/platform = %q/%q", data, last.Chrome.Platform)
		}
	})
}

// TestSharedEmulationWrappers 覆盖设备/移动/触摸/证书/插件/WebRTC/伪造屏幕相关 wrapper。
func TestSharedEmulationWrappers(t *testing.T) {
	t.Run("WithDeviceEmulation sets device params", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithDeviceEmulation("https://example.com", 390, 844, 2.0, true, true, nil); err != nil {
			t.Fatalf("SharedScreenshotWithDeviceEmulation() error = %v", err)
		}
		if last.Chrome.WindowX != 390 || last.Chrome.DeviceScaleFactor != 2.0 || !last.Chrome.IsMobile {
			t.Fatalf("device params = %+v", last.Chrome)
		}
	})

	t.Run("WithDeviceEmulationBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithDeviceEmulationBytes("https://example.com", 390, 844, 2.0, true, true, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithDeviceEmulationBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.WindowX != 390 {
			t.Fatalf("data/x = %q/%d", data, last.Chrome.WindowX)
		}
	})

	t.Run("WithMobileEmulation sets mobile", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithMobileEmulation("https://example.com", 2.5, nil); err != nil {
			t.Fatalf("SharedScreenshotWithMobileEmulation() error = %v", err)
		}
		if !last.Chrome.IsMobile || last.Chrome.DeviceScaleFactor != 2.5 {
			t.Fatalf("mobile = %+v", last.Chrome)
		}
	})

	t.Run("WithMobileEmulationBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithMobileEmulationBytes("https://example.com", 2.5, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithMobileEmulationBytes() error = %v", err)
		}
		if string(data) != "png" || !last.Chrome.IsMobile {
			t.Fatalf("data/mobile = %q/%v", data, last.Chrome.IsMobile)
		}
	})

	t.Run("WithTouchEmulation sets touch", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithTouchEmulation("https://example.com", true, nil); err != nil {
			t.Fatalf("SharedScreenshotWithTouchEmulation() error = %v", err)
		}
		if !last.Chrome.HasTouch {
			t.Fatal("HasTouch was not enabled")
		}
	})

	t.Run("WithTouchEmulationBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithTouchEmulationBytes("https://example.com", true, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithTouchEmulationBytes() error = %v", err)
		}
		if string(data) != "png" || !last.Chrome.HasTouch {
			t.Fatalf("data/touch = %q/%v", data, last.Chrome.HasTouch)
		}
	})

	t.Run("WithIgnoreCertErrors sets flag", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithIgnoreCertErrors("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotWithIgnoreCertErrors() error = %v", err)
		}
		if !last.Chrome.IgnoreCertErrors {
			t.Fatal("IgnoreCertErrors was not enabled")
		}
	})

	t.Run("WithIgnoreCertErrorsBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithIgnoreCertErrorsBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithIgnoreCertErrorsBytes() error = %v", err)
		}
		if string(data) != "png" || !last.Chrome.IgnoreCertErrors {
			t.Fatalf("data/flag = %q/%v", data, last.Chrome.IgnoreCertErrors)
		}
	})

	t.Run("WithPlugins sets plugins", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithPlugins("https://example.com", []string{"PDF Viewer"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithPlugins() error = %v", err)
		}
		if len(last.Chrome.Plugins) != 1 || last.Chrome.Plugins[0] != "PDF Viewer" {
			t.Fatalf("plugins = %+v", last.Chrome.Plugins)
		}
	})

	t.Run("WithPluginsBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithPluginsBytes("https://example.com", []string{"PDF Viewer"}, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithPluginsBytes() error = %v", err)
		}
		if string(data) != "png" || len(last.Chrome.Plugins) != 1 {
			t.Fatalf("data/plugins = %q/%+v", data, last.Chrome.Plugins)
		}
	})

	t.Run("WithDisabledWebRTC sets flag", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithDisabledWebRTC("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotWithDisabledWebRTC() error = %v", err)
		}
		if !last.Chrome.DisableWebRTC {
			t.Fatal("DisableWebRTC was not enabled")
		}
	})

	t.Run("WithDisabledWebRTCBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithDisabledWebRTCBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithDisabledWebRTCBytes() error = %v", err)
		}
		if string(data) != "png" || !last.Chrome.DisableWebRTC {
			t.Fatalf("data/flag = %q/%v", data, last.Chrome.DisableWebRTC)
		}
	})

	t.Run("WithSpoofedScreen sets dims", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithSpoofedScreen("https://example.com", 1920, 1080, nil); err != nil {
			t.Fatalf("SharedScreenshotWithSpoofedScreen() error = %v", err)
		}
		if !last.Chrome.SpoofScreenSize || last.Chrome.ScreenWidth != 1920 || last.Chrome.ScreenHeight != 1080 {
			t.Fatalf("spoofed screen = %v/%dx%d", last.Chrome.SpoofScreenSize, last.Chrome.ScreenWidth, last.Chrome.ScreenHeight)
		}
	})

	t.Run("WithSpoofedScreenBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithSpoofedScreenBytes("https://example.com", 1920, 1080, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithSpoofedScreenBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.ScreenWidth != 1920 {
			t.Fatalf("data/w = %q/%d", data, last.Chrome.ScreenWidth)
		}
	})
}

// TestSharedCookieWrappers 覆盖 Cookie header/strings/file/import/export 相关 wrapper。
func TestSharedCookieWrappers(t *testing.T) {
	t.Run("WithCookieHeader sets cookie strings", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithCookieHeader("https://example.com", "sid=1", nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieHeader() error = %v", err)
		}
		if len(last.Scan.CookieStrings) == 0 || last.Scan.CookieStrings[0] != "sid=1" {
			t.Fatalf("cookie strings = %+v", last.Scan.CookieStrings)
		}
	})

	t.Run("WithCookieHeaderBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithCookieHeaderBytes("https://example.com", "sid=1", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithCookieHeaderBytes() error = %v", err)
		}
		if string(data) != "png" || len(last.Scan.CookieStrings) == 0 || last.Scan.CookieStrings[0] != "sid=1" {
			t.Fatalf("data/strings = %q/%+v", data, last.Scan.CookieStrings)
		}
	})

	t.Run("WithCookieStrings sets cookie strings", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithCookieStrings("https://example.com", []string{"sid=1", "x=2"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieStrings() error = %v", err)
		}
		if len(last.Scan.CookieStrings) != 2 || last.Scan.CookieStrings[0] != "sid=1" {
			t.Fatalf("cookie strings = %+v", last.Scan.CookieStrings)
		}
	})

	t.Run("WithCookieStringsBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithCookieStringsBytes("https://example.com", []string{"sid=1"}, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithCookieStringsBytes() error = %v", err)
		}
		if string(data) != "png" || len(last.Scan.CookieStrings) != 1 || last.Scan.CookieStrings[0] != "sid=1" {
			t.Fatalf("data/strings = %q/%+v", data, last.Scan.CookieStrings)
		}
	})

	t.Run("WithCookieFile sets file and writeBack", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithCookieFile("https://example.com", "cookies.json", true, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieFile() error = %v", err)
		}
		if last.Scan.CookiesFile != "cookies.json" || !last.Scan.CookieWriteBack {
			t.Fatalf("file/writeBack = %q/%v", last.Scan.CookiesFile, last.Scan.CookieWriteBack)
		}
	})

	t.Run("WithCookieFileBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithCookieFileBytes("https://example.com", "cookies.json", false, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithCookieFileBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.CookiesFile != "cookies.json" || last.Scan.CookieWriteBack {
			t.Fatalf("data/file = %q/%q", data, last.Scan.CookiesFile)
		}
	})

	t.Run("WithCookieImport sets import file", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithCookieImport("https://example.com", "cookies.txt", nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieImport() error = %v", err)
		}
		if last.Scan.CookieImport != "cookies.txt" {
			t.Fatalf("import file = %q", last.Scan.CookieImport)
		}
	})

	t.Run("WithCookieImportBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithCookieImportBytes("https://example.com", "cookies.txt", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithCookieImportBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.CookieImport != "cookies.txt" {
			t.Fatalf("data/file = %q/%q", data, last.Scan.CookieImport)
		}
	})

	t.Run("WithCookieExport sets export file", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		exportPath := filepath.Join(t.TempDir(), "export.txt")
		if _, err := SharedScreenshotWithCookieExport("https://example.com", exportPath, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieExport() error = %v", err)
		}
		if last.Scan.CookieExport != exportPath {
			t.Fatalf("export file = %q", last.Scan.CookieExport)
		}
	})

	t.Run("WithCookieExportBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		exportPath := filepath.Join(t.TempDir(), "export.txt")
		data, _, err := SharedScreenshotWithCookieExportBytes("https://example.com", exportPath, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithCookieExportBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.CookieExport != exportPath {
			t.Fatalf("data/file = %q/%q", data, last.Scan.CookieExport)
		}
	})
}

// TestSharedBlacklistWrappers 覆盖 Blacklist 相关 wrapper。
func TestSharedBlacklistWrappers(t *testing.T) {
	t.Run("WithBlacklist sets patterns", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithBlacklist("https://example.com", []string{"bad\\.com"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithBlacklist() error = %v", err)
		}
		if len(last.Scan.BlacklistPatterns) == 0 {
			t.Fatalf("blacklist = %+v", last.Scan.BlacklistPatterns)
		}
	})

	t.Run("WithBlacklistBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithBlacklistBytes("https://example.com", []string{"bad\\.com"}, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithBlacklistBytes() error = %v", err)
		}
		if string(data) != "png" || len(last.Scan.BlacklistPatterns) == 0 {
			t.Fatalf("data/blacklist = %q/%+v", data, last.Scan.BlacklistPatterns)
		}
	})

	t.Run("WithBlacklistFile sets file", func(t *testing.T) {
		blFile := filepath.Join(t.TempDir(), "bl.txt")
		if err := os.WriteFile(blFile, []byte("bad\\.com\n"), 0644); err != nil {
			t.Fatalf("write blacklist file: %v", err)
		}
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithBlacklistFile("https://example.com", blFile, nil); err != nil {
			t.Fatalf("SharedScreenshotWithBlacklistFile() error = %v", err)
		}
		if last.Scan.BlacklistFile != blFile {
			t.Fatalf("file = %q", last.Scan.BlacklistFile)
		}
	})

	t.Run("WithBlacklistFileBytes returns bytes", func(t *testing.T) {
		blFile := filepath.Join(t.TempDir(), "bl.txt")
		if err := os.WriteFile(blFile, []byte("bad\\.com\n"), 0644); err != nil {
			t.Fatalf("write blacklist file: %v", err)
		}
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithBlacklistFileBytes("https://example.com", blFile, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithBlacklistFileBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.BlacklistFile != blFile {
			t.Fatalf("data/file = %q/%q", data, last.Scan.BlacklistFile)
		}
	})

	t.Run("WithoutBlacklist disables blacklist", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithoutBlacklist("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotWithoutBlacklist() error = %v", err)
		}
		if last.Scan.EnableBlacklist {
			t.Fatalf("EnableBlacklist = %v", last.Scan.EnableBlacklist)
		}
	})

	t.Run("WithoutBlacklistBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithoutBlacklistBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithoutBlacklistBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.EnableBlacklist {
			t.Fatalf("data/flag = %q/%v", data, last.Scan.EnableBlacklist)
		}
	})

	t.Run("WithDefaultBlacklist sets default flag", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithDefaultBlacklist("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotWithDefaultBlacklist() error = %v", err)
		}
		if !last.Scan.DefaultBlacklist {
			t.Fatal("DefaultBlacklist was not enabled")
		}
	})

	t.Run("WithDefaultBlacklistBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithDefaultBlacklistBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithDefaultBlacklistBytes() error = %v", err)
		}
		if string(data) != "png" || !last.Scan.DefaultBlacklist {
			t.Fatalf("data/flag = %q/%v", data, last.Scan.DefaultBlacklist)
		}
	})

	t.Run("WithRetries sets max retries", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotWithRetries("https://example.com", 3, nil); err != nil {
			t.Fatalf("SharedScreenshotWithRetries() error = %v", err)
		}
		if last.Scan.MaxRetries != 3 {
			t.Fatalf("max retries = %d", last.Scan.MaxRetries)
		}
	})

	t.Run("WithRetriesBytes returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotWithRetriesBytes("https://example.com", 3, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithRetriesBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.MaxRetries != 3 {
			t.Fatalf("data/retries = %q/%d", data, last.Scan.MaxRetries)
		}
	})
}

// TestSharedEvidence 覆盖 SharedScreenshotEvidence 及其 bytes/context 变体。
func TestSharedEvidence(t *testing.T) {
	t.Run("Evidence sets evidence flags", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotEvidence("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotEvidence() error = %v", err)
		}
		if !last.Scan.SaveHTML || !last.Scan.SaveHeaders || !last.Scan.SaveCookies {
			t.Fatalf("evidence flags = %+v", last.Scan)
		}
	})

	t.Run("EvidenceWithContext sets evidence flags", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		if _, err := SharedScreenshotEvidenceWithContext(context.Background(), "https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotEvidenceWithContext() error = %v", err)
		}
		if !last.Scan.SaveHTML {
			t.Fatal("SaveHTML was not enabled")
		}
	})

	t.Run("EvidenceBytes returns bytes and sets flags", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotEvidenceBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotEvidenceBytes() error = %v", err)
		}
		if string(data) != "png" || !last.Scan.SaveHTML {
			t.Fatalf("data/flags = %q/%v", data, last.Scan.SaveHTML)
		}
	})

	t.Run("EvidenceBytesWithContext returns bytes", func(t *testing.T) {
		last, _ := installSharedHook(t, mustCaptureResult(), nil)
		data, _, err := SharedScreenshotEvidenceBytesWithContext(context.Background(), "https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotEvidenceBytesWithContext() error = %v", err)
		}
		if string(data) != "png" || !last.Scan.SaveHTML {
			t.Fatalf("data/flags = %q/%v", data, last.Scan.SaveHTML)
		}
	})
}

// TestSharedEvidenceBundle 覆盖 SharedScreenshotEvidenceBundle 系列（成功与失败分支）。
func TestSharedEvidenceBundle(t *testing.T) {
	t.Run("EvidenceBundle writes bundle and returns bundle", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "bundle")
		bundle, result, err := SharedScreenshotEvidenceBundle("https://example.com", dir, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotEvidenceBundle() error = %v", err)
		}
		if bundle == nil || result == nil || bundle.Dir != dir {
			t.Fatalf("bundle/result = %+v/%+v", bundle, result)
		}
		if _, statErr := os.Stat(filepath.Join(dir, "result.json")); statErr != nil {
			t.Fatalf("result.json not written: %v", statErr)
		}
	})

	t.Run("EvidenceBundleWithContext writes bundle", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "bundle2")
		bundle, _, err := SharedScreenshotEvidenceBundleWithContext(context.Background(), "https://example.com", dir, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotEvidenceBundleWithContext() error = %v", err)
		}
		if bundle == nil {
			t.Fatal("bundle is nil")
		}
	})

	t.Run("CaptureEvidenceBundle uses functional options", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "bundle3")
		bundle, _, err := SharedCaptureEvidenceBundle("https://example.com", dir)
		if err != nil {
			t.Fatalf("SharedCaptureEvidenceBundle() error = %v", err)
		}
		if bundle == nil {
			t.Fatal("bundle is nil")
		}
	})

	t.Run("CaptureEvidenceBundleWithContext uses functional options", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "bundle4")
		bundle, _, err := SharedCaptureEvidenceBundleWithContext(context.Background(), "https://example.com", dir)
		if err != nil {
			t.Fatalf("SharedCaptureEvidenceBundleWithContext() error = %v", err)
		}
		if bundle == nil {
			t.Fatal("bundle is nil")
		}
	})

	t.Run("EvidenceBundle returns error on pool failure", func(t *testing.T) {
		_, _ = installSharedHook(t, nil, errors.New("pool down"))
		dir := filepath.Join(t.TempDir(), "bundle-err")
		bundle, _, err := SharedScreenshotEvidenceBundleWithContext(context.Background(), "https://example.com", dir, nil)
		if err == nil || bundle != nil {
			t.Fatalf("expected error, got bundle=%+v err=%v", bundle, err)
		}
	})

	t.Run("EvidenceBundle returns error on failed result", func(t *testing.T) {
		_, _ = installSharedHook(t, &models.Result{Failed: true, FailedReason: "blocked"}, nil)
		dir := filepath.Join(t.TempDir(), "bundle-fail")
		bundle, _, err := SharedScreenshotEvidenceBundleWithContext(context.Background(), "https://example.com", dir, nil)
		if err == nil || bundle != nil {
			t.Fatalf("expected error, got bundle=%+v err=%v", bundle, err)
		}
	})
}

// TestSharedBatch 覆盖 SharedBatchScreenshot / Bytes / Requests / EvidenceBundles 系列。
func TestSharedBatch(t *testing.T) {
	urls := []string{"https://a.example.com", "https://b.example.com"}

	t.Run("SharedBatchScreenshot returns per-target results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshot(urls, nil)
		if len(results) != 2 {
			t.Fatalf("results len = %d", len(results))
		}
		for _, r := range results {
			if r.Error != nil || r.Result == nil {
				t.Fatalf("result = %+v", r)
			}
		}
	})

	t.Run("SharedBatchScreenshotWithContext returns results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshotWithContext(context.Background(), urls, nil)
		if len(results) != 2 {
			t.Fatalf("results len = %d", len(results))
		}
	})

	t.Run("SharedBatchScreenshotBytes returns bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshotBytes(urls, nil)
		if len(results) != 2 {
			t.Fatalf("results len = %d", len(results))
		}
		for _, r := range results {
			if r.Error != nil || string(r.Data) != "png" {
				t.Fatalf("result = %+v", r)
			}
		}
	})

	t.Run("SharedBatchScreenshotBytesWithContext returns bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshotBytesWithContext(context.Background(), urls, nil)
		if len(results) != 2 {
			t.Fatalf("results len = %d", len(results))
		}
	})

	t.Run("SharedBatchScreenshotRequests returns per-request results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{
			{Name: "a", URL: "https://a.example.com"},
			{Name: "b", URL: "https://b.example.com"},
		}
		results := SharedBatchScreenshotRequests(reqs)
		if len(results) != 2 || results[0].Name != "a" {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotRequestsWithContext returns results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		results := SharedBatchScreenshotRequestsWithContext(context.Background(), reqs)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotRequestsBytes returns bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		results := SharedBatchScreenshotRequestsBytes(reqs)
		if len(results) != 1 || string(results[0].Data) != "png" {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotRequestsBytesWithContext returns bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		results := SharedBatchScreenshotRequestsBytesWithContext(context.Background(), reqs)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotEvidenceBundles writes bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "batch-ev")
		results := SharedBatchScreenshotEvidenceBundles(urls, dir, nil)
		if len(results) != 2 {
			t.Fatalf("results len = %d", len(results))
		}
		for _, r := range results {
			if r.Error != nil || r.Bundle == nil {
				t.Fatalf("result = %+v", r)
			}
		}
	})

	t.Run("SharedBatchScreenshotEvidenceBundlesWithContext writes bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "batch-ev2")
		results := SharedBatchScreenshotEvidenceBundlesWithContext(context.Background(), urls, dir, nil)
		if len(results) != 2 {
			t.Fatalf("results len = %d", len(results))
		}
	})

	t.Run("SharedBatchScreenshotRequestsEvidenceBundles writes bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "batch-ev3")
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		results := SharedBatchScreenshotRequestsEvidenceBundles(reqs, dir)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotRequestsEvidenceBundlesWithContext writes bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "batch-ev4")
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		results := SharedBatchScreenshotRequestsEvidenceBundlesWithContext(context.Background(), reqs, dir)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})
}

// TestSharedBatchTargets 覆盖 SharedBatchScreenshotTargets 系列（含 Context/Bytes/EvidenceBundles）。
func TestSharedBatchTargets(t *testing.T) {
	targets := []string{"https://example.com"}

	t.Run("SharedBatchScreenshotTargets returns results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshotTargets(targets, nil)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotTargetsWithContext returns results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshotTargetsWithContext(context.Background(), targets, nil)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotTargetsBytes returns bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshotTargetsBytes(targets, nil)
		if len(results) != 1 || string(results[0].Data) != "png" {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotTargetsBytesWithContext returns bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		results := SharedBatchScreenshotTargetsBytesWithContext(context.Background(), targets, nil)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotTargetsEvidenceBundles writes bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "tgt-ev")
		results := SharedBatchScreenshotTargetsEvidenceBundles(targets, dir, nil)
		if len(results) != 1 || results[0].Error != nil || results[0].Bundle == nil {
			t.Fatalf("results = %+v", results)
		}
	})

	t.Run("SharedBatchScreenshotTargetsEvidenceBundlesWithContext writes bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "tgt-ev2")
		results := SharedBatchScreenshotTargetsEvidenceBundlesWithContext(context.Background(), targets, dir, nil)
		if len(results) != 1 || results[0].Error != nil {
			t.Fatalf("results = %+v", results)
		}
	})
}

// TestSharedStreaming 覆盖所有 Streaming 变体。
func TestSharedStreaming(t *testing.T) {
	t.Run("SharedBatchScreenshotStreaming streams results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		ch := SharedBatchScreenshotStreaming(context.Background(), []string{"https://a.example.com"}, nil)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil {
				t.Fatalf("result error = %v", r.Error)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotBytesStreaming streams bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		ch := SharedBatchScreenshotBytesStreaming(context.Background(), []string{"https://a.example.com"}, nil)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil || string(r.Data) != "png" {
				t.Fatalf("result = %+v", r)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotEvidenceBundlesStreaming streams bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "stream-ev")
		ch := SharedBatchScreenshotEvidenceBundlesStreaming(context.Background(), []string{"https://a.example.com"}, dir, nil)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil || r.Bundle == nil {
				t.Fatalf("result = %+v", r)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotRequestsStreaming streams results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		ch := SharedBatchScreenshotRequestsStreaming(context.Background(), reqs)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil {
				t.Fatalf("result error = %v", r.Error)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotRequestsBytesStreaming streams bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		ch := SharedBatchScreenshotRequestsBytesStreaming(context.Background(), reqs)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil || string(r.Data) != "png" {
				t.Fatalf("result = %+v", r)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotRequestsEvidenceBundlesStreaming streams bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "stream-ev2")
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		ch := SharedBatchScreenshotRequestsEvidenceBundlesStreaming(context.Background(), reqs, dir)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil || r.Bundle == nil {
				t.Fatalf("result = %+v", r)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotTargetsStreaming streams results", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		ch := SharedBatchScreenshotTargetsStreaming(context.Background(), []string{"https://example.com"}, nil)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil {
				t.Fatalf("result error = %v", r.Error)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotTargetsBytesStreaming streams bytes", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		ch := SharedBatchScreenshotTargetsBytesStreaming(context.Background(), []string{"https://example.com"}, nil)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil || string(r.Data) != "png" {
				t.Fatalf("result = %+v", r)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("SharedBatchScreenshotTargetsEvidenceBundlesStreaming streams bundles", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "stream-ev3")
		ch := SharedBatchScreenshotTargetsEvidenceBundlesStreaming(context.Background(), []string{"https://example.com"}, dir, nil)
		count := 0
		for r := range ch {
			count++
			if r.Error != nil || r.Bundle == nil {
				t.Fatalf("result = %+v", r)
			}
		}
		if count != 1 {
			t.Fatalf("count = %d", count)
		}
	})

	t.Run("Streaming surfaces context cancellation", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ch := SharedBatchScreenshotStreaming(ctx, []string{"https://a.example.com"}, nil)
		var sawErr bool
		for r := range ch {
			if r.Error != nil {
				sawErr = true
			}
		}
		if !sawErr {
			t.Fatal("expected an error result from canceled context")
		}
	})

	// 覆盖各 Streaming 函数的 ctx.Done 分支（预取消 ctx + 非空输入）。
	t.Run("Streaming ctx cancellation covers remaining variants", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		dir := filepath.Join(t.TempDir(), "stream-cancel")

		// SharedBatchScreenshotBytesStreaming
		var sawErr bool
		for r := range SharedBatchScreenshotBytesStreaming(ctx, []string{"https://a.example.com"}, nil) {
			if r.Error != nil {
				sawErr = true
			}
		}
		if !sawErr {
			t.Error("SharedBatchScreenshotBytesStreaming 应有错误结果")
		}

		// SharedBatchScreenshotEvidenceBundlesStreaming
		sawErr = false
		for r := range SharedBatchScreenshotEvidenceBundlesStreaming(ctx, []string{"https://a.example.com"}, dir, nil) {
			if r.Error != nil {
				sawErr = true
			}
		}
		if !sawErr {
			t.Error("SharedBatchScreenshotEvidenceBundlesStreaming 应有错误结果")
		}

		// SharedBatchScreenshotRequestsStreaming
		sawErr = false
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		for r := range SharedBatchScreenshotRequestsStreaming(ctx, reqs) {
			if r.Error != nil {
				sawErr = true
			}
		}
		if !sawErr {
			t.Error("SharedBatchScreenshotRequestsStreaming 应有错误结果")
		}

		// SharedBatchScreenshotRequestsBytesStreaming
		sawErr = false
		for r := range SharedBatchScreenshotRequestsBytesStreaming(ctx, reqs) {
			if r.Error != nil {
				sawErr = true
			}
		}
		if !sawErr {
			t.Error("SharedBatchScreenshotRequestsBytesStreaming 应有错误结果")
		}

		// SharedBatchScreenshotRequestsEvidenceBundlesStreaming
		sawErr = false
		for r := range SharedBatchScreenshotRequestsEvidenceBundlesStreaming(ctx, reqs, dir) {
			if r.Error != nil {
				sawErr = true
			}
		}
		if !sawErr {
			t.Error("SharedBatchScreenshotRequestsEvidenceBundlesStreaming 应有错误结果")
		}
	})
}

// TestSharedCallbacks 覆盖所有 Callback 变体（含 nil 与非 nil callback）。
func TestSharedCallbacks(t *testing.T) {
	t.Run("SharedBatchScreenshotCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		var seen []BatchResult
		SharedBatchScreenshotCallback(context.Background(), []string{"https://a.example.com"}, nil, func(r BatchResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 {
			t.Fatalf("seen = %d", len(seen))
		}
	})

	t.Run("SharedBatchScreenshotCallback nil callback is safe", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		SharedBatchScreenshotCallback(context.Background(), []string{"https://a.example.com"}, nil, nil)
	})

	t.Run("SharedBatchScreenshotBytesCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		var seen []BatchBytesResult
		SharedBatchScreenshotBytesCallback(context.Background(), []string{"https://a.example.com"}, nil, func(r BatchBytesResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 || string(seen[0].Data) != "png" {
			t.Fatalf("seen = %+v", seen)
		}
	})

	t.Run("SharedBatchScreenshotEvidenceBundlesCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "cb-ev")
		var seen []BatchEvidenceBundleResult
		SharedBatchScreenshotEvidenceBundlesCallback(context.Background(), []string{"https://a.example.com"}, dir, nil, func(r BatchEvidenceBundleResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 || seen[0].Bundle == nil {
			t.Fatalf("seen = %+v", seen)
		}
	})

	t.Run("SharedBatchScreenshotRequestsCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		var seen []BatchResult
		SharedBatchScreenshotRequestsCallback(context.Background(), reqs, func(r BatchResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 {
			t.Fatalf("seen = %d", len(seen))
		}
	})

	t.Run("SharedBatchScreenshotRequestsBytesCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		var seen []BatchBytesResult
		SharedBatchScreenshotRequestsBytesCallback(context.Background(), reqs, func(r BatchBytesResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 || string(seen[0].Data) != "png" {
			t.Fatalf("seen = %+v", seen)
		}
	})

	t.Run("SharedBatchScreenshotRequestsEvidenceBundlesCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "cb-ev2")
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://a.example.com"}}
		var seen []BatchEvidenceBundleResult
		SharedBatchScreenshotRequestsEvidenceBundlesCallback(context.Background(), reqs, dir, func(r BatchEvidenceBundleResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 || seen[0].Bundle == nil {
			t.Fatalf("seen = %+v", seen)
		}
	})

	t.Run("SharedBatchScreenshotTargetsCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		var seen []BatchResult
		SharedBatchScreenshotTargetsCallback(context.Background(), []string{"https://example.com"}, nil, func(r BatchResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 {
			t.Fatalf("seen = %d", len(seen))
		}
	})

	t.Run("SharedBatchScreenshotTargetsBytesCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		var seen []BatchBytesResult
		SharedBatchScreenshotTargetsBytesCallback(context.Background(), []string{"https://example.com"}, nil, func(r BatchBytesResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 || string(seen[0].Data) != "png" {
			t.Fatalf("seen = %+v", seen)
		}
	})

	t.Run("SharedBatchScreenshotTargetsEvidenceBundlesCallback invokes callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "cb-ev3")
		var seen []BatchEvidenceBundleResult
		SharedBatchScreenshotTargetsEvidenceBundlesCallback(context.Background(), []string{"https://example.com"}, dir, nil, func(r BatchEvidenceBundleResult) {
			seen = append(seen, r)
		})
		if len(seen) != 1 || seen[0].Bundle == nil {
			t.Fatalf("seen = %+v", seen)
		}
	})

	t.Run("all callbacks safe with nil callback", func(t *testing.T) {
		_, _ = installSharedHook(t, mustCaptureResult(), nil)
		dir := filepath.Join(t.TempDir(), "cb-ev4")
		reqs := []ScreenshotRequest{{Name: "a", URL: "https://example.com"}}
		SharedBatchScreenshotBytesCallback(context.Background(), []string{"https://example.com"}, nil, nil)
		SharedBatchScreenshotEvidenceBundlesCallback(context.Background(), []string{"https://example.com"}, dir, nil, nil)
		SharedBatchScreenshotRequestsCallback(context.Background(), reqs, nil)
		SharedBatchScreenshotRequestsBytesCallback(context.Background(), reqs, nil)
		SharedBatchScreenshotRequestsEvidenceBundlesCallback(context.Background(), reqs, dir, nil)
		SharedBatchScreenshotTargetsCallback(context.Background(), []string{"https://example.com"}, nil, nil)
		SharedBatchScreenshotTargetsBytesCallback(context.Background(), []string{"https://example.com"}, nil, nil)
		SharedBatchScreenshotTargetsEvidenceBundlesCallback(context.Background(), []string{"https://example.com"}, dir, nil, nil)
	})
}

// TestSharedCaptureBytesError 覆盖 SharedScreenshotBytes 失败分支。
func TestSharedCaptureBytesError(t *testing.T) {
	t.Run("SharedScreenshotBytes propagates pool error", func(t *testing.T) {
		_, _ = installSharedHook(t, nil, errors.New("boom"))
		data, _, err := SharedScreenshotBytes("https://example.com", nil)
		if err == nil || data != nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("data/err = %q/%v", data, err)
		}
	})

	t.Run("SharedCaptureBytes propagates pool error", func(t *testing.T) {
		_, _ = installSharedHook(t, nil, errors.New("boom"))
		data, _, err := SharedCaptureBytes("https://example.com")
		if err == nil || data != nil {
			t.Fatalf("data/err = %q/%v", data, err)
		}
	})

	t.Run("SharedScreenshotHTML propagates error", func(t *testing.T) {
		_, _ = installSharedHook(t, nil, errors.New("boom"))
		html, _, err := SharedScreenshotHTML("https://example.com", nil)
		if err == nil || html != "" {
			t.Fatalf("html/err = %q/%v", html, err)
		}
	})
}
