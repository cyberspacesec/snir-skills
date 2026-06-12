package sdk

import (
	"os"
	"testing"
	"time"
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

	result, err := client.Screenshot("https://example.com", nil)
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

	result, err := client.Screenshot("https://example.com", screenshotOpts)
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
	base := toRunnerOptions(opts)

	so := &ScreenshotOptions{
		Timeout:         60 * time.Second,
		Selector:        "#main",
		CaptureFullPage: true,
	}

	merged := mergeWithScreenshotOptions(base, so)

	if merged.Chrome.Timeout != 60 {
		t.Errorf("Timeout 合并后 = %d, want 60", merged.Chrome.Timeout)
	}
	if merged.Scan.Selector != "#main" {
		t.Errorf("Selector 合并后 = %s, want #main", merged.Scan.Selector)
	}
	if !merged.Scan.CaptureFullPage {
		t.Error("CaptureFullPage 合并后应为 true")
	}
}
