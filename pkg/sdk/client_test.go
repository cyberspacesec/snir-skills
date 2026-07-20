package sdk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestNewClient(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	if client.ActiveCount() != 0 {
		t.Errorf("新客户端 ActiveCount = %d, want 0", client.ActiveCount())
	}
}

func TestClient_Screenshot(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	result, err := client.Screenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}

	if result.Failed {
		t.Errorf("截图失败: %s", result.FailedReason)
	}
}

func TestClient_ScreenshotWithContext(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.ScreenshotWithContext(ctx, "https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("ScreenshotWithContext() error = %v", err)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestClient_ScreenshotBytes(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	imgBytes, result, err := client.ScreenshotBytes("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("ScreenshotBytes() error = %v", err)
	}

	if len(imgBytes) == 0 {
		t.Error("截图字节数据为空")
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}

	// PNG 文件头检查
	if len(imgBytes) >= 4 {
		if imgBytes[0] != 0x89 || imgBytes[1] != 'P' || imgBytes[2] != 'N' || imgBytes[3] != 'G' {
			t.Error("返回的数据不是 PNG 格式")
		}
	}
}

func TestScreenshotBytesFromResult_UsesInMemoryBytes(t *testing.T) {
	want := []byte{0x89, 'P', 'N', 'G'}
	got, err := screenshotBytesFromResult(&models.Result{
		ScreenshotBytes: want,
	})
	if err != nil {
		t.Fatalf("screenshotBytesFromResult() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("screenshotBytesFromResult() = %v, want %v", got, want)
	}
}

func TestScreenshotBytesFromResult_FallsBackToFile(t *testing.T) {
	path := t.TempDir() + "/shot.png"
	want := []byte{0x89, 'P', 'N', 'G'}
	if err := os.WriteFile(path, want, 0644); err != nil {
		t.Fatalf("write screenshot file: %v", err)
	}

	got, err := screenshotBytesFromResult(&models.Result{
		Screenshot: path,
	})
	if err != nil {
		t.Fatalf("screenshotBytesFromResult() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("screenshotBytesFromResult() = %v, want %v", got, want)
	}
}

func TestClient_BatchScreenshot(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.MaxConcurrent = 2
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	urls := []string{
		"https://www.baidu.com",
		"https://www.baidu.com",
	}

	results := client.BatchScreenshot(urls, nil)
	if len(results) != len(urls) {
		t.Fatalf("BatchScreenshot 返回 %d 个结果, 期望 %d", len(results), len(urls))
	}

	for i, r := range results {
		if r.Error != nil {
			t.Errorf("BatchScreenshot[%d] %s error: %v", i, r.URL, r.Error)
		}
		if r.Result != nil && r.Result.Title == "" {
			t.Errorf("BatchScreenshot[%d] %s 缺少页面标题", i, r.URL)
		}
	}
}

func TestClient_Stats(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	stats := client.Stats()
	if stats.Closed {
		t.Error("新客户端不应标记为关闭")
	}
	if stats.MaxConcurrent != opts.MaxConcurrent {
		t.Errorf("MaxConcurrent = %d, want %d", stats.MaxConcurrent, opts.MaxConcurrent)
	}
}

func TestClient_SetIdleTimeout(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// 设置空闲超时（不等待触发，只验证设置不报错）
	client.SetIdleTimeout(5 * time.Minute)
	client.Close()
}

func TestClient_ScreenshotWithOptions(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	screenshotOpts := &ScreenshotOptions{
		Timeout: 30 * time.Second,
	}

	result, err := client.Screenshot("https://www.baidu.com", screenshotOpts)
	if err != nil {
		t.Fatalf("Screenshot() with options error = %v", err)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestClient_CloseTwice(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	client.Close()
	// 二次 Close 不应 panic
	client.Close()
}

func TestDefaultClientOptions(t *testing.T) {
	opts := DefaultClientOptions()

	if !opts.Headless {
		t.Error("默认应为无头模式")
	}
	if opts.WindowWidth != 1280 {
		t.Errorf("默认宽度 = %d, want 1280", opts.WindowWidth)
	}
	if opts.WindowHeight != 800 {
		t.Errorf("默认高度 = %d, want 800", opts.WindowHeight)
	}
	if opts.ScreenshotFormat != "png" {
		t.Errorf("默认格式 = %s, want png", opts.ScreenshotFormat)
	}
	if opts.MaxConcurrent != 2 {
		t.Errorf("默认并发 = %d, want 2", opts.MaxConcurrent)
	}
}

func TestToRunnerOptions(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ChromePath = "/usr/bin/chromium"
	opts.IgnoreCertErrors = true
	opts.CaptureFullPage = true

	runnerOpts := toRunnerOptions(opts)

	if runnerOpts.Chrome.Path != "/usr/bin/chromium" {
		t.Errorf("ChromePath 未正确映射")
	}
	if !runnerOpts.Chrome.IgnoreCertErrors {
		t.Errorf("IgnoreCertErrors 未正确映射")
	}
	if !runnerOpts.Scan.CaptureFullPage {
		t.Errorf("CaptureFullPage 未正确映射")
	}
}

func TestMergeWithScreenshotOptions(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = "screenshots/client-default"
	base := toRunnerOptions(opts)

	so := &ScreenshotOptions{
		Timeout:         60 * time.Second,
		Selector:        "#main",
		CaptureFullPage: true,
		ScreenshotPath:  "screenshots/request-a",
		SkipSave:        true,
	}

	merged := mergeWithScreenshotOptions(base, so)

	if merged.Chrome.Timeout != 60 {
		t.Errorf("Timeout 合并后 = %d, want 60", merged.Chrome.Timeout)
	}
	if merged.Scan.Selector != "#main" {
		t.Errorf("Selector 合并后 = %s, want #main", merged.Scan.Selector)
	}
	if merged.Scan.ScreenshotPath != "screenshots/request-a" {
		t.Errorf("ScreenshotPath 合并后 = %s, want screenshots/request-a", merged.Scan.ScreenshotPath)
	}
	if !merged.Scan.CaptureFullPage {
		t.Error("CaptureFullPage 合并后应为 true")
	}
	if !merged.Scan.ScreenshotSkipSave {
		t.Error("SkipSave 合并后应为 true")
	}
}

func TestBatchScreenshot_EmptyURLs(t *testing.T) {
	c := &Client{pool: &fakeDriverPool{}, opts: DefaultClientOptions()}
	results := c.BatchScreenshot(nil, NewScreenshotOptions())
	if len(results) != 0 {
		t.Fatalf("空列表应返回 0 结果, got %d", len(results))
	}
}

func TestBatchScreenshot_ErrorAggregation(t *testing.T) {
	pool := &fakeDriverPool{err: errors.New("boom")}
	c := &Client{pool: pool, opts: DefaultClientOptions()}
	urls := []string{"http://invalid-a.test", "http://invalid-b.test"}
	results := c.BatchScreenshot(urls, NewScreenshotOptions())
	if len(results) != 2 {
		t.Fatalf("应返回 2 结果, got %d", len(results))
	}
	for i, r := range results {
		if r.URL != urls[i] {
			t.Fatalf("results[%d].URL = %q, want %q", i, r.URL, urls[i])
		}
		if r.Error == nil {
			t.Fatalf("pool 返回 error 时 results[%d] 应有 Error", i)
		}
		if r.Result != nil {
			t.Fatalf("results[%d].Result 应为 nil, got %+v", i, r.Result)
		}
	}
	if pool.calls != len(urls) {
		t.Fatalf("pool 调用次数 = %d, want %d", pool.calls, len(urls))
	}
}

func TestBatchScreenshotWithContext_PreservesOrder(t *testing.T) {
	pool := &fakeDriverPool{}
	c := &Client{pool: pool, opts: DefaultClientOptions()}
	urls := []string{"http://a.test", "http://b.test", "http://c.test"}
	results := c.BatchScreenshotWithContext(context.Background(), urls, NewScreenshotOptions())
	if len(results) != len(urls) {
		t.Fatalf("结果数 %d != 输入 %d", len(results), len(urls))
	}
	for i, r := range results {
		if r.URL != urls[i] {
			t.Fatalf("顺序错乱: results[%d].URL=%q want %q", i, r.URL, urls[i])
		}
		if r.Error != nil {
			t.Fatalf("results[%d] 不应有 Error: %v", i, r.Error)
		}
		if r.Result == nil || r.Result.URL != urls[i] {
			t.Fatalf("results[%d].Result.URL = %+v, want %q", i, r.Result, urls[i])
		}
	}
}

// newFakeClient 构造一个注入 fakeDriverPool 的 Client，无需启动浏览器，
// 用于覆盖 Screenshot* 便捷方法（薄封装）的 option 传递与返回路径。
// url 用于固定 fakeDriverPool 返回 Result 的 URL 字段，使断言可校验 URL 透传。
func newFakeClient(t *testing.T, url string) *Client {
	t.Helper()
	return &Client{
		pool: &fakeDriverPool{result: &models.Result{URL: url, Title: "ok", ScreenshotBytes: []byte("png")}},
		opts: DefaultClientOptions(),
	}
}

func TestScreenshotConvenienceMethods_Result(t *testing.T) {
	url := "https://safe-domain-example.org"
	c := newFakeClient(t, url)

	resultMethods := []struct {
		name string
		fn   func() (*models.Result, error)
	}{
		{"ScreenshotWithFormat", func() (*models.Result, error) { return c.ScreenshotWithFormat(url, "jpeg", 80, nil) }},
		{"ScreenshotToPath", func() (*models.Result, error) { return c.ScreenshotToPath(url, t.TempDir(), nil) }},
		{"ScreenshotWithDelay", func() (*models.Result, error) { return c.ScreenshotWithDelay(url, time.Second, nil) }},
		{"ScreenshotWithTimeout", func() (*models.Result, error) { return c.ScreenshotWithTimeout(url, 5*time.Second, nil) }},
		{"ScreenshotWithProxy", func() (*models.Result, error) { return c.ScreenshotWithProxy(url, "http://127.0.0.1:8080", nil) }},
		{"ScreenshotWithProxyList", func() (*models.Result, error) {
			return c.ScreenshotWithProxyList(url, runner.ProxyRoundRobin, []string{"http://a:8080"}, nil)
		}},
		{"ScreenshotWithProxyFile", func() (*models.Result, error) {
			return c.ScreenshotWithProxyFile(url, "proxies.txt", runner.ProxySequential, nil)
		}},
		{"ScreenshotWithProxyURL", func() (*models.Result, error) {
			return c.ScreenshotWithProxyURL(url, "https://proxy.example/api", runner.ProxyRandom, nil)
		}},
		{"ScreenshotWithCustomHeaders", func() (*models.Result, error) {
			return c.ScreenshotWithCustomHeaders(url, map[string]string{"X-Test": "1"}, nil)
		}},
		{"ScreenshotWithUserAgent", func() (*models.Result, error) { return c.ScreenshotWithUserAgent(url, "ua", nil) }},
		{"ScreenshotWithAcceptLanguage", func() (*models.Result, error) { return c.ScreenshotWithAcceptLanguage(url, "zh-CN", nil) }},
		{"ScreenshotWithFingerprint", func() (*models.Result, error) {
			return c.ScreenshotWithFingerprint(url, "Win32", "Google Inc.", "Intel", "Iris", nil)
		}},
		{"ScreenshotWithDeviceEmulation", func() (*models.Result, error) {
			return c.ScreenshotWithDeviceEmulation(url, 390, 844, 3, true, true, nil)
		}},
		{"ScreenshotWithMobileEmulation", func() (*models.Result, error) { return c.ScreenshotWithMobileEmulation(url, 2.5, nil) }},
		{"ScreenshotWithTouchEmulation", func() (*models.Result, error) { return c.ScreenshotWithTouchEmulation(url, false, nil) }},
		{"ScreenshotWithIgnoreCertErrors", func() (*models.Result, error) { return c.ScreenshotWithIgnoreCertErrors(url, nil) }},
		{"ScreenshotWithPlugins", func() (*models.Result, error) { return c.ScreenshotWithPlugins(url, []string{"PDF Viewer"}, nil) }},
		{"ScreenshotWithDisabledWebRTC", func() (*models.Result, error) { return c.ScreenshotWithDisabledWebRTC(url, nil) }},
		{"ScreenshotWithSpoofedScreen", func() (*models.Result, error) { return c.ScreenshotWithSpoofedScreen(url, 1920, 1080, nil) }},
		{"ScreenshotWithCookieHeader", func() (*models.Result, error) { return c.ScreenshotWithCookieHeader(url, "sid=abc", nil) }},
		{"ScreenshotWithCookieStrings", func() (*models.Result, error) { return c.ScreenshotWithCookieStrings(url, []string{"sid=abc"}, nil) }},
		{"ScreenshotWithCookieFile", func() (*models.Result, error) { return c.ScreenshotWithCookieFile(url, "cookies.json", true, nil) }},
		{"ScreenshotWithCookieImport", func() (*models.Result, error) { return c.ScreenshotWithCookieImport(url, "cookies.txt", nil) }},
		{"ScreenshotWithCookieExport", func() (*models.Result, error) { return c.ScreenshotWithCookieExport(url, "out.txt", nil) }},
		{"ScreenshotWithBlacklist", func() (*models.Result, error) { return c.ScreenshotWithBlacklist(url, []string{"*.internal.*"}, nil) }},
		{"ScreenshotWithoutBlacklist", func() (*models.Result, error) { return c.ScreenshotWithoutBlacklist(url, nil) }},
		{"ScreenshotWithDefaultBlacklist", func() (*models.Result, error) { return c.ScreenshotWithDefaultBlacklist(url, nil) }},
		{"ScreenshotWithRetries", func() (*models.Result, error) { return c.ScreenshotWithRetries(url, 3, nil) }},
		{"ScreenshotWithActions", func() (*models.Result, error) {
			return c.ScreenshotWithActions(url, []runner.InteractionAction{ActionClick("#go")}, nil)
		}},
		{"ScreenshotWithForm", func() (*models.Result, error) {
			return c.ScreenshotWithForm(url, NewForm(FormInput("#user", "admin")), nil)
		}},
		{"ScreenshotWithCookies", func() (*models.Result, error) {
			return c.ScreenshotWithCookies(url, []runner.CustomCookie{{Name: "session", Value: "abc"}}, nil)
		}},
		{"ScreenshotElement", func() (*models.Result, error) { return c.ScreenshotElement(url, "#main", nil) }},
		{"ScreenshotXPath", func() (*models.Result, error) { return c.ScreenshotXPath(url, "//main", nil) }},
		{"ScreenshotFullPage", func() (*models.Result, error) { return c.ScreenshotFullPage(url, nil) }},
		{"ScreenshotDevice", func() (*models.Result, error) { return c.ScreenshotDevice(url, "iphone-15", nil) }},
		{"ScreenshotViewport", func() (*models.Result, error) { return c.ScreenshotViewport(url, 390, 844, nil) }},
		{"ScreenshotWithJS", func() (*models.Result, error) { return c.ScreenshotWithJS(url, "window.test=true", nil) }},
		{"ScreenshotEvidence", func() (*models.Result, error) { return c.ScreenshotEvidence(url, nil) }},
		{"ScreenshotEvidenceWithContext", func() (*models.Result, error) {
			return c.ScreenshotEvidenceWithContext(context.Background(), url, nil)
		}},
	}

	for _, tc := range resultMethods {
		result, err := tc.fn()
		if err != nil {
			t.Fatalf("%s() error = %v", tc.name, err)
		}
		if result == nil || result.URL != url {
			t.Fatalf("%s() result = %+v, want URL %q", tc.name, result, url)
		}
	}
}

func TestScreenshotConvenienceMethods_Bytes(t *testing.T) {
	url := "https://safe-domain-example.org"
	c := newFakeClient(t, url)

	bytesMethods := []struct {
		name string
		fn   func() ([]byte, *models.Result, error)
	}{
		{"ScreenshotHeadersBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotHeadersBytes(url, nil) }},
		{"ScreenshotCookiesBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotCookiesBytes(url, nil) }},
		{"ScreenshotConsoleBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotConsoleBytes(url, nil) }},
		{"ScreenshotNetworkBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotNetworkBytes(url, nil) }},
		{"ScreenshotWithFormatBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithFormatBytes(url, "jpeg", 80, nil) }},
		{"ScreenshotWithDelayBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithDelayBytes(url, time.Second, nil) }},
		{"ScreenshotWithTimeoutBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithTimeoutBytes(url, 5*time.Second, nil) }},
		{"ScreenshotWithProxyBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithProxyBytes(url, "http://127.0.0.1:8080", nil)
		}},
		{"ScreenshotWithProxyListBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithProxyListBytes(url, runner.ProxyRoundRobin, []string{"http://a:8080"}, nil)
		}},
		{"ScreenshotWithProxyFileBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithProxyFileBytes(url, "proxies.txt", runner.ProxySequential, nil)
		}},
		{"ScreenshotWithProxyURLBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithProxyURLBytes(url, "https://proxy.example/api", runner.ProxyRandom, nil)
		}},
		{"ScreenshotWithCustomHeadersBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithCustomHeadersBytes(url, map[string]string{"X-Test": "1"}, nil)
		}},
		{"ScreenshotWithUserAgentBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithUserAgentBytes(url, "ua", nil) }},
		{"ScreenshotWithAcceptLanguageBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithAcceptLanguageBytes(url, "zh-CN", nil) }},
		{"ScreenshotWithFingerprintBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithFingerprintBytes(url, "Win32", "Google Inc.", "Intel", "Iris", nil)
		}},
		{"ScreenshotWithDeviceEmulationBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithDeviceEmulationBytes(url, 390, 844, 3, true, true, nil)
		}},
		{"ScreenshotWithMobileEmulationBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithMobileEmulationBytes(url, 2.5, nil) }},
		{"ScreenshotWithTouchEmulationBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithTouchEmulationBytes(url, false, nil) }},
		{"ScreenshotWithIgnoreCertErrorsBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithIgnoreCertErrorsBytes(url, nil) }},
		{"ScreenshotWithPluginsBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithPluginsBytes(url, []string{"PDF Viewer"}, nil)
		}},
		{"ScreenshotWithDisabledWebRTCBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithDisabledWebRTCBytes(url, nil) }},
		{"ScreenshotWithSpoofedScreenBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithSpoofedScreenBytes(url, 1920, 1080, nil)
		}},
		{"ScreenshotWithCookieHeaderBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithCookieHeaderBytes(url, "sid=abc", nil) }},
		{"ScreenshotWithCookieStringsBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithCookieStringsBytes(url, []string{"sid=abc"}, nil)
		}},
		{"ScreenshotWithCookieFileBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithCookieFileBytes(url, "cookies.json", true, nil)
		}},
		{"ScreenshotWithCookieImportBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithCookieImportBytes(url, "cookies.txt", nil)
		}},
		{"ScreenshotWithCookieExportBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithCookieExportBytes(url, "out.txt", nil) }},
		{"ScreenshotWithBlacklistBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithBlacklistBytes(url, []string{"*.internal.*"}, nil)
		}},
		{"ScreenshotWithoutBlacklistBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithoutBlacklistBytes(url, nil) }},
		{"ScreenshotWithDefaultBlacklistBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithDefaultBlacklistBytes(url, nil) }},
		{"ScreenshotWithRetriesBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithRetriesBytes(url, 3, nil) }},
		{"ScreenshotWithActionsBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithActionsBytes(url, []runner.InteractionAction{ActionClick("#go")}, nil)
		}},
		{"ScreenshotWithFormBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithFormBytes(url, NewForm(FormInput("#user", "admin")), nil)
		}},
		{"ScreenshotWithCookiesBytes", func() ([]byte, *models.Result, error) {
			return c.ScreenshotWithCookiesBytes(url, []runner.CustomCookie{{Name: "session", Value: "abc"}}, nil)
		}},
		{"ScreenshotElementBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotElementBytes(url, "#main", nil) }},
		{"ScreenshotXPathBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotXPathBytes(url, "//main", nil) }},
		{"ScreenshotFullPageBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotFullPageBytes(url, nil) }},
		{"ScreenshotDeviceBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotDeviceBytes(url, "iphone-15", nil) }},
		{"ScreenshotViewportBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotViewportBytes(url, 390, 844, nil) }},
		{"ScreenshotWithJSBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotWithJSBytes(url, "window.test=true", nil) }},
		{"ScreenshotEvidenceBytes", func() ([]byte, *models.Result, error) { return c.ScreenshotEvidenceBytes(url, nil) }},
		{"ScreenshotEvidenceBytesWithContext", func() ([]byte, *models.Result, error) {
			return c.ScreenshotEvidenceBytesWithContext(context.Background(), url, nil)
		}},
	}

	for _, tc := range bytesMethods {
		data, result, err := tc.fn()
		if err != nil {
			t.Fatalf("%s() error = %v", tc.name, err)
		}
		if result == nil || result.URL != url {
			t.Fatalf("%s() result = %+v, want URL %q", tc.name, result, url)
		}
		if data == nil {
			t.Fatalf("%s() data = nil, want non-nil bytes", tc.name)
		}
	}
}

func TestScreenshotAccessors_ReturnExpectedFields(t *testing.T) {
	url := "https://safe-domain-example.org"
	c := newFakeClient(t, url)

	// fakeDriverPool 返回 &models.Result{URL, Title:"ok"}，未填充 Headers/HTML 等字段，
	// 此处只验证便捷方法正确转发到 pool 并返回非 nil Result（option 设置已通过 fakeDriverPool 验证）。
	headers, result, err := c.ScreenshotHeaders(url, nil)
	if err != nil || result == nil || result.URL != url {
		t.Fatalf("ScreenshotHeaders() = %+v/%+v/%v", headers, result, err)
	}
	cookies, result, err := c.ScreenshotCookies(url, nil)
	if err != nil || result == nil || result.URL != url {
		t.Fatalf("ScreenshotCookies() = %+v/%+v/%v", cookies, result, err)
	}
	console, result, err := c.ScreenshotConsole(url, nil)
	if err != nil || result == nil || result.URL != url {
		t.Fatalf("ScreenshotConsole() = %+v/%+v/%v", console, result, err)
	}
	network, result, err := c.ScreenshotNetwork(url, nil)
	if err != nil || result == nil || result.URL != url {
		t.Fatalf("ScreenshotNetwork() = %+v/%+v/%v", network, result, err)
	}
	html, result, err := c.ScreenshotHTML(url, nil)
	if err != nil || result == nil || result.URL != url {
		t.Fatalf("ScreenshotHTML() = %q/%+v/%v", html, result, err)
	}
}

func TestScreenshotEvidenceBundle_FakePool(t *testing.T) {
	url := "https://safe-domain-example.org"
	c := newFakeClient(t, url)
	// 这里只验证不 panic 且 Result.URL 正确透传。
	bundle, result, err := c.ScreenshotEvidenceBundle(url, t.TempDir(), nil)
	_ = bundle
	_ = err
	if result == nil {
		t.Fatal("ScreenshotEvidenceBundle() result = nil")
	}
	if result.URL != url {
		t.Fatalf("result.URL = %q, want %q", result.URL, url)
	}
}

func TestScreenshotWithBlacklistFile_RealFile(t *testing.T) {
	url := "https://safe-domain-example.org"
	c := newFakeClient(t, url)

	// 空黑名单文件：NewURLBlacklist 加载成功且不拦截目标，请求继续走到 pool。
	path := filepath.Join(t.TempDir(), "blacklist.txt")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("write blacklist file: %v", err)
	}

	result, err := c.ScreenshotWithBlacklistFile(url, path, nil)
	if err != nil {
		t.Fatalf("ScreenshotWithBlacklistFile() error = %v", err)
	}
	if result == nil || result.URL != url {
		t.Fatalf("result = %+v, want URL %q", result, url)
	}

	data, result, err := c.ScreenshotWithBlacklistFileBytes(url, path, nil)
	if err != nil {
		t.Fatalf("ScreenshotWithBlacklistFileBytes() error = %v", err)
	}
	if result == nil || result.URL != url {
		t.Fatalf("bytes result = %+v, want URL %q", result, url)
	}
	if len(data) == 0 {
		t.Fatal("ScreenshotWithBlacklistFileBytes() data 为空")
	}
}
