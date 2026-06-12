package runner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/models"
)

// DriverPool 管理一组可复用的 Chrome 浏览器实例
// 复用 allocCtx（浏览器进程级别），每次截图创建新 tab（标签页级别）
// 截图完成后关闭 tab 但保留浏览器进程，避免反复启动 Chrome
type DriverPool struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	opts        *Options
	sem         chan struct{} // 信号量控制并发截图数
	mu          sync.Mutex
	active      atomic.Int32
	closed      bool
}

// NewDriverPool 创建一个新的 ChromeDP 连接池
// maxConcurrent: 同时执行截图的最大并发数，每个并发占用一个 Chrome tab
func NewDriverPool(opts *Options, maxConcurrent int) (*DriverPool, error) {
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}

	// 构建浏览器进程级别的 allocCtx（整个池共享一个 Chrome 进程）
	chromedpOpts := buildAllocOptions(opts)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), chromedpOpts...)

	// 预启动浏览器进程，确保可用
	ctx, cancel := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		cancel()
		allocCancel()
		return nil, fmt.Errorf("启动浏览器进程失败: %v", err)
	}
	cancel() // 关闭初始 tab，但浏览器进程保留

	pool := &DriverPool{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		opts:        opts,
		sem:         make(chan struct{}, maxConcurrent),
	}

	log.Info("浏览器连接池已创建", "max_concurrent", maxConcurrent)
	return pool, nil
}

// Screenshot 在池中的浏览器实例里执行截图
// 从池中获取一个 tab 槽位，创建新 tab 执行截图，完成后关闭 tab 释放槽位
func (p *DriverPool) Screenshot(target string, opts *Options) (*models.Result, error) {
	if p.closed {
		return nil, fmt.Errorf("连接池已关闭")
	}

	// 获取并发槽位
	p.sem <- struct{}{}
	p.active.Add(1)
	defer func() {
		<-p.sem
		p.active.Add(-1)
	}()

	// 在共享的浏览器进程中创建新 tab
	tabCtx, tabCancel := chromedp.NewContext(p.allocCtx)
	defer tabCancel()

	// 设置超时
	if opts != nil && opts.Chrome.Timeout > 0 {
		tabCtx, tabCancel = context.WithTimeout(tabCtx, time.Duration(opts.Chrome.Timeout)*time.Second)
		defer tabCancel()
	}

	// 使用传入的 opts 或池默认的 opts
	if opts == nil {
		opts = p.opts
	}

	// 创建临时 ChromeDP 实例执行截图
	driver := &ChromeDP{
		ctx:    tabCtx,
		cancel: tabCancel,
		opts:   opts,
	}

	return driver.Witness(target, opts)
}

// ActiveCount 返回当前正在执行的截图数
func (p *DriverPool) ActiveCount() int {
	return int(p.active.Load())
}

// Close 关闭连接池，释放浏览器进程
func (p *DriverPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	p.allocCancel()
	log.Info("浏览器连接池已关闭")
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