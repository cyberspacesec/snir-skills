package runner

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
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

func TestDriverPool_ScreenshotWithContext(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := pool.ScreenshotWithContext(ctx, "https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("ScreenshotWithContext() error = %v", err)
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

	// 下次截图应该能自动重启浏览器进程
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

func TestProxyProviderForPool(t *testing.T) {
	opts := &Options{}
	if provider := proxyProviderForPool(opts); provider != nil {
		t.Fatalf("没有轮换代理配置时不应创建 provider, got %s", provider.Name())
	}

	opts.Chrome.ProxyList = []string{"http://a:8080", "http://b:8080"}
	opts.Chrome.ProxyStrategy = ProxyRoundRobin
	provider := proxyProviderForPool(opts)
	if provider == nil {
		t.Fatal("ProxyList 应创建 provider")
	}
	if !strings.HasPrefix(provider.Name(), "proxy-list(") {
		t.Fatalf("provider.Name() = %q, want proxy-list prefix", provider.Name())
	}
}

func TestDriverPoolProxyProviderForOptions(t *testing.T) {
	poolOpts := &Options{}
	poolOpts.Chrome.ProxyList = []string{"http://pool-a:8080", "http://pool-b:8080"}
	poolOpts.Chrome.ProxyStrategy = ProxyRoundRobin
	poolProvider := proxyProviderForPool(poolOpts)
	pool := &DriverPool{opts: poolOpts, proxyProvider: poolProvider}

	same := *poolOpts
	if got := pool.proxyProviderForOptions(&same); got != poolProvider {
		t.Fatalf("same proxy source should reuse pool provider, got %v want %v", got, poolProvider)
	}

	requestList := *poolOpts
	requestList.Chrome.ProxyList = []string{"http://request-a:8080"}
	got := pool.proxyProviderForOptions(&requestList)
	if got == nil || got == poolProvider || !strings.HasPrefix(got.Name(), "proxy-list(") {
		t.Fatalf("request proxy list should create request provider, got %v", got)
	}

	requestStatic := *poolOpts
	requestStatic.Chrome.ProxyList = nil
	requestStatic.Chrome.Proxy = "http://static:8080"
	if got := pool.proxyProviderForOptions(&requestStatic); got != nil {
		t.Fatalf("static request proxy should not use pool proxy provider, got %s", got.Name())
	}

	noDefaultPool := &DriverPool{opts: &Options{}}
	requestOnly := &Options{}
	requestOnly.Chrome.ProxyList = []string{"http://request-only:8080"}
	requestOnly.Chrome.ProxyStrategy = ProxyRandom
	got = noDefaultPool.proxyProviderForOptions(requestOnly)
	if got == nil || !strings.HasPrefix(got.Name(), "proxy-list(") {
		t.Fatalf("request-only proxy list should create provider, got %v", got)
	}
}

func TestDriverPoolBrowserContextRejectsRequestProxyProviderWithWSS(t *testing.T) {
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/test"
	opts.Chrome.ProxyList = []string{"http://request:8080"}
	pool := &DriverPool{opts: &Options{}}

	if provider := pool.proxyProviderForOptions(opts); provider != nil {
		t.Fatalf("WSS request should not create proxy provider, got %s", provider.Name())
	}
	if _, err := pool.browserContextForOptions(opts); err == nil {
		t.Fatal("WSS 模式结合按请求代理池应返回错误")
	}
}

func TestNewDriverPool_RemoteWSSRejectsProxy(t *testing.T) {
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/test"
	opts.Chrome.Proxy = "http://127.0.0.1:8080"

	if _, err := NewDriverPool(opts, 1); err == nil {
		t.Fatal("WSS 模式结合静态代理应返回错误")
	}
}

func TestNewDriverPool_RemoteWSSRejectsProxyProvider(t *testing.T) {
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/test"
	opts.Chrome.ProxyList = []string{"http://127.0.0.1:8080"}

	if _, err := NewDriverPool(opts, 1); err == nil {
		t.Fatal("WSS 模式结合代理池应返回错误")
	}
}

func TestDriverPool_ConcurrentScreenshots(t *testing.T) {
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

	pool, err := NewDriverPool(opts, 3)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}
	defer pool.Close()

	// 并发3个截图
	type screenshotResult struct {
		result *models.Result
		err    error
	}

	resultsCh := make(chan screenshotResult, 3)
	for i := 0; i < 3; i++ {
		go func() {
			result, err := pool.Screenshot("https://www.baidu.com", nil)
			resultsCh <- screenshotResult{result, err}
		}()
	}

	for i := 0; i < 3; i++ {
		r := <-resultsCh
		if r.err != nil {
			t.Errorf("并发截图[%d]失败: %v", i, r.err)
		}
		if r.result != nil && r.result.Title == "" {
			t.Errorf("并发截图[%d]缺少页面标题", i)
		}
	}

	stats := pool.Stats()
	if stats.TotalScreenshots != 3 {
		t.Errorf("TotalScreenshots = %d, want 3", stats.TotalScreenshots)
	}
}
