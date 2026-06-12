package runner

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestGetSharedPool(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	// 重置全局状态
	globalPool = nil
	globalPoolOnce = resetOnce()

	pool1, err := GetSharedPool()
	if err != nil {
		t.Fatalf("GetSharedPool() error = %v", err)
	}

	pool2, err := GetSharedPool()
	if err != nil {
		t.Fatalf("GetSharedPool() 第二次 error = %v", err)
	}

	if pool1 != pool2 {
		t.Error("两次 GetSharedPool 返回了不同的实例")
	}

	// 清理
	CloseSharedPool()
}

func TestInitSharedPool(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	// 重置全局状态
	globalPool = nil
	globalPoolOnce = resetOnce()

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	err := InitSharedPool(opts, 4)
	if err != nil {
		t.Fatalf("InitSharedPool() error = %v", err)
	}

	pool, err := GetSharedPool()
	if err != nil {
		t.Fatalf("GetSharedPool() error = %v", err)
	}

	stats := pool.Stats()
	if stats.MaxConcurrent != 4 {
		t.Errorf("MaxConcurrent = %d, want 4", stats.MaxConcurrent)
	}

	CloseSharedPool()
}

func TestSharedScreenshot(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	// 重置全局状态
	globalPool = nil
	globalPoolOnce = resetOnce()

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	err := InitSharedPool(opts, 2)
	if err != nil {
		t.Fatalf("InitSharedPool() error = %v", err)
	}
	defer CloseSharedPool()

	result, err := SharedScreenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("SharedScreenshot() error = %v", err)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}

	stats, _ := SharedPoolStats()
	if stats.TotalScreenshots != 1 {
		t.Errorf("TotalScreenshots = %d, want 1", stats.TotalScreenshots)
	}
}

func TestCloseSharedPool(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	// 重置全局状态
	globalPool = nil
	globalPoolOnce = resetOnce()

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	InitSharedPool(opts, 2)
	CloseSharedPool()

	// 二次关闭不应 panic
	CloseSharedPool()
}

func TestSharedSetIdleTimeout(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	// 重置全局状态
	globalPool = nil
	globalPoolOnce = resetOnce()

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	InitSharedPool(opts, 2)
	defer CloseSharedPool()

	SharedSetIdleTimeout(5 * time.Minute)
}

// resetOnce 重置 sync.Once（测试用）
// sync.Once 不能重置，这里用替换方式
func resetOnce() sync.Once {
	return sync.Once{}
}
