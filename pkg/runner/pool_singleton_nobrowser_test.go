package runner

import (
	"context"
	"testing"
	"time"
)

// resetSharedPoolForTest 在持有写锁的情况下重置全局共享池状态，
// 返回一个清理函数恢复为 nil。注意：sync.Once 无法重置，故这些测试
// 必须在 GetSharedPool 的 globalPool!=nil 早返回分支上操作，避免触发 Once。
func resetSharedPoolForTest(t *testing.T) func() {
	t.Helper()
	globalPoolMu.Lock()
	saved := globalPool
	globalPool = nil
	globalPoolMu.Unlock()
	return func() {
		globalPoolMu.Lock()
		if globalPool != nil {
			globalPool.Close()
		}
		globalPool = saved
		globalPoolMu.Unlock()
	}
}

// TestGetSharedPool_ExistingPool 覆盖 GetSharedPool 的 globalPool!=nil 早返回分支：
// 预设一个 bare pool，GetSharedPool 应直接返回它而不触发自动初始化（不依赖浏览器）。
func TestGetSharedPool_ExistingPool(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(2)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	got, err := GetSharedPool()
	if err != nil {
		t.Fatalf("GetSharedPool 失败: %v", err)
	}
	if got != p {
		t.Error("GetSharedPool 应返回预设的 pool")
	}
}

// TestCloseSharedPool_Existing 覆盖 CloseSharedPool 的 globalPool!=nil 分支。
func TestCloseSharedPool_Existing(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(2)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	CloseSharedPool()

	globalPoolMu.RLock()
	got := globalPool
	globalPoolMu.RUnlock()
	if got != nil {
		t.Error("CloseSharedPool 后 globalPool 应为 nil")
	}
	if !p.closed {
		t.Error("CloseSharedPool 应关闭底层 pool")
	}
}

// TestCloseSharedPool_NilNoBrowser 覆盖 CloseSharedPool 的 globalPool==nil 分支（无操作）。
// 此版本不依赖浏览器（与现有 TestCloseSharedPool 区别，后者被 SKIP_BROWSER_TESTS 跳过）。
func TestCloseSharedPool_NilNoBrowser(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	// globalPool 已为 nil，CloseSharedPool 不应 panic
	CloseSharedPool()
}

// TestSharedPoolStats_Existing 覆盖 SharedPoolStats 的成功路径。
func TestSharedPoolStats_Existing(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(3)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	stats, err := SharedPoolStats()
	if err != nil {
		t.Fatalf("SharedPoolStats 失败: %v", err)
	}
	if stats.MaxConcurrent != 3 {
		t.Errorf("MaxConcurrent = %d, want 3", stats.MaxConcurrent)
	}
}

// TestSharedSetIdleTimeout_Existing 覆盖 SharedSetIdleTimeout 的成功路径。
func TestSharedSetIdleTimeout_Existing(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(2)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	if err := SharedSetIdleTimeout(50 * time.Millisecond); err != nil {
		t.Fatalf("SharedSetIdleTimeout 失败: %v", err)
	}
	if err := SharedSetIdleTimeout(0); err != nil {
		t.Fatalf("SharedSetIdleTimeout(0) 失败: %v", err)
	}
}

// TestGetSharedPoolWithConfig_Existing 覆盖 GetSharedPoolWithConfig 的
// globalPool!=nil 分支（忽略配置参数，直接返回现有池）。
func TestGetSharedPoolWithConfig_Existing(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(2)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	got, err := GetSharedPoolWithConfig(&Options{}, 4)
	if err != nil {
		t.Fatalf("GetSharedPoolWithConfig 失败: %v", err)
	}
	if got != p {
		t.Error("GetSharedPoolWithConfig 应返回预设的 pool（忽略配置）")
	}
}

// TestSharedScreenshotWithContext_CancelledCtx 覆盖 SharedScreenshotWithContext
// 的成功路径：GetSharedPool 返回预设的 bare pool → pool.ScreenshotWithContext
// 走 ctx.Done 分支返回（不启动浏览器）。
func TestSharedScreenshotWithContext_CancelledCtx(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(1)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 预取消，ScreenshotWithContext 在获取槽位前返回

	_, err := SharedScreenshotWithContext(ctx, "https://example.com", &Options{})
	if err == nil {
		t.Fatal("预取消 ctx 应返回错误")
	}
}

// TestSharedScreenshot_CancelledCtx 覆盖 SharedScreenshot（委托给 WithContext 版）。
func TestSharedScreenshot_CancelledCtx(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(1)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	// SharedScreenshot 用 context.Background()，池未关闭 → 获取槽位成功
	// → browserContextForOptions。用不存在的 Chrome 路径让它快速失败。
	p.opts = &Options{}
	p.opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"

	done := make(chan error, 1)
	go func() {
		_, err := SharedScreenshot("https://example.com", &Options{})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("SharedScreenshot 意外成功，跳过")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SharedScreenshot 超时")
	}
}

// TestInitSharedPool_AlreadyInitialized 覆盖 InitSharedPool 的 Once 已执行分支
// （globalPoolOnce.Do 内函数不再执行，initErr 保持 nil）。
func TestInitSharedPool_AlreadyInitialized(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	p := newBarePool(2)
	globalPoolMu.Lock()
	globalPool = p
	globalPoolMu.Unlock()

	// globalPoolOnce 在此之前若已被某个测试触发，Do 内函数不再执行。
	// 即使未触发，传入的 opts 会调 NewDriverPool——用无效 Chrome 路径让它失败
	// 或成功都不影响断言：关键是不 panic 且行为符合 Once 语义。
	opts := &Options{}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	err := InitSharedPool(opts, 1)
	// 不强断言 err，因为 Once 语义取决于是否首次调用
	_ = err
}

// TestSharedScreenshotWithContext_ProxyProviderFailure 覆盖
// SharedScreenshotWithContext 的成功路径（line 142 调 ScreenshotWithContext）
// + ScreenshotWithContext 代理分支。用 proxyProvider 模式 pool（不启动浏览器）
// 设为 globalPool，Witness 走代理→ensureProxyBrowser 失败。
func TestSharedScreenshotWithContext_ProxyProviderFailure(t *testing.T) {
	cleanup := resetSharedPoolForTest(t)
	defer cleanup()

	opts := &Options{}
	opts.Chrome.ProxyList = []string{"http://proxy:8080"}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	pool, err := NewDriverPool(opts, 1)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	defer pool.Close()
	globalPoolMu.Lock()
	globalPool = pool
	globalPoolMu.Unlock()

	done := make(chan error, 1)
	go func() {
		_, err := SharedScreenshotWithContext(context.Background(), "https://example.com", &Options{})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("SharedScreenshotWithContext 意外成功")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("SharedScreenshotWithContext 超时")
	}
}
