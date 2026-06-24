package sdk

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
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
