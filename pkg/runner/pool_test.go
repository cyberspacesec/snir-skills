package runner

import (
	"os"
	"testing"
)

func TestNewDriverPool(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	pool, err := NewDriverPool(opts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}
	defer pool.Close()

	if pool.ActiveCount() != 0 {
		t.Errorf("新池的 ActiveCount = %d, want 0", pool.ActiveCount())
	}
}

func TestDriverPool_Screenshot(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	pool, err := NewDriverPool(opts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}
	defer pool.Close()

	result, err := pool.Screenshot("https://example.com", nil)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}

	if result.Failed {
		t.Errorf("截图失败: %s", result.FailedReason)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestDriverPool_ClosedPool(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	pool, err := NewDriverPool(opts, 1)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}

	pool.Close()

	_, err = pool.Screenshot("https://example.com", nil)
	if err == nil {
		t.Error("关闭的池应该返回错误")
	}
}

func TestBuildAllocOptions(t *testing.T) {
	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1920
	opts.Chrome.WindowY = 1080
	opts.Chrome.UserAgent = "TestAgent"
	opts.Chrome.Proxy = "http://127.0.0.1:8080"
	opts.Chrome.IgnoreCertErrors = true

	allocOpts := buildAllocOptions(opts)

	if len(allocOpts) < 3 {
		t.Errorf("buildAllocOptions 返回选项太少: %d", len(allocOpts))
	}
}
