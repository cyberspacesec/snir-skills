package runner

import (
	"os"
	"testing"
	"time"
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

	stats := pool.Stats()
	if stats.Closed {
		t.Error("新池不应标记为关闭")
	}
	if stats.MaxConcurrent != 2 {
		t.Errorf("MaxConcurrent = %d, want 2", stats.MaxConcurrent)
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

	result, err := pool.Screenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}

	if result.Failed {
		t.Errorf("截图失败: %s", result.FailedReason)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}

	stats := pool.Stats()
	if stats.TotalScreenshots != 1 {
		t.Errorf("TotalScreenshots = %d, want 1", stats.TotalScreenshots)
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

	_, err = pool.Screenshot("https://www.baidu.com", nil)
	if err == nil {
		t.Error("关闭的池应该返回错误")
	}

	stats := pool.Stats()
	if !stats.Closed {
		t.Error("关闭的池应标记为关闭")
	}
}

func TestDriverPool_GracefulClose(t *testing.T) {
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

	// 优雅关闭不应阻塞
	pool.CloseWithTimeout(5 * time.Second)

	// 二次 Close 不应 panic
	pool.Close()
}

func TestDriverPool_Stats(t *testing.T) {
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

	pool, err := NewDriverPool(opts, 4)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}
	defer pool.Close()

	stats := pool.Stats()
	if stats.MaxConcurrent != 4 {
		t.Errorf("MaxConcurrent = %d, want 4", stats.MaxConcurrent)
	}
	if stats.ActiveCount != 0 {
		t.Errorf("初始 ActiveCount = %d, want 0", stats.ActiveCount)
	}
	if stats.CreatedAt.IsZero() {
		t.Error("CreatedAt 应有值")
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

func TestDriverPool_IdleTimeout(t *testing.T) {
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

	pool, err := NewDriverPool(opts, 1)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}
	defer pool.Close()

	// 设置空闲超时为 3 秒（测试用）
	pool.SetIdleTimeout(3 * time.Second)

	// 等待超时触发
	time.Sleep(5 * time.Second)

	// 验证：空闲超时后浏览器进程已被释放
	// 注意：可能因为时序问题不完全精确，所以只做基本检查
	// 下次截图应该能自动重启
	result, err := pool.Screenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("空闲超时后截图应能自动恢复: %v", err)
	}
	if result.Failed {
		t.Errorf("截图失败: %s", result.FailedReason)
	}

	// 验证重连次数 >= 1
	stats := pool.Stats()
	if stats.ReconnectCount < 1 {
		t.Errorf("空闲超时后应至少重连1次, ReconnectCount = %d", stats.ReconnectCount)
	}
}