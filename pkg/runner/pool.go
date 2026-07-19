package runner

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// PoolStats 连接池统计信息
type PoolStats struct {
	ActiveCount       int       // 当前正在执行的截图数
	MaxConcurrent     int       // 最大并发数
	TotalScreenshots  int64     // 总截图次数
	FailedScreenshots int64     // 失败截图次数
	ReconnectCount    int64     // 浏览器重连次数
	LastActive        time.Time // 最后一次截图时间
	CreatedAt         time.Time // 池创建时间
	Closed            bool      // 是否已关闭
}

type browserProcess struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
}

// DriverPool 管理一组可复用的 Chrome 浏览器实例
// 复用 allocCtx（浏览器进程级别），每次截图创建新 tab（标签页级别）
// 截图完成后关闭 tab 但保留浏览器进程，避免反复启动 Chrome
//
// 支持以下高级特性：
//   - 健康检查：每次截图前验证浏览器进程可用
//   - 自动恢复：浏览器崩溃时自动重启 allocCtx
//   - 优雅关闭：等待进行中的截图完成后再关闭
//   - 空闲超时：长时间不使用自动关闭浏览器，下次使用自动重启
type DriverPool struct {
	// 浏览器进程级上下文（可被重建）
	allocCtx    context.Context
	allocCancel context.CancelFunc
	opts        *Options

	// 并发控制
	sem chan struct{} // 信号量控制并发截图数

	// 状态管理
	mu      sync.RWMutex
	active  atomic.Int32
	closed  bool
	closing bool // 正在优雅关闭中

	// 统计
	totalScreenshots  atomic.Int64
	failedScreenshots atomic.Int64
	reconnectCount    atomic.Int64
	lastActive        atomic.Int64 // UnixNano
	createdAt         time.Time

	// 空闲超时
	idleTimeout time.Duration // 0 表示不自动关闭
	idleTimer   *time.Timer
	idleMu      sync.Mutex

	// 优雅关闭
	shutdownCh chan struct{}
	wg         sync.WaitGroup // 跟踪进行中的截图

	// 事件总线
	events *eventBus

	// 代理池
	proxyProvider ProxyProvider
	proxyBrowsers map[string]*browserProcess

	// Cookie 持久化
	cookieJar *CookieJar
}

// NewDriverPool 创建一个新的 ChromeDP 连接池
// maxConcurrent: 同时执行截图的最大并发数，每个并发占用一个 Chrome tab
func NewDriverPool(opts *Options, maxConcurrent int) (*DriverPool, error) {
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}

	proxyProvider := proxyProviderForPool(opts)
	if opts.Chrome.WSS != "" && (proxyProvider != nil || opts.Chrome.Proxy != "") {
		return nil, fmt.Errorf("远程 Chrome WebSocket 模式不支持通过连接池设置代理；请在远程 Chrome 进程启动时配置代理")
	}

	var allocCtx context.Context
	var allocCancel context.CancelFunc
	var err error
	if proxyProvider == nil {
		allocCtx, allocCancel, err = startBrowserProcess(opts)
		if err != nil {
			return nil, err
		}
	}

	pool := &DriverPool{
		allocCtx:      allocCtx,
		allocCancel:   allocCancel,
		opts:          opts,
		sem:           make(chan struct{}, maxConcurrent),
		createdAt:     time.Now(),
		shutdownCh:    make(chan struct{}),
		events:        newEventBus(),
		proxyProvider: proxyProvider,
		proxyBrowsers: make(map[string]*browserProcess),
	}

	// 记录初始活跃时间
	pool.lastActive.Store(time.Now().UnixNano())

	log.Info("浏览器连接池已创建", "max_concurrent", maxConcurrent)
	return pool, nil
}

// SetIdleTimeout 设置空闲超时
// 当池空闲超过此时间后，自动关闭浏览器进程释放资源
// 下次截图时会自动重启浏览器进程
// 设为 0 表示不自动关闭（默认行为）
func (p *DriverPool) SetIdleTimeout(timeout time.Duration) {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	p.idleTimeout = timeout

	// 停止旧定时器
	if p.idleTimer != nil {
		p.idleTimer.Stop()
	}

	// 设置新定时器
	if timeout > 0 {
		p.idleTimer = time.AfterFunc(timeout, p.handleIdleTimeout)
		log.Info("浏览器连接池空闲超时已设置", "timeout", timeout)
	}
}

// handleIdleTimeout 空闲超时回调：关闭浏览器进程
func (p *DriverPool) handleIdleTimeout() {
	p.mu.RLock()
	closed := p.closed
	closing := p.closing
	active := p.active.Load()
	p.mu.RUnlock()

	if closed || closing || active > 0 {
		return
	}

	p.mu.Lock()
	if p.closed || p.closing || p.active.Load() > 0 {
		p.mu.Unlock()
		return
	}

	log.Info("浏览器连接池空闲超时，关闭浏览器进程", "timeout", p.idleTimeout)
	if p.allocCancel != nil {
		p.allocCancel()
	}
	p.allocCtx = nil
	p.allocCancel = nil
	for proxy, proc := range p.proxyBrowsers {
		if proc.allocCancel != nil {
			proc.allocCancel()
		}
		delete(p.proxyBrowsers, proxy)
	}
	p.mu.Unlock()
}

// resetIdleTimer 重置空闲定时器
func (p *DriverPool) resetIdleTimer() {
	p.idleMu.Lock()
	defer p.idleMu.Unlock()

	if p.idleTimer != nil && p.idleTimeout > 0 {
		p.idleTimer.Stop()
		p.idleTimer = time.AfterFunc(p.idleTimeout, p.handleIdleTimeout)
	}
}

// Screenshot 在池中的浏览器实例里执行截图
// 从池中获取一个 tab 槽位，创建新 tab 执行截图，完成后关闭 tab 释放槽位
func (p *DriverPool) Screenshot(target string, opts *Options) (*models.Result, error) {
	return p.ScreenshotWithContext(context.Background(), target, opts)
}

// ScreenshotWithContext 支持取消的截图
// ctx 可用于取消长时间运行的截图任务
func (p *DriverPool) ScreenshotWithContext(ctx context.Context, target string, opts *Options) (*models.Result, error) {
	p.mu.RLock()
	closed := p.closed
	closing := p.closing
	p.mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("连接池已关闭")
	}
	if closing {
		return nil, fmt.Errorf("连接池正在关闭")
	}

	// 获取并发槽位
	select {
	case p.sem <- struct{}{}:
		// 获得槽位
	case <-ctx.Done():
		return nil, fmt.Errorf("截图取消: %v", ctx.Err())
	}

	p.active.Add(1)
	p.wg.Add(1)
	p.totalScreenshots.Add(1)
	p.lastActive.Store(time.Now().UnixNano())
	p.events.emitScreenshotStart(target)

	startTime := time.Now()

	defer func() {
		<-p.sem
		p.active.Add(-1)
		p.wg.Done()
		p.resetIdleTimer()
	}()

	// 使用传入的 opts 或池默认的 opts
	if opts == nil {
		opts = p.opts
	}

	// 代理轮换：优先使用本次请求的代理源，未覆盖时使用池默认代理源。
	proxyProvider := p.proxyProviderForOptions(opts)
	if proxyProvider != nil {
		proxy, err := proxyProvider.GetProxy()
		if err != nil {
			log.Debug("获取代理失败，使用默认设置", "error", err)
		} else if proxy != "" {
			// 复制 opts 避免修改原始配置
			proxyOpts := *opts
			proxyOpts.Chrome.Proxy = proxy
			opts = &proxyOpts
			log.Debug("使用代理", "proxy", proxy, "provider", proxyProvider.Name())
		}
	}

	allocCtx, err := p.browserContextForOptions(opts)
	if err != nil {
		p.failedScreenshots.Add(1)
		duration := time.Since(startTime)
		p.events.emitScreenshotFailed(target, duration, err)
		return nil, fmt.Errorf("浏览器进程不可用: %v", err)
	}

	// 在共享的浏览器进程中创建新 tab
	tabCtx, tabCancel := chromedp.NewContext(allocCtx)
	defer tabCancel()

	// 设置超时
	if opts.Chrome.Timeout > 0 {
		var timeoutCancel context.CancelFunc
		tabCtx, timeoutCancel = context.WithTimeout(tabCtx, time.Duration(opts.Chrome.Timeout)*time.Second)
		defer timeoutCancel()
	}

	// 支持外部取消
	select {
	case <-ctx.Done():
		p.failedScreenshots.Add(1)
		duration := time.Since(startTime)
		err := fmt.Errorf("截图取消: %v", ctx.Err())
		p.events.emitScreenshotFailed(target, duration, err)
		return nil, err
	default:
	}

	// 创建临时 ChromeDP 实例执行截图
	driver := &ChromeDP{
		ctx:    tabCtx,
		cancel: tabCancel,
		opts:   opts,
	}

	result, err := driver.Witness(target, opts)
	duration := time.Since(startTime)

	if err != nil {
		p.failedScreenshots.Add(1)
		p.events.emitScreenshotFailed(target, duration, err)
		return nil, err
	}

	if result.Failed {
		p.failedScreenshots.Add(1)
		p.events.emitScreenshotFailed(target, duration, fmt.Errorf("%s", result.FailedReason))
	}

	// Cookie 写回：将浏览器获取的 Cookie 写回 CookieJar
	if p.cookieJar != nil && opts.Scan.CookieWriteBack && len(result.Cookies) > 0 {
		domain := extractDomainSimple(target)
		for _, c := range result.Cookies {
			cookieDomain := c.Domain
			if cookieDomain == "" {
				cookieDomain = domain
			}
			p.cookieJar.AddCookie(PersistentCookie{
				Name:       c.Name,
				Value:      c.Value,
				Domain:     cookieDomain,
				Path:       c.Path,
				Persistent: true,
				Source:     "session",
			})
		}
	}

	p.events.emitScreenshotComplete(target, duration, result)
	return result, nil
}

// SetCookieJar 设置 Cookie 持久化存储
func (p *DriverPool) SetCookieJar(jar *CookieJar) {
	p.cookieJar = jar
}

// extractDomainSimple 从 URL 提取域名
func extractDomainSimple(rawURL string) string {
	u := rawURL
	for _, prefix := range []string{"http://", "https://"} {
		if len(u) > len(prefix) && u[:len(prefix)] == prefix {
			u = u[len(prefix):]
			break
		}
	}
	for i, c := range u {
		if c == '/' || c == ':' || c == '?' || c == '#' {
			return u[:i]
		}
	}
	return u
}

// ensureBrowserProcess 确保浏览器进程可用
// 如果进程被空闲超时关闭了，自动重启
func (p *DriverPool) ensureBrowserProcess() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("连接池已关闭")
	}

	// 浏览器进程已在运行
	if p.allocCtx != nil && p.allocCtx.Err() == nil {
		return nil
	}

	// 浏览器进程被关闭（空闲超时或崩溃），需要重启
	log.Info("浏览器进程不可用，正在重启...")
	allocCtx, allocCancel, err := startBrowserProcess(p.opts)
	if err != nil {
		return fmt.Errorf("重启浏览器进程失败: %v", err)
	}

	p.allocCtx = allocCtx
	p.allocCancel = allocCancel
	p.reconnectCount.Add(1)
	p.events.emitReconnect(p.reconnectCount.Load())

	log.Info("浏览器进程已重启", "reconnect_count", p.reconnectCount.Load())
	return nil
}

func (p *DriverPool) browserContextForOptions(opts *Options) (context.Context, error) {
	if opts.Chrome.WSS != "" {
		if opts.Chrome.Proxy != "" || hasProxyProviderSource(opts) {
			return nil, fmt.Errorf("远程 Chrome WebSocket 模式不支持按请求设置代理")
		}
		if err := p.ensureBrowserProcess(); err != nil {
			return nil, err
		}
		return p.allocCtx, nil
	}

	if opts.Chrome.Proxy != "" && (p.proxyProvider != nil || opts.Chrome.Proxy != p.opts.Chrome.Proxy) {
		return p.ensureProxyBrowser(opts.Chrome.Proxy, opts)
	}

	if err := p.ensureBrowserProcess(); err != nil {
		return nil, err
	}
	return p.allocCtx, nil
}

func (p *DriverPool) ensureProxyBrowser(proxy string, opts *Options) (context.Context, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, fmt.Errorf("连接池已关闭")
	}
	if proc, ok := p.proxyBrowsers[proxy]; ok && proc.allocCtx != nil && proc.allocCtx.Err() == nil {
		return proc.allocCtx, nil
	}

	proxyOpts := *opts
	proxyOpts.Chrome.Proxy = proxy
	allocCtx, allocCancel, err := startBrowserProcess(&proxyOpts)
	if err != nil {
		return nil, fmt.Errorf("启动代理浏览器进程失败 (proxy=%s): %v", proxy, err)
	}
	p.proxyBrowsers[proxy] = &browserProcess{allocCtx: allocCtx, allocCancel: allocCancel}
	p.reconnectCount.Add(1)
	p.events.emitReconnect(p.reconnectCount.Load())

	log.Info("代理浏览器进程已启动", "proxy", proxy, "provider", providerName(p.proxyProvider))
	return allocCtx, nil
}

func proxyProviderForPool(opts *Options) ProxyProvider {
	if opts == nil {
		return nil
	}
	if !hasProxyProviderSource(opts) {
		return nil
	}
	return CreateProxyProvider(opts)
}

func (p *DriverPool) proxyProviderForOptions(opts *Options) ProxyProvider {
	if opts == nil {
		opts = p.opts
	}
	if opts.Chrome.WSS != "" {
		return nil
	}
	if !hasProxyProviderSource(opts) {
		return nil
	}
	if p.proxyProvider != nil && sameProxyProviderSource(p.opts, opts) {
		return p.proxyProvider
	}
	return CreateProxyProvider(opts)
}

func hasProxyProviderSource(opts *Options) bool {
	if opts == nil {
		return false
	}
	return opts.Chrome.ProxyURL != "" || opts.Chrome.ProxyFile != "" || len(opts.Chrome.ProxyList) > 0
}

func sameProxyProviderSource(a, b *Options) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Chrome.ProxyURL == b.Chrome.ProxyURL &&
		a.Chrome.ProxyFile == b.Chrome.ProxyFile &&
		a.Chrome.ProxyStrategy == b.Chrome.ProxyStrategy &&
		slices.Equal(a.Chrome.ProxyList, b.Chrome.ProxyList)
}

func providerName(provider ProxyProvider) string {
	if provider == nil {
		return ""
	}
	return provider.Name()
}

// On 注册池事件监听器
// 事件类型: screenshot_start, screenshot_complete, screenshot_failed, reconnect, idle_close, pool_closed
// 回调是异步执行的，不会阻塞主流程
func (p *DriverPool) On(handler PoolEventHandler) {
	p.events.On(handler)
}

// Stats 返回连接池统计信息
func (p *DriverPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	lastActiveNano := p.lastActive.Load()
	var lastActive time.Time
	if lastActiveNano > 0 {
		lastActive = time.Unix(0, lastActiveNano)
	}

	return PoolStats{
		ActiveCount:       int(p.active.Load()),
		MaxConcurrent:     cap(p.sem),
		TotalScreenshots:  p.totalScreenshots.Load(),
		FailedScreenshots: p.failedScreenshots.Load(),
		ReconnectCount:    p.reconnectCount.Load(),
		LastActive:        lastActive,
		CreatedAt:         p.createdAt,
		Closed:            p.closed,
	}
}

// ActiveCount 返回当前正在执行的截图数
func (p *DriverPool) ActiveCount() int {
	return int(p.active.Load())
}

// Close 关闭连接池，释放浏览器进程
// 如果有进行中的截图，等待它们完成
func (p *DriverPool) Close() {
	p.CloseWithTimeout(30 * time.Second)
}

// CloseWithTimeout 带超时的优雅关闭
// timeout: 等待进行中截图完成的最大时间
func (p *DriverPool) CloseWithTimeout(timeout time.Duration) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}

	p.closing = true
	p.mu.Unlock()

	// 停止空闲定时器
	p.idleMu.Lock()
	if p.idleTimer != nil {
		p.idleTimer.Stop()
	}
	p.idleMu.Unlock()

	// 等待进行中的截图完成
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("所有进行中的截图已完成")
	case <-time.After(timeout):
		log.Warn("等待截图完成超时，强制关闭", "timeout", timeout)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	if p.allocCancel != nil {
		p.allocCancel()
	}
	for proxy, proc := range p.proxyBrowsers {
		if proc.allocCancel != nil {
			proc.allocCancel()
		}
		delete(p.proxyBrowsers, proxy)
	}
	log.Info("浏览器连接池已关闭")
	p.events.emitPoolClosed()
}

// startBrowserProcess 启动一个新的浏览器进程
// 支持两种模式：
// 1. 本地模式（opts.Chrome.WSS 为空）：启动本地 Chrome 进程
// 2. 远程模式（opts.Chrome.WSS 不为空）：连接到已有的远程 Chrome 实例
// 返回 allocCtx, allocCancel, error
func startBrowserProcess(opts *Options) (context.Context, context.CancelFunc, error) {
	// 远程模式：通过 WebSocket URL 连接已有的 Chrome 实例
	if opts.Chrome.WSS != "" {
		allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), opts.Chrome.WSS)

		// 验证远程连接可用
		ctx, cancel := chromedp.NewContext(allocCtx)
		if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
			cancel()
			allocCancel()
			return nil, nil, fmt.Errorf("连接远程浏览器失败 (wsURL=%s): %v", opts.Chrome.WSS, err)
		}
		cancel()

		log.Info("已连接到远程浏览器", "ws_url", opts.Chrome.WSS)
		return allocCtx, allocCancel, nil
	}

	// 本地模式：启动新的 Chrome 进程
	chromedpOpts := buildAllocOptions(opts)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), chromedpOpts...)

	// 预启动浏览器进程，确保可用
	ctx, cancel := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		cancel()
		allocCancel()
		return nil, nil, fmt.Errorf("启动浏览器进程失败: %v", err)
	}
	cancel() // 关闭初始 tab，但浏览器进程保留

	return allocCtx, allocCancel, nil
}

// buildAllocOptions 构建 chromedp ExecAllocator 选项
// 将 Options 中的 Chrome 配置转换为 chromedp 选项
func buildAllocOptions(opts *Options) []chromedp.ExecAllocatorOption {
	chromedpOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
	}

	if opts.Chrome.Headless {
		chromedpOpts = append(chromedpOpts, chromedp.Headless)
	}

	chromedpOpts = append(chromedpOpts, chromedp.WindowSize(opts.Chrome.WindowX, opts.Chrome.WindowY))

	if opts.Chrome.UserAgent != "" {
		chromedpOpts = append(chromedpOpts, chromedp.UserAgent(opts.Chrome.UserAgent))
	}

	if opts.Chrome.Proxy != "" {
		chromedpOpts = append(chromedpOpts, chromedp.ProxyServer(opts.Chrome.Proxy))
	}

	if opts.Chrome.Path != "" {
		chromedpOpts = append(chromedpOpts, chromedp.ExecPath(opts.Chrome.Path))
	}

	if opts.Chrome.IgnoreCertErrors {
		chromedpOpts = append(chromedpOpts, chromedp.Flag("ignore-certificate-errors", true))
	}

	return chromedpOpts
}
