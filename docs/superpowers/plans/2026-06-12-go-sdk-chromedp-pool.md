# Go SDK + ChromeDP 连接池化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 让 go-snir 的截图能力可作为 Go API 被其他项目直接 import 调用，且多个调用方共享底层 ChromeDP 浏览器进程池，避免每次截图都启动/销毁 Chrome 进程。

**Architecture:** 其他 Go 项目 `import "github.com/cyberspacesec/snir-skills/pkg/sdk"` → `sdk.NewClient(opts)` 创建客户端 → 内部持有 `DriverPool` 连接池 → `client.Screenshot(url)` 从池中获取空闲 ChromeDP → 在已有 Chrome 进程中创建新 tab 执行截图 → 归还到池。核心设计：复用 `allocCtx`（浏览器进程级别），每次截图创建 `tabCtx`（标签页级别），截图完成后关闭标签页但保留浏览器进程。

**Tech Stack:** Go 1.23, chromedp v0.13.0, sync.Pool, context 树形取消

**Risks:**
- chromedp 的 context cancel 后不可复用 → 缓解：Pool 复用 allocCtx（浏览器进程级），每次截图创建子 tabCtx（标签页级），tab 结束不影响浏览器进程
- 并发安全 → 缓解：DriverPool 使用 channel 实现信号量模式，保证并发安全
- 浏览器进程崩溃 → 缓解：Pool 检测到崩溃后自动重建 allocCtx
- 修改 Driver 接口可能影响现有代码 → 缓解：不改 Driver 接口，Pool 实现独立的新接口

---

### Task 1: ChromeDP 连接池（DriverPool）

**Depends on:** None
**Files:**
- Create: `pkg/runner/pool.go`
- Modify: `pkg/runner/chromedp.go:20-80`（拆分 ChromeDP 创建逻辑以支持池化复用）

- [ ] **Step 1: 创建 DriverPool — 管理可复用的 Chrome 浏览器进程池**

```go
// pkg/runner/pool.go
package runner

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
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
```

- [ ] **Step 2: 重构 NewChromeDP — 提取 allocCtx 构建逻辑，复用 buildAllocOptions**

文件: `pkg/runner/chromedp.go:28-79`（替换 NewChromeDP 函数）

```go
// NewChromeDP creates a new ChromeDP driver
func NewChromeDP(opts *Options) (*ChromeDP, error) {
	// 使用共享的 allocOptions 构建逻辑
	chromedpOpts := buildAllocOptions(opts)

	// 创建Chrome上下文
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), chromedpOpts...)

	// 创建新的Chrome实例
	ctx, cancel = chromedp.NewContext(ctx)

	// 设置超时
	if opts.Chrome.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.Chrome.Timeout)*time.Second)
	}

	return &ChromeDP{
		ctx:    ctx,
		cancel: cancel,
		opts:   opts,
	}, nil
}
```

- [ ] **Step 3: 创建 DriverPool 单元测试**

```go
// pkg/runner/pool_test.go
package runner

import (
	"os"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/log"
)

func TestNewDriverPool(t *testing.T) {
	logger := log.InitLogger("debug", "text")

	opts := &Options{
		Chrome: ChromeOptions{
			Headless: true,
			WindowX:  1280,
			WindowY:  800,
			Timeout:  30,
		},
		Scan: ScanOptions{
			ScreenshotPath:   t.TempDir(),
			ScreenshotFormat: "png",
		},
	}

	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	pool, err := NewDriverPool(opts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}
	defer pool.Close()

	if pool.ActiveCount() != 0 {
		t.Errorf("新池的 ActiveCount = %d, want 0", pool.ActiveCount())
	}

	_ = logger
}

func TestDriverPool_Screenshot(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := &Options{
		Chrome: ChromeOptions{
			Headless: true,
			WindowX:  1280,
			WindowY:  800,
			Timeout:  30,
		},
		Scan: ScanOptions{
			ScreenshotPath:   t.TempDir(),
			ScreenshotFormat: "png",
		},
	}

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
	opts := &Options{
		Chrome: ChromeOptions{
			Headless: true,
			WindowX:  1280,
			WindowY:  800,
		},
		Scan: ScanOptions{
			ScreenshotPath:   t.TempDir(),
			ScreenshotFormat: "png",
		},
	}

	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

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
	opts := &Options{
		Chrome: ChromeOptions{
			Headless:         true,
			WindowX:          1920,
			WindowY:          1080,
			UserAgent:        "TestAgent",
			Proxy:            "http://127.0.0.1:8080",
			IgnoreCertErrors: true,
		},
	}

	allocOpts := buildAllocOptions(opts)

	if len(allocOpts) < 3 {
		t.Errorf("buildAllocOptions 返回选项太少: %d", len(allocOpts))
	}
}
```

- [ ] **Step 4: 验证编译和测试**
Run: `go build ./pkg/runner/... && go test ./pkg/runner/... -run TestBuildAllocOptions -count=1 -short`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

- [ ] **Step 5: 提交**
Run: `git add pkg/runner/pool.go pkg/runner/pool_test.go pkg/runner/chromedp.go && git commit -m "feat(runner): add DriverPool for ChromeDP connection reuse"`

---

### Task 2: Go SDK 客户端包

**Depends on:** Task 1
**Files:**
- Create: `pkg/sdk/client.go`
- Create: `pkg/sdk/options.go`
- Create: `pkg/sdk/client_test.go`

- [ ] **Step 1: 创建 SDK 配置选项 — 定义外部项目集成时的配置结构**

```go
// pkg/sdk/options.go
package sdk

import (
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// ClientOptions SDK 客户端配置选项
// 外部项目通过设置这些选项来初始化截图客户端
type ClientOptions struct {
	// Chrome 浏览器相关配置
	ChromePath    string        // Chrome 可执行文件路径（留空则自动查找）
	Headless      bool          // 是否使用无头模式（默认 true）
	WindowWidth   int           // 窗口宽度（默认 1280）
	WindowHeight  int           // 窗口高度（默认 800）
	Timeout       time.Duration // 页面加载超时（默认 30s）
	Delay         time.Duration // 截图前等待时间
	UserAgent     string        // 自定义 User-Agent
	Proxy         string        // 代理服务器地址

	// 连接池配置
	MaxConcurrent int           // 最大并发截图数（默认 2）

	// 截图输出配置
	ScreenshotPath   string     // 截图保存目录（默认 "screenshots"）
	ScreenshotFormat string     // 截图格式: "png" 或 "jpeg"（默认 "png"）
	SkipSave         bool       // 是否跳过保存截图到磁盘（仅返回内存数据）

	// 高级选项
	IgnoreCertErrors bool       // 忽略证书错误
	CaptureFullPage  bool       // 截取完整页面
	Selector         string     // CSS 选择器截图
	XPath            string     // XPath 截图
}

// DefaultClientOptions 返回默认的客户端配置
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Headless:         true,
		WindowWidth:      1280,
		WindowHeight:     800,
		Timeout:          30 * time.Second,
		MaxConcurrent:    2,
		ScreenshotPath:   "screenshots",
		ScreenshotFormat: "png",
		SkipSave:         false,
		IgnoreCertErrors: false,
		CaptureFullPage:  false,
	}
}

// ScreenshotOptions 单次截图的可选配置
// 可以覆盖 ClientOptions 中的部分设置
type ScreenshotOptions struct {
	Timeout       time.Duration // 本次截图超时
	Delay         time.Duration // 本次截图延迟
	UserAgent     string        // 本次截图 User-Agent
	Selector      string        // 本次截图 CSS 选择器
	XPath         string        // 本次截图 XPath
	CaptureFullPage bool        // 本次截图是否全页
	JavaScript    string        // 要执行的 JavaScript
}

// toRunnerOptions 将 ClientOptions 转换为 runner.Options
func toRunnerOptions(co ClientOptions) runner.Options {
	return runner.Options{
		Chrome: runner.ChromeOptions{
			Path:             co.ChromePath,
			Headless:         co.Headless,
			WindowX:          co.WindowWidth,
			WindowY:          co.WindowHeight,
			Timeout:          int(co.Timeout.Seconds()),
			Delay:            int(co.Delay.Seconds()),
			UserAgent:        co.UserAgent,
			Proxy:            co.Proxy,
			IgnoreCertErrors: co.IgnoreCertErrors,
		},
		Scan: runner.ScanOptions{
			ScreenshotPath:     co.ScreenshotPath,
			ScreenshotFormat:   co.ScreenshotFormat,
			ScreenshotSkipSave: co.SkipSave,
			Selector:           co.Selector,
			XPath:              co.XPath,
			CaptureFullPage:    co.CaptureFullPage,
			HTTP:               true,
			HTTPS:              true,
		},
	}
}

// mergeWithScreenshotOptions 将 ScreenshotOptions 合并到 runner.Options
func mergeWithScreenshotOptions(base runner.Options, so *ScreenshotOptions) runner.Options {
	if so == nil {
		return base
	}

	if so.Timeout > 0 {
		base.Chrome.Timeout = int(so.Timeout.Seconds())
	}
	if so.Delay > 0 {
		base.Chrome.Delay = int(so.Delay.Seconds())
	}
	if so.UserAgent != "" {
		base.Chrome.UserAgent = so.UserAgent
	}
	if so.Selector != "" {
		base.Scan.Selector = so.Selector
	}
	if so.XPath != "" {
		base.Scan.XPath = so.XPath
	}
	if so.CaptureFullPage {
		base.Scan.CaptureFullPage = true
	}
	if so.JavaScript != "" {
		base.Scan.JavaScript = so.JavaScript
		base.Scan.RunJSAfter = true
	}

	return base
}
```

- [ ] **Step 2: 创建 SDK Client — 对外提供简洁的 Go API 调用接口**

```go
// pkg/sdk/client.go
package sdk

import (
	"fmt"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// Client 是 go-snir 截图 SDK 的主入口
// 其他 Go 项目通过 import 此包来复用截图能力
// 内部持有 DriverPool，多个调用方共享同一个 Chrome 浏览器进程
//
// 使用示例:
//
//	client, _ := sdk.NewClient(sdk.DefaultClientOptions())
//	defer client.Close()
//	result, _ := client.Screenshot("https://example.com", nil)
//	fmt.Println(result.Title, result.Filename)
type Client struct {
	pool *runner.DriverPool
	opts ClientOptions
}

// NewClient 创建一个新的截图客户端
// 内部初始化 Chrome 浏览器进程池，多个截图请求复用同一浏览器进程
func NewClient(opts ClientOptions) (*Client, error) {
	runnerOpts := toRunnerOptions(opts)
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("初始化截图客户端失败: %v", err)
	}

	log.Info("截图SDK客户端已创建", "max_concurrent", opts.MaxConcurrent)
	return &Client{
		pool: pool,
		opts: opts,
	}, nil
}

// Screenshot 对指定 URL 执行截图
// url: 目标网页 URL
// screenshotOpts: 单次截图的可选配置，可覆盖客户端默认配置，传 nil 使用默认配置
// 返回截图结果，包含页面标题、截图文件路径、状态码等信息
func (c *Client) Screenshot(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	runnerOpts := toRunnerOptions(c.opts)
	runnerOpts = mergeWithScreenshotOptions(runnerOpts, screenshotOpts)

	result, err := c.pool.Screenshot(url, &runnerOpts)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	if result.Failed {
		return result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}

	return result, nil
}

// ActiveCount 返回当前正在执行的截图数
func (c *Client) ActiveCount() int {
	return c.pool.ActiveCount()
}

// Close 关闭客户端，释放浏览器进程
// 调用后客户端不可再使用
func (c *Client) Close() {
	c.pool.Close()
	log.Info("截图SDK客户端已关闭")
}
```

- [ ] **Step 3: 创建 SDK Client 测试**

```go
// pkg/sdk/client_test.go
package sdk

import (
	"os"
	"testing"
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
	opts.Timeout = 30

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
		Timeout: 30,
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
		Timeout:         60,
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
```

- [ ] **Step 4: 验证 SDK 编译和纯逻辑测试**
Run: `go build ./pkg/sdk/... && go test ./pkg/sdk/... -run "TestDefaultClientOptions|TestToRunnerOptions|TestMergeWithScreenshotOptions" -count=1`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

- [ ] **Step 5: 提交**
Run: `git add pkg/sdk/client.go pkg/sdk/options.go pkg/sdk/client_test.go && git commit -m "feat(sdk): add Go SDK client package for programmatic screenshot access"`

---

### Task 3: API Server 集成池化驱动 + 集成测试

**Depends on:** Task 1
**Files:**
- Modify: `pkg/api/server_methods.go:31-58`（使用 DriverPool 替代每次创建 ChromeDP）
- Modify: `pkg/api/types.go:147-154`（Server 结构体增加 pool 字段）
- Create: `pkg/sdk/integration_test.go`

- [ ] **Step 1: 修改 Server 结构体 — 增加 DriverPool 字段以复用浏览器进程**

文件: `pkg/api/types.go:147-154`（替换 Server 结构体）

```go
type Server struct {
	Options          ServerOptions
	Router           *mux.Router
	concurrencyLimit interface{}   // 并发限制器
	shutdownCh       chan struct{} // 关闭通道
	serverStartTime  time.Time     // 服务器启动时间
	pool             *runner.DriverPool // 浏览器连接池，复用 Chrome 进程
}
```

- [ ] **Step 2: 修改 NewServer 和 ProcessScreenshot — 初始化连接池并使用池化截图**

文件: `pkg/api/server_methods.go:14-58`（替换 NewServer 和 ProcessScreenshot）

```go
// NewServer 创建一个新的API服务器
func NewServer(options ServerOptions) *Server {
	// 初始化并发限制器
	InitConcurrencyLimiter(options.MaxConcurrentRequests, options.RequestQueueSize)

	return &Server{
		Options: options,
		Router:  mux.NewRouter(),
	}
}

// InitPool 初始化浏览器连接池
// 必须在服务器启动前调用，使用共享的 Chrome 进程处理所有截图请求
func (s *Server) InitPool(opts *runner.Options) error {
	pool, err := runner.NewDriverPool(opts, s.Options.MaxConcurrentRequests)
	if err != nil {
		return fmt.Errorf("初始化浏览器连接池失败: %v", err)
	}
	s.pool = pool
	log.Info("API服务器浏览器连接池已初始化", "max_concurrent", s.Options.MaxConcurrentRequests)
	return nil
}

// ClosePool 关闭浏览器连接池
func (s *Server) ClosePool() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// ProcessScreenshot 处理单个URL的截图请求
// 优先使用连接池，若池未初始化则回退到单次创建模式
func (s *Server) ProcessScreenshot(req ScreenshotRequest, opts runner.Options) (*models.Result, error) {
	// 优先使用连接池
	if s.pool != nil {
		result, err := s.pool.Screenshot(req.URL, &opts)
		if err != nil {
			return nil, fmt.Errorf("截图失败: %v", err)
		}
		if result.Failed {
			return nil, fmt.Errorf(result.FailedReason)
		}
		return result, nil
	}

	// 回退：连接池未初始化时使用单次模式
	driver, err := runner.NewChromeDP(&opts)
	if err != nil {
		return nil, fmt.Errorf("创建浏览器驱动失败: %v", err)
	}
	defer driver.Close()

	runnerInstance, err := runner.NewRunner(log.GetLogger(), driver, opts, nil)
	if err != nil {
		return nil, fmt.Errorf("创建截图运行器失败: %v", err)
	}
	defer runnerInstance.Close()

	result, err := driver.Witness(req.URL, &opts)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	if result.Failed {
		return nil, fmt.Errorf(result.FailedReason)
	}

	return result, nil
}
```

- [ ] **Step 3: 创建 SDK 集成测试 — 模拟外部项目调用**

```go
// pkg/sdk/integration_test.go
package sdk

import (
	"os"
	"testing"
)

// TestSDKIntegration 集成测试：模拟外部项目使用 SDK 的完整流程
// 验证：1. 创建客户端 2. 多次截图复用同一浏览器 3. 关闭客户端
func TestSDKIntegration(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	// 模拟外部项目的使用方式
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.MaxConcurrent = 2
	opts.Timeout = 30

	// 步骤 1: 创建客户端（内部启动一个 Chrome 进程）
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("步骤1 - 创建客户端失败: %v", err)
	}
	defer client.Close()

	// 步骤 2: 第一次截图
	result1, err := client.Screenshot("https://example.com", nil)
	if err != nil {
		t.Fatalf("步骤2 - 第一次截图失败: %v", err)
	}
	if result1.Title == "" {
		t.Error("步骤2 - 截图结果缺少页面标题")
	}

	// 步骤 3: 第二次截图（复用同一浏览器进程）
	result2, err := client.Screenshot("https://example.com", nil)
	if err != nil {
		t.Fatalf("步骤3 - 第二次截图失败: %v", err)
	}
	if result2.Title == "" {
		t.Error("步骤3 - 截图结果缺少页面标题")
	}

	// 步骤 4: 验证并发计数已归零
	if client.ActiveCount() != 0 {
		t.Errorf("步骤4 - ActiveCount = %d, want 0", client.ActiveCount())
	}
}

// TestSDKMultipleURLs 测试对多个不同 URL 截图
func TestSDKMultipleURLs(t *testing.T) {
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

	urls := []string{
		"https://example.com",
		"https://httpbin.org/get",
	}

	for _, url := range urls {
		result, err := client.Screenshot(url, nil)
		if err != nil {
			t.Errorf("Screenshot(%s) error = %v", url, err)
			continue
		}
		if result.Failed {
			t.Errorf("Screenshot(%s) failed: %s", url, result.FailedReason)
		}
	}
}
```

- [ ] **Step 4: 验证全量编译和测试**
Run: `go build ./... && go test ./pkg/runner/... ./pkg/sdk/... ./pkg/api/... -run "TestBuildAllocOptions|TestDefaultClientOptions|TestToRunnerOptions|TestMergeWithScreenshotOptions" -count=1`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

- [ ] **Step 5: 提交**
Run: `git add pkg/api/server_methods.go pkg/api/types.go pkg/sdk/integration_test.go && git commit -m "feat(api): integrate DriverPool into API server for Chrome process reuse"`
