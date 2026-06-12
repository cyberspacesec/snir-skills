// Package provider 提供 CDP Provider 服务
// 在大型系统中，多个工具/服务可能都需要 Chrome/CDP 能力
// Provider 启动一个 Chrome 实例并暴露 WebSocket URL
// 其他工具通过连接 Provider 来共享同一个 Chrome 实例，避免每个工具各启一个
//
// 架构：
//
//	┌──────────────────────┐
//	│   大系统 (多个工具)    │
//	│                      │
//	│  ┌────────┐ ┌──────┐ │
//	│  │工具 A  │ │工具B │ │
//	│  └───┬────┘ └──┬───┘ │
//	│      │         │     │
//	│      ▼         ▼     │
//	│  ┌──────────────────┐ │
//	│  │  CDP Provider    │ │  ← 本包提供
//	│  │  (1个Chrome实例) │ │
//	│  └──────────────────┘ │
//	└──────────────────────┘
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/runner"
)

// ProviderOptions CDP Provider 配置
type ProviderOptions struct {
	// Chrome 配置
	ChromePath    string // Chrome 可执行文件路径（留空则自动查找）
	Headless      bool   // 是否使用无头模式（默认 true）
	WindowWidth   int    // 窗口宽度（默认 1280）
	WindowHeight  int    // 窗口高度（默认 800）
	UserAgent     string // 自定义 User-Agent
	Proxy         string // 代理服务器地址
	IgnoreCertErrors bool // 忽略证书错误

	// Provider 服务配置
	Host string // Provider 监听地址（默认 "0.0.0.0"）
	Port int    // Provider 监听端口（默认 9223，避免与 Chrome 的 9222 冲突）

	// Chrome 远程调试端口（Provider 启动 Chrome 时的端口）
	ChromeDebugPort int // Chrome remote debugging 端口（默认 9222）

	// 连接池配置
	MaxConcurrent int // 最大并发截图数（默认 10）

	// 空闲超时
	IdleTimeout time.Duration // 浏览器空闲超时（0 表示不自动关闭）
}

// DefaultProviderOptions 返回默认配置
func DefaultProviderOptions() ProviderOptions {
	return ProviderOptions{
		Headless:        true,
		WindowWidth:     1280,
		WindowHeight:    800,
		Host:            "0.0.0.0",
		Port:            9223,
		ChromeDebugPort: 9222,
		MaxConcurrent:   10,
		IdleTimeout:     0,
	}
}

// Provider CDP Provider 服务
// 管理 Chrome 实例的生命周期，通过 HTTP API 暴露 WebSocket URL
type Provider struct {
	opts       ProviderOptions
	pool       *runner.DriverPool
	server     *http.Server
	mu         sync.RWMutex
	ready      bool
	startedAt  time.Time
	requests   int64
}

// NewProvider 创建一个新的 CDP Provider
func NewProvider(opts ProviderOptions) *Provider {
	return &Provider{
		opts: opts,
	}
}

// Start 启动 Provider 服务
// 1. 启动 Chrome 实例并创建连接池
// 2. 启动 HTTP 服务暴露 API
func (p *Provider) Start() error {
	// 创建 runner.Options
	runnerOpts := p.toRunnerOptions()

	// 创建连接池
	pool, err := runner.NewDriverPool(&runnerOpts, p.opts.MaxConcurrent)
	if err != nil {
		return fmt.Errorf("启动 Chrome 实例失败: %v", err)
	}
	p.pool = pool

	// 设置空闲超时
	if p.opts.IdleTimeout > 0 {
		p.pool.SetIdleTimeout(p.opts.IdleTimeout)
	}

	// 注册事件监听
	p.pool.On(func(event runner.PoolEvent) {
		switch event.Type {
		case runner.EventReconnect:
			log.Info("Provider: 浏览器进程已重连", "reconnect_count", event.ReconnectCount)
		case runner.EventIdleClose:
			log.Info("Provider: 浏览器空闲超时已关闭")
		case runner.EventScreenshotFailed:
			log.Warn("Provider: 截图失败", "url", event.URL, "error", event.Error)
		}
	})

	// 创建 HTTP 服务
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleIndex)
	mux.HandleFunc("/ws", p.handleWebSocketURL)
	mux.HandleFunc("/health", p.handleHealth)
	mux.HandleFunc("/stats", p.handleStats)
	mux.HandleFunc("/screenshot", p.handleScreenshot)

	p.server = &http.Server{
		Addr:    net.JoinHostPort(p.opts.Host, strconv.Itoa(p.opts.Port)),
		Handler: mux,
	}

	p.ready = true
	p.startedAt = time.Now()

	log.Info("CDP Provider 已启动",
		"host", p.opts.Host,
		"port", p.opts.Port,
		"max_concurrent", p.opts.MaxConcurrent,
		"ws_url", fmt.Sprintf("ws://%s", net.JoinHostPort(p.opts.Host, strconv.Itoa(p.opts.ChromeDebugPort))),
	)

	return p.server.ListenAndServe()
}

// StartWithContext 启动 Provider，支持 context 取消
func (p *Provider) StartWithContext(ctx context.Context) error {
	// 创建 runner.Options
	runnerOpts := p.toRunnerOptions()

	// 创建连接池
	pool, err := runner.NewDriverPool(&runnerOpts, p.opts.MaxConcurrent)
	if err != nil {
		return fmt.Errorf("启动 Chrome 实例失败: %v", err)
	}
	p.pool = pool

	if p.opts.IdleTimeout > 0 {
		p.pool.SetIdleTimeout(p.opts.IdleTimeout)
	}

	// 创建 HTTP 服务
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleIndex)
	mux.HandleFunc("/ws", p.handleWebSocketURL)
	mux.HandleFunc("/health", p.handleHealth)
	mux.HandleFunc("/stats", p.handleStats)
	mux.HandleFunc("/screenshot", p.handleScreenshot)

	p.server = &http.Server{
		Addr:    net.JoinHostPort(p.opts.Host, strconv.Itoa(p.opts.Port)),
		Handler: mux,
	}

	p.ready = true
	p.startedAt = time.Now()

	// 在 goroutine 中启动服务
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Provider HTTP 服务异常退出", "error", err)
		}
	}()

	log.Info("CDP Provider 已启动",
		"host", p.opts.Host,
		"port", p.opts.Port,
		"max_concurrent", p.opts.MaxConcurrent,
	)

	// 等待 context 取消
	<-ctx.Done()
	return p.Shutdown()
}

// Shutdown 优雅关闭 Provider
func (p *Provider) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ready = false

	// 关闭 HTTP 服务
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			log.Warn("Provider HTTP 服务关闭异常", "error", err)
		}
	}

	// 关闭连接池
	if p.pool != nil {
		p.pool.Close()
	}

	log.Info("CDP Provider 已关闭")
	return nil
}

// handleIndex 首页 - 显示 Provider 信息
func (p *Provider) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := map[string]interface{}{
		"name":         "go-snir CDP Provider",
		"version":      "1.0.0",
		"description":  "共享 Chrome/CDP 实例，供多个工具/服务复用",
		"ws_endpoint":  p.getWebSocketURL(r),
		"endpoints": map[string]string{
			"GET /ws":         "获取 WebSocket URL（用于远程连接）",
			"GET /health":     "健康检查",
			"GET /stats":      "连接池统计信息",
			"POST /screenshot": "直接截图（无需客户端）",
		},
	}

	json.NewEncoder(w).Encode(info)
}

// handleWebSocketURL 返回 Chrome 的 WebSocket URL
// 其他工具通过此 URL 连接到同一个 Chrome 实例
func (p *Provider) handleWebSocketURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	wsURL := p.getWebSocketURL(r)

	response := map[string]interface{}{
		"ws_url":       wsURL,
		"max_concurrent": p.opts.MaxConcurrent,
		"usage": map[string]string{
			"go":   `client, _ := sdk.NewRemoteClient("` + wsURL + `", 4)`,
			"curl": `curl -s http://` + r.Host + `/ws | jq .ws_url`,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// handleHealth 健康检查
func (p *Provider) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := p.pool.Stats()

	response := map[string]interface{}{
		"status":    "ok",
		"ready":     p.ready,
		"closed":    stats.Closed,
		"uptime":    time.Since(p.startedAt).String(),
	}

	if stats.Closed {
		response["status"] = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// handleStats 统计信息
func (p *Provider) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := p.pool.Stats()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_screenshots": stats.ActiveCount,
		"max_concurrent":     stats.MaxConcurrent,
		"total_screenshots":  stats.TotalScreenshots,
		"failed_screenshots": stats.FailedScreenshots,
		"reconnect_count":    stats.ReconnectCount,
		"last_active":        stats.LastActive,
		"created_at":         stats.CreatedAt,
		"closed":             stats.Closed,
		"uptime":             time.Since(p.startedAt).String(),
	})
}

// handleScreenshot 直接截图 API
// POST /screenshot?url=https://example.com
// 方便调试和快速验证
func (p *Provider) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := r.URL.Query().Get("url")
	if url == "" {
		// 尝试从 body 读取
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			var req struct {
				URL string `json:"url"`
			}
			if json.Unmarshal(body, &req) == nil && req.URL != "" {
				url = req.URL
			}
		}
	}

	if url == "" {
		http.Error(w, `{"error": "url parameter required"}`, http.StatusBadRequest)
		return
	}

	opts := p.toRunnerOptions()
	result, err := p.pool.Screenshot(url, &opts)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// getWebSocketURL 构建 WebSocket URL
func (p *Provider) getWebSocketURL(r *http.Request) string {
	// 如果有 ChromeDebugPort，构造标准的 Chrome DevTools Protocol URL
	host := p.opts.Host
	if host == "0.0.0.0" {
		// 使用请求的 Host
		if r != nil {
			h, _, _ := net.SplitHostPort(r.Host)
			if h != "" {
				host = h
			}
		}
		if host == "0.0.0.0" || host == "" {
			host = "127.0.0.1"
		}
	}
	return fmt.Sprintf("ws://%s:%d/devtools/browser", host, p.opts.ChromeDebugPort)
}

// toRunnerOptions 将 ProviderOptions 转换为 runner.Options
func (p *Provider) toRunnerOptions() runner.Options {
	opts := runner.Options{}
	opts.Chrome.Path = p.opts.ChromePath
	opts.Chrome.Headless = p.opts.Headless
	opts.Chrome.WindowX = p.opts.WindowWidth
	opts.Chrome.WindowY = p.opts.WindowHeight
	opts.Chrome.UserAgent = p.opts.UserAgent
	opts.Chrome.Proxy = p.opts.Proxy
	opts.Chrome.IgnoreCertErrors = p.opts.IgnoreCertErrors
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = "screenshots"
	opts.Scan.ScreenshotFormat = "png"
	opts.Scan.HTTP = true
	opts.Scan.HTTPS = true
	return opts
}

// WaitForSignal 等待系统信号，优雅关闭
func WaitForSignal(p *Provider) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("收到关闭信号，正在优雅关闭...")
	p.Shutdown()
}

// DiscoverChrome 尝试发现本地已运行的 Chrome 实例
// 委托给 runner.DiscoverChrome 实现
func DiscoverChrome(host string, ports []int) (string, error) {
	instance, err := runner.DiscoverChrome(host, ports)
	if err != nil {
		return "", err
	}
	return instance.WsURL, nil
}
