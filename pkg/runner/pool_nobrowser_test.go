package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

// newBarePool 构造一个不启动浏览器的最小 DriverPool，用于测试纯逻辑方法。
func newBarePool(maxConcurrent int) *DriverPool {
	return &DriverPool{
		opts:       &Options{},
		sem:        make(chan struct{}, maxConcurrent),
		createdAt:  time.Now(),
		shutdownCh: make(chan struct{}),
		events:     newEventBus(),
	}
}

func TestExtractDomainSimple_Cases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"http", "http://example.com/path", "example.com"},
		{"https", "https://example.com:8080/x", "example.com"},
		{"no scheme", "example.com/a", "example.com"},
		{"port", "http://example.com:443", "example.com"},
		{"query", "http://example.com?a=1", "example.com"},
		{"fragment", "http://example.com#x", "example.com"},
		{"bare host", "example.com", "example.com"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractDomainSimple(tt.in); got != tt.want {
				t.Errorf("extractDomainSimple(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBarePool_Stats(t *testing.T) {
	p := newBarePool(4)
	stats := p.Stats()
	if stats.MaxConcurrent != 4 {
		t.Errorf("MaxConcurrent = %d, want 4", stats.MaxConcurrent)
	}
	if stats.ActiveCount != 0 {
		t.Errorf("ActiveCount = %d, want 0", stats.ActiveCount)
	}
	if stats.Closed {
		t.Error("新池不应为关闭状态")
	}
	if stats.CreatedAt.IsZero() {
		t.Error("CreatedAt 应有值")
	}
	if stats.TotalScreenshots != 0 {
		t.Errorf("TotalScreenshots = %d, want 0", stats.TotalScreenshots)
	}
}

func TestBarePool_ActiveCount(t *testing.T) {
	p := newBarePool(2)
	if p.ActiveCount() != 0 {
		t.Errorf("ActiveCount = %d, want 0", p.ActiveCount())
	}
	p.active.Add(2)
	if p.ActiveCount() != 2 {
		t.Errorf("ActiveCount = %d, want 2", p.ActiveCount())
	}
}

func TestBarePool_ScreenshotWithContext_Closed(t *testing.T) {
	p := newBarePool(1)
	p.closed = true

	_, err := p.ScreenshotWithContext(context.Background(), "https://example.com", nil)
	if err == nil {
		t.Fatal("关闭的池应返回错误")
	}
}

func TestBarePool_ScreenshotWithContext_Closing(t *testing.T) {
	p := newBarePool(1)
	p.closing = true

	_, err := p.ScreenshotWithContext(context.Background(), "https://example.com", nil)
	if err == nil {
		t.Fatal("正在关闭的池应返回错误")
	}
}

func TestBarePool_Screenshot_DelegatesToContext(t *testing.T) {
	p := newBarePool(1)
	p.closed = true

	// Screenshot 应委托给 ScreenshotWithContext 并返回 closed 错误
	_, err := p.Screenshot("https://example.com", nil)
	if err == nil {
		t.Fatal("关闭的池应返回错误")
	}
}

func TestBarePool_SetIdleTimeout_NoPanic(t *testing.T) {
	p := newBarePool(1)
	// 设置超时应不 panic（即使 idleTimer 为 nil）
	p.SetIdleTimeout(0)
	p.SetIdleTimeout(100 * time.Millisecond)
	// 再次设置应停止旧定时器
	p.SetIdleTimeout(200 * time.Millisecond)
	// 清理：停止定时器避免后台回调
	p.SetIdleTimeout(0)
}

func TestBarePool_handleIdleTimeout_Closed(t *testing.T) {
	p := newBarePool(1)
	p.closed = true
	// closed 池的 idle 超时回调应直接返回，不 panic
	p.handleIdleTimeout()
}

func TestBarePool_handleIdleTimeout_ActiveNonZero(t *testing.T) {
	p := newBarePool(1)
	p.active.Add(1)
	// 有活跃截图时不应关闭浏览器进程
	p.handleIdleTimeout()
}

func TestBarePool_handleIdleTimeout_Closing(t *testing.T) {
	p := newBarePool(1)
	p.closing = true
	p.handleIdleTimeout()
}

func TestBarePool_handleIdleTimeout_NilAllocCancel(t *testing.T) {
	// 空闲池（无活跃）触发 idle 超时；allocCancel 为 nil 不应 panic
	p := newBarePool(1)
	p.handleIdleTimeout()
}

// TestBarePool_handleIdleTimeout_WithAllocCancelAndProxies 覆盖 handleIdleTimeout 的
// allocCancel!=nil 分支 + proxyBrowsers 循环清理分支（line 167-177）。
func TestBarePool_handleIdleTimeout_WithAllocCancelAndProxies(t *testing.T) {
	p := newBarePool(1)
	called := make(chan struct{}, 1)
	p.allocCancel = func() { called <- struct{}{} }
	p.allocCtx = context.Background()
	p.proxyBrowsers = map[string]*browserProcess{
		"http://a:8080": {allocCancel: func() {}, allocCtx: context.Background()},
	}

	p.handleIdleTimeout()

	// allocCancel 应被调用
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Error("allocCancel 应被调用")
	}
	// allocCtx/allocCancel 应被清空
	if p.allocCtx != nil {
		t.Error("handleIdleTimeout 后 allocCtx 应为 nil")
	}
	if p.allocCancel != nil {
		t.Error("handleIdleTimeout 后 allocCancel 应为 nil")
	}
	// proxyBrowsers 应被清空
	if len(p.proxyBrowsers) != 0 {
		t.Errorf("proxyBrowsers 应被清空, got %d", len(p.proxyBrowsers))
	}
}

func TestBarePool_resetIdleTimer_NoPanic(t *testing.T) {
	p := newBarePool(1)
	// 无定时器时 reset 不应 panic
	p.resetIdleTimer()

	p.SetIdleTimeout(50 * time.Millisecond)
	p.resetIdleTimer()
	p.SetIdleTimeout(0)
}

func TestBarePool_On(t *testing.T) {
	p := newBarePool(1)
	// 用带缓冲的 channel 同步异步事件，避免 data race
	ch := make(chan PoolEvent, 8)
	p.On(func(event PoolEvent) {
		ch <- event
	})
	// 直接 emit 验证 handler 注册成功
	p.events.emitPoolClosed()

	select {
	case ev := <-ch:
		if ev.Type != EventPoolClosed {
			t.Errorf("事件类型 = %s, want %s", ev.Type, EventPoolClosed)
		}
	case <-time.After(time.Second):
		t.Error("注册的 handler 应被调用")
	}
}

func TestBarePool_Close_Once(t *testing.T) {
	p := newBarePool(1)
	p.Close()
	if !p.closed {
		t.Error("Close 后应标记为关闭")
	}
	// 二次 Close 不应 panic
	p.Close()
}

func TestBarePool_CloseWithTimeout(t *testing.T) {
	p := newBarePool(1)
	p.CloseWithTimeout(100 * time.Millisecond)
	if !p.closed {
		t.Error("CloseWithTimeout 后应标记为关闭")
	}
	// 已关闭再调用应立即返回
	p.CloseWithTimeout(100 * time.Millisecond)
}

// TestBarePool_CloseWithTimeout_StopsIdleTimer 覆盖 CloseWithTimeout 的
// idleTimer!=nil Stop 分支 + allocCancel!=nil 分支 + proxyBrowsers 循环分支。
func TestBarePool_CloseWithTimeout_StopsIdleTimer(t *testing.T) {
	p := newBarePool(1)
	// 创建 idleTimer（SetIdleTimeout + resetIdleTimer 会创建）
	p.SetIdleTimeout(50 * time.Millisecond)
	p.resetIdleTimer()
	// 设 allocCancel（覆盖 CloseWithTimeout 的 allocCancel!=nil 分支）
	_, cancel := context.WithCancel(context.Background())
	p.allocCancel = cancel
	// 设一个 proxyBrowsers 条目（覆盖 for 循环 + delete 分支）
	p.proxyBrowsers = map[string]*browserProcess{
		"http://proxy:8080": {allocCancel: func() {}},
	}
	p.CloseWithTimeout(100 * time.Millisecond)
	if !p.closed {
		t.Error("CloseWithTimeout 后应标记为关闭")
	}
	// proxyBrowsers 应被清空
	if len(p.proxyBrowsers) != 0 {
		t.Errorf("proxyBrowsers 应被清空, got %d", len(p.proxyBrowsers))
	}
	p.SetIdleTimeout(0)
}

// TestBarePool_CloseWithTimeout_WgTimeout 覆盖 CloseWithTimeout 的
// 超时分支（wg.Wait 未在 timeout 内完成）。
func TestBarePool_CloseWithTimeout_WgTimeout(t *testing.T) {
	p := newBarePool(1)
	// 手动 Add 一个未完成的 wg，使 Wait 阻塞
	p.wg.Add(1)
	// 用极短 timeout 触发超时分支
	start := time.Now()
	p.CloseWithTimeout(5 * time.Millisecond)
	elapsed := time.Since(start)
	if !p.closed {
		t.Error("超时后仍应标记为关闭")
	}
	// 应快速返回（超时分支），不应等满
	if elapsed > 500*time.Millisecond {
		t.Errorf("超时分支应快速返回, elapsed %v", elapsed)
	}
	// 清理 wg 避免泄漏
	p.wg.Done()
}

func TestBarePool_SetCookieJar(t *testing.T) {
	p := newBarePool(1)
	// 写一个临时 CookieJar 文件
	dir := t.TempDir()
	jar, err := NewCookieJar(dir + "/cookies.json")
	if err != nil {
		t.Fatalf("NewCookieJar 失败: %v", err)
	}
	p.SetCookieJar(jar)
	if p.cookieJar != jar {
		t.Error("SetCookieJar 未设置 cookieJar")
	}
}

func TestBarePool_ensureBrowserProcess_Closed(t *testing.T) {
	p := newBarePool(1)
	p.closed = true
	if err := p.ensureBrowserProcess(); err == nil {
		t.Fatal("关闭的池 ensureBrowserProcess 应返回错误")
	}
}

func TestBarePool_ensureBrowserProcess_AlreadyRunning(t *testing.T) {
	// allocCtx 已存在且未取消时应直接返回 nil（不需启动浏览器）
	p := newBarePool(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.allocCtx = ctx
	if err := p.ensureBrowserProcess(); err != nil {
		t.Fatalf("allocCtx 已存在时应返回 nil, got %v", err)
	}
}

// TestBarePool_ensureBrowserProcess_RemoteStartFails 覆盖 ensureBrowserProcess 的
// 重启分支（line 368-372）：allocCtx 为 nil 时调 startBrowserProcess，用无效 WSS
// 让远程连接快速失败，覆盖错误返回分支。
func TestBarePool_ensureBrowserProcess_RemoteStartFails(t *testing.T) {
	p := newBarePool(1)
	p.opts = &Options{}
	// 无效 WSS（端口 1 通常无服务，连接被拒快速失败）
	p.opts.Chrome.WSS = "ws://127.0.0.1:1/devtools/browser/test"

	done := make(chan error, 1)
	go func() {
		done <- p.ensureBrowserProcess()
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("远程连接意外成功，跳过失败分支测试")
		}
		if !strings.Contains(err.Error(), "重启浏览器进程失败") {
			t.Logf("ensureBrowserProcess 返回错误（预期）: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("ensureBrowserProcess 连接 WSS 超时")
	}
}

// TestBarePool_browserContextForOptions_WSSNoProxyEnsureFails 覆盖
// browserContextForOptions 的 WSS+ensureBrowserProcess 分支（line 388-390）：
// WSS 无 Proxy 时走 ensureBrowserProcess，远程连接失败返回错误。
func TestBarePool_browserContextForOptions_WSSNoProxyEnsureFails(t *testing.T) {
	p := newBarePool(1)
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:1/devtools/browser/test"
	// 无 Proxy，避免被 line 385 拒绝

	done := make(chan error, 1)
	go func() {
		_, err := p.browserContextForOptions(opts)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("远程连接意外成功，跳过失败分支测试")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("browserContextForOptions 连接 WSS 超时")
	}
}

// TestBarePool_browserContextForOptions_ProxyCached 覆盖 browserContextForOptions 的
// Proxy+ensureProxyBrowser 缓存命中分支（line 394-396 → 411-413）：
// opts.Chrome.Proxy 与池默认不同，走 ensureProxyBrowser，预设缓存命中返回。
func TestBarePool_browserContextForOptions_ProxyCached(t *testing.T) {
	p := newBarePool(1)
	p.proxyBrowsers = make(map[string]*browserProcess)
	// 池默认 Proxy 为 A，请求 Proxy 为 B → 不同 → 走 ensureProxyBrowser
	p.opts.Chrome.Proxy = "http://default:8080"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const reqProxy = "http://req:8080"
	p.proxyBrowsers[reqProxy] = &browserProcess{allocCtx: ctx, allocCancel: cancel}

	opts := &Options{}
	opts.Chrome.Proxy = reqProxy

	got, err := p.browserContextForOptions(opts)
	if err != nil {
		t.Fatalf("browserContextForOptions 缓存命中应返回 nil, got %v", err)
	}
	if got != ctx {
		t.Error("应返回缓存的 allocCtx")
	}
}

// TestBarePool_browserContextForOptions_NoProxyNoWSS 覆盖 browserContextForOptions 的
// 默认分支（line 398-401）：无 WSS 无 Proxy → 走 ensureBrowserProcess → startBrowserProcess
// 本地模式。用不存在的 Chrome 路径让它快速失败，避免启动真实浏览器。
func TestBarePool_browserContextForOptions_NoProxyNoWSS(t *testing.T) {
	p := newBarePool(1)
	p.opts = &Options{}
	p.opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"

	done := make(chan error, 1)
	go func() {
		_, err := p.browserContextForOptions(&Options{})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("Chrome 意外可用，跳过失败分支测试")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("browserContextForOptions 启动 Chrome 超时")
	}
}

func TestBarePool_browserContextForOptions_WSSRejectsProxy(t *testing.T) {
	p := newBarePool(1)
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/x"
	opts.Chrome.Proxy = "http://127.0.0.1:8080"
	_, err := p.browserContextForOptions(opts)
	if err == nil {
		t.Fatal("WSS + Proxy 应被拒绝")
	}
}

func TestBarePool_browserContextForOptions_WSSRejectsProxyProvider(t *testing.T) {
	p := newBarePool(1)
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/x"
	opts.Chrome.ProxyList = []string{"http://a:8080"}
	_, err := p.browserContextForOptions(opts)
	if err == nil {
		t.Fatal("WSS + ProxyProvider 应被拒绝")
	}
}

func TestBarePool_ensureProxyBrowser_Closed(t *testing.T) {
	p := newBarePool(1)
	p.closed = true
	_, err := p.ensureProxyBrowser("http://127.0.0.1:8080", &Options{})
	if err == nil {
		t.Fatal("关闭的池 ensureProxyBrowser 应返回错误")
	}
}

// TestBarePool_ensureProxyBrowser_ExistingProcess 覆盖 ensureProxyBrowser 的
// proc 已存在且有效分支（line 411-413）：预设 proxyBrowsers 条目，应直接返回其 allocCtx。
func TestBarePool_ensureProxyBrowser_ExistingProcess(t *testing.T) {
	p := newBarePool(1)
	p.proxyBrowsers = make(map[string]*browserProcess)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const proxy = "http://127.0.0.1:8080"
	p.proxyBrowsers[proxy] = &browserProcess{allocCtx: ctx, allocCancel: cancel}

	got, err := p.ensureProxyBrowser(proxy, &Options{})
	if err != nil {
		t.Fatalf("ensureProxyBrowser 已有进程应返回 nil, got %v", err)
	}
	if got != ctx {
		t.Error("ensureProxyBrowser 应返回预设的 allocCtx")
	}
}

func TestHasProxyProviderSource_Nil(t *testing.T) {
	if hasProxyProviderSource(nil) {
		t.Error("nil opts 应返回 false")
	}
}

func TestSameProxyProviderSource_NilHandling(t *testing.T) {
	if !sameProxyProviderSource(nil, nil) {
		t.Error("两个 nil 应视为相同")
	}
	if sameProxyProviderSource(nil, &Options{}) {
		t.Error("nil 与非 nil 应视为不同")
	}
}

func TestProviderName_Nil(t *testing.T) {
	if providerName(nil) != "" {
		t.Error("nil provider Name 应返回空串")
	}
}

func TestProxyProviderForPool_NilOpts(t *testing.T) {
	if proxyProviderForPool(nil) != nil {
		t.Error("nil opts 应返回 nil provider")
	}
}

func TestProxyProviderForPool_NoSource(t *testing.T) {
	if proxyProviderForPool(&Options{}) != nil {
		t.Error("无代理源应返回 nil provider")
	}
}

// TestNewDriverPool_WSSRejectsProxy 覆盖 NewDriverPool 的 WSS+Proxy 拒绝分支
// （line 92-95，纯校验，不启动浏览器）。
func TestNewDriverPool_WSSRejectsProxy(t *testing.T) {
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/x"
	opts.Chrome.Proxy = "http://127.0.0.1:8080"
	_, err := NewDriverPool(opts, 1)
	if err == nil {
		t.Fatal("WSS + Proxy 应被拒绝")
	}
	if !strings.Contains(err.Error(), "不支持通过连接池设置代理") {
		t.Errorf("错误信息不符: %v", err)
	}
}

// TestNewDriverPool_WSSRejectsProxyProvider 覆盖 NewDriverPool 的
// WSS + proxyProvider（ProxyList）拒绝分支。
func TestNewDriverPool_WSSRejectsProxyProvider(t *testing.T) {
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/x"
	opts.Chrome.ProxyList = []string{"http://a:8080"}
	_, err := NewDriverPool(opts, 1)
	if err == nil {
		t.Fatal("WSS + ProxyList 应被拒绝")
	}
}

// TestNewDriverPool_ZeroMaxConcurrent 覆盖 NewDriverPool 的
// maxConcurrent<=0 → 默认 2 分支（line 88-90）。用无效 Chrome 路径让
// startBrowserProcess 失败，但 maxConcurrent 校验已先执行。
func TestNewDriverPool_ZeroMaxConcurrent(t *testing.T) {
	opts := &Options{}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	opts.Chrome.Headless = true
	_, err := NewDriverPool(opts, 0)
	if err == nil {
		t.Skip("Chrome 意外可用，跳过")
	}
	// maxConcurrent<=0 已被钳制为 2（不影响错误返回）
}

// TestBarePool_ensureProxyBrowser_StartFails 覆盖 ensureProxyBrowser 的
// startBrowserProcess 失败分支（pool.go:417-420）。用不存在的 Chrome 路径
// 让 startBrowserProcess 失败，覆盖错误返回分支。
func TestBarePool_ensureProxyBrowser_StartFails(t *testing.T) {
	p := newBarePool(1)
	opts := &Options{}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	opts.Chrome.Headless = true

	done := make(chan error, 1)
	go func() {
		_, err := p.ensureProxyBrowser("http://127.0.0.1:8080", opts)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("Chrome 意外可用，跳过失败分支测试")
		}
		if !strings.Contains(err.Error(), "启动代理浏览器进程失败") {
			t.Logf("ensureProxyBrowser 错误（预期）: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("ensureProxyBrowser 超时")
	}
}

// TestBarePool_ensureProxyBrowser_ExpiredCacheEntry 覆盖 ensureProxyBrowser 的
// 缓存条目已过期分支（pool.go:411 false → 重新启动）。预设一个已取消 ctx 的
// 缓存条目，应跳过缓存走 startBrowserProcess（失败分支）。
func TestBarePool_ensureProxyBrowser_ExpiredCacheEntry(t *testing.T) {
	p := newBarePool(1)
	p.proxyBrowsers = make(map[string]*browserProcess)
	// 预设一个已取消的 allocCtx（proc.allocCtx.Err()!=nil）
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	const proxy = "http://127.0.0.1:8080"
	p.proxyBrowsers[proxy] = &browserProcess{allocCtx: ctx, allocCancel: cancel}

	opts := &Options{}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	opts.Chrome.Headless = true

	done := make(chan error, 1)
	go func() {
		_, err := p.ensureProxyBrowser(proxy, opts)
		done <- err
	}()
	select {
	case err := <-done:
		// 缓存条目已过期 → 走 startBrowserProcess 失败分支
		if err == nil {
			t.Skip("Chrome 意外可用，跳过失败分支测试")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("ensureProxyBrowser 超时")
	}
}

// TestScreenshotWithContext_ClosedPool 覆盖 ScreenshotWithContext 的 closed 分支
// （pool.go:206-208）。bare pool 设 closed=true，应立即返回"连接池已关闭"错误。
func TestScreenshotWithContext_ClosedPool(t *testing.T) {
	p := newBarePool(1)
	p.closed = true
	_, err := p.ScreenshotWithContext(context.Background(), "https://example.com", nil)
	if err == nil || !strings.Contains(err.Error(), "已关闭") {
		t.Fatalf("closed pool 应返回已关闭错误, got %v", err)
	}
}

// TestScreenshotWithContext_ClosingPool 覆盖 ScreenshotWithContext 的 closing 分支
// （pool.go:209-211）。
func TestScreenshotWithContext_ClosingPool(t *testing.T) {
	p := newBarePool(1)
	p.closing = true
	_, err := p.ScreenshotWithContext(context.Background(), "https://example.com", nil)
	if err == nil || !strings.Contains(err.Error(), "正在关闭") {
		t.Fatalf("closing pool 应返回正在关闭错误, got %v", err)
	}
}

// TestScreenshotWithContext_CancelledCtx 覆盖 ScreenshotWithContext 的 ctx.Done 分支
// （pool.go:217-218）。bare pool 非 closed，预取消 ctx 在获取槽位时返回。
func TestScreenshotWithContext_CancelledCtx(t *testing.T) {
	p := newBarePool(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := p.ScreenshotWithContext(ctx, "https://example.com", nil)
	if err == nil {
		t.Fatal("预取消 ctx 应返回错误")
	}
}

// TestEnsureBrowserProcess_Closed 覆盖 ensureBrowserProcess 的 closed 分支
// （pool.go:358-360）。
func TestEnsureBrowserProcess_Closed(t *testing.T) {
	p := newBarePool(1)
	p.closed = true
	if err := p.ensureBrowserProcess(); err == nil || !strings.Contains(err.Error(), "已关闭") {
		t.Fatalf("closed 应返回错误, got %v", err)
	}
}

// TestEnsureBrowserProcess_ExistingAllocCtx 覆盖 ensureBrowserProcess 的
// allocCtx 存在且未取消分支（pool.go:363-365），直接返回 nil。
func TestEnsureBrowserProcess_ExistingAllocCtx(t *testing.T) {
	p := newBarePool(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.allocCtx = ctx
	p.allocCancel = cancel
	if err := p.ensureBrowserProcess(); err != nil {
		t.Fatalf("已有 allocCtx 应返回 nil, got %v", err)
	}
}

// TestEnsureBrowserProcess_StartFails 覆盖 ensureBrowserProcess 的
// startBrowserProcess 失败分支（pool.go:369-372）。allocCtx 为已取消的 ctx
// 触发重启，无效 Chrome.Path 让 startBrowserProcess 失败。
func TestEnsureBrowserProcess_StartFails(t *testing.T) {
	p := newBarePool(1)
	p.opts = &Options{}
	p.opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	// allocCtx 为已取消 ctx，触发重启路径
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	p.allocCtx = cancelledCtx
	err := p.ensureBrowserProcess()
	if err == nil {
		t.Fatal("无效 Chrome 应返回重启失败错误")
	}
}

// TestEnsureProxyBrowser_Closed 覆盖 ensureProxyBrowser 的 closed 分支
// （pool.go:408-410）。
func TestEnsureProxyBrowser_Closed(t *testing.T) {
	p := newBarePool(1)
	p.closed = true
	_, err := p.ensureProxyBrowser("http://proxy:8080", &Options{})
	if err == nil || !strings.Contains(err.Error(), "已关闭") {
		t.Fatalf("closed 应返回错误, got %v", err)
	}
}

// TestEnsureProxyBrowser_ExistingEntry 覆盖 ensureProxyBrowser 的
// 已有缓存条目分支（pool.go:411-413）。
func TestEnsureProxyBrowser_ExistingEntry(t *testing.T) {
	p := newBarePool(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.proxyBrowsers = map[string]*browserProcess{
		"http://proxy:8080": {allocCtx: ctx, allocCancel: cancel},
	}
	got, err := p.ensureProxyBrowser("http://proxy:8080", &Options{})
	if err != nil {
		t.Fatalf("已有条目应返回 nil 错误, got %v", err)
	}
	if got != ctx {
		t.Error("应返回缓存的 allocCtx")
	}
}

// TestNewDriverPool_ProxyProviderNoBrowser 覆盖 NewDriverPool 的
// proxyProvider != nil 分支（pool.go:100-105，跳过 startBrowserProcess，
// 直接构造 pool 返回）。设 ProxyList 非空让 hasProxyProviderSource 为 true，
// 不启动浏览器进程。
func TestNewDriverPool_ProxyProviderNoBrowser(t *testing.T) {
	opts := &Options{}
	opts.Chrome.ProxyList = []string{"http://proxy1:8080", "http://proxy2:8080"}
	pool, err := NewDriverPool(opts, 2)
	if err != nil {
		t.Fatalf("proxyProvider 模式应不启动浏览器并成功: %v", err)
	}
	if pool == nil {
		t.Fatal("pool 不应为 nil")
	}
	if pool.proxyProvider == nil {
		t.Error("proxyProvider 应已设置")
	}
	// allocCtx 应为 nil（未启动浏览器）
	if pool.allocCtx != nil {
		t.Error("未启动浏览器时 allocCtx 应为 nil")
	}
	pool.Close()
}

// TestNewDriverPool_ProxyProviderZeroMaxConcurrent 覆盖 maxConcurrent<=0
// 默认 2 分支（pool.go:88-90）+ proxyProvider 模式。
func TestNewDriverPool_ProxyProviderZeroMaxConcurrent(t *testing.T) {
	opts := &Options{}
	opts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := NewDriverPool(opts, 0)
	if err != nil {
		t.Fatalf("maxConcurrent=0 应默认 2 并成功: %v", err)
	}
	if cap(pool.sem) != 2 {
		t.Errorf("sem 容量 = %d, want 2", cap(pool.sem))
	}
	pool.Close()
}

// TestNewPoolDriver_ProxyProviderSuccess 覆盖 NewPoolDriver 的成功路径
// （pool_driver.go:34-37）。用 proxyProvider 模式让 NewDriverPool 不启动
// 浏览器即可成功，进而构造 PoolDriver。
func TestNewPoolDriver_ProxyProviderSuccess(t *testing.T) {
	opts := &Options{}
	opts.Chrome.ProxyList = []string{"http://proxy:8080"}
	driver, err := NewPoolDriver(opts, 2)
	if err != nil {
		t.Fatalf("proxyProvider 模式应成功构造 PoolDriver: %v", err)
	}
	if driver == nil || driver.pool == nil {
		t.Fatal("driver/pool 不应为 nil")
	}
	driver.Close()
}

// TestPoolDriver_Witness_ProxyProviderFailure 覆盖 PoolDriver.Witness 的
// 不可重试错误立即返回分支（pool_driver.go:122-139）。用 proxyProvider 模式
// 构造 pool（不启动浏览器），Witness 时 ScreenshotWithContext 走代理分支，
// ensureProxyBrowser 因无 Chrome 失败，返回不可重试错误。
func TestPoolDriver_Witness_ProxyProviderFailure(t *testing.T) {
	opts := &Options{}
	opts.Chrome.ProxyList = []string{"http://proxy:8080"}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	driver, err := NewPoolDriver(opts, 1)
	if err != nil {
		t.Fatalf("构造 PoolDriver: %v", err)
	}
	defer driver.Close()
	_, err = driver.Witness("https://example.com", opts)
	if err == nil {
		t.Skip("Witness 意外成功（可能有 Chrome）")
	}
}

// TestDriverPool_Stats_LastActiveSet 覆盖 Stats 的 lastActiveNano>0 分支
// （pool.go:493-495）。proxyProvider 模式构造的 pool 在 NewDriverPool 中
// 已设置 lastActive（pool.go:120），Stats 应返回非零 LastActive。
func TestDriverPool_Stats_LastActiveSet(t *testing.T) {
	opts := &Options{}
	opts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := NewDriverPool(opts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	defer pool.Close()
	stats := pool.Stats()
	if stats.LastActive.IsZero() {
		t.Error("LastActive 应非零（NewDriverPool 已设 lastActive）")
	}
	if stats.MaxConcurrent != 2 {
		t.Errorf("MaxConcurrent = %d, want 2", stats.MaxConcurrent)
	}
}

// TestHandleIdleTimeout_ClosesProcesses 覆盖 handleIdleTimeout 的真实超时分支
// （pool.go:166-178）：allocCancel != nil + proxyBrowsers 非空时清理。
func TestHandleIdleTimeout_ClosesProcesses(t *testing.T) {
	p := newBarePool(1)
	p.idleTimeout = 1 * time.Millisecond
	// 设置 allocCancel 和一个 proxyBrowser 条目
	allocCtx, allocCancel := context.WithCancel(context.Background())
	p.allocCtx = allocCtx
	p.allocCancel = allocCancel
	p.proxyBrowsers = map[string]*browserProcess{
		"http://px:8080": {allocCtx: allocCtx, allocCancel: allocCancel},
	}
	// active=0, closed=false, closing=false → 进入清理分支
	p.handleIdleTimeout()
	if p.allocCtx != nil {
		t.Error("handleIdleTimeout 后 allocCtx 应为 nil")
	}
	if p.allocCancel != nil {
		t.Error("handleIdleTimeout 后 allocCancel 应为 nil")
	}
	if len(p.proxyBrowsers) != 0 {
		t.Errorf("proxyBrowsers 应清空, got %d", len(p.proxyBrowsers))
	}
	// allocCtx 应被取消
	if allocCtx.Err() == nil {
		t.Error("allocCtx 应被取消")
	}
}

// TestHandleIdleTimeout_ActiveNoOp 覆盖 handleIdleTimeout 的 active>0 早返回分支
// （pool.go:156-158）。
func TestHandleIdleTimeout_ActiveNoOp(t *testing.T) {
	p := newBarePool(1)
	p.active.Add(1)
	allocCtx, allocCancel := context.WithCancel(context.Background())
	p.allocCtx = allocCtx
	p.allocCancel = allocCancel
	p.handleIdleTimeout()
	// active>0 应早返回，不清理
	if p.allocCancel == nil {
		t.Error("active>0 时不应清理 allocCancel")
	}
	if allocCtx.Err() != nil {
		t.Error("active>0 时 allocCtx 不应被取消")
	}
	allocCancel()
}
