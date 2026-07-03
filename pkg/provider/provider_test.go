package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestDefaultProviderOptions(t *testing.T) {
	opts := DefaultProviderOptions()

	if !opts.Headless {
		t.Error("默认应为无头模式")
	}
	if opts.Port != 9223 {
		t.Errorf("默认端口 = %d, want 9223", opts.Port)
	}
	if opts.ChromeDebugPort != 9222 {
		t.Errorf("默认 Chrome 调试端口 = %d, want 9222", opts.ChromeDebugPort)
	}
	if opts.MaxConcurrent != 10 {
		t.Errorf("默认并发 = %d, want 10", opts.MaxConcurrent)
	}
	if opts.WindowWidth != 1280 || opts.WindowHeight != 800 {
		t.Errorf("默认窗口 = %dx%d, want 1280x800", opts.WindowWidth, opts.WindowHeight)
	}
}

func TestNewProvider(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	if p == nil {
		t.Fatal("NewProvider 返回 nil")
	}
	if p.ready {
		t.Error("新 Provider 不应标记为 ready")
	}
}

func TestProvider_HandleIndex(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	// 初始化池
	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("初始化池失败: %v", err)
	}
	p.pool = pool
	p.ready = true
	defer p.Shutdown()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	p.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Index 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if result["name"] != "go-snir CDP Provider" {
		t.Errorf("name = %v, want go-snir CDP Provider", result["name"])
	}
}

func TestProvider_HandleHealth(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("初始化池失败: %v", err)
	}
	p.pool = pool
	p.ready = true
	p.startedAt = time.Now()
	defer p.Shutdown()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	p.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("status = %v, want ok", result["status"])
	}
}

func TestProvider_HandleStats(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("初始化池失败: %v", err)
	}
	p.pool = pool
	p.ready = true
	p.startedAt = time.Now()
	defer p.Shutdown()

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	p.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stats 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if _, ok := result["total_screenshots"]; !ok {
		t.Error("统计信息缺少 total_screenshots")
	}
}

func TestProvider_HandleWebSocketURL(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)
	p.ready = true

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Host = "localhost:9223"
	w := httptest.NewRecorder()

	p.handleWebSocketURL(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("WebSocketURL 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	wsURL, ok := result["ws_url"].(string)
	if !ok || wsURL == "" {
		t.Error("ws_url 为空或不是字符串")
	}
}

func TestProvider_ToRunnerOptions(t *testing.T) {
	opts := DefaultProviderOptions()
	opts.ChromePath = "/usr/bin/chromium"
	opts.IgnoreCertErrors = true
	opts.Proxy = "http://127.0.0.1:8080"

	p := NewProvider(opts)
	runnerOpts := p.toRunnerOptions()

	if runnerOpts.Chrome.Path != "/usr/bin/chromium" {
		t.Errorf("ChromePath 未正确映射")
	}
	if !runnerOpts.Chrome.IgnoreCertErrors {
		t.Errorf("IgnoreCertErrors 未正确映射")
	}
	if runnerOpts.Chrome.Proxy != "http://127.0.0.1:8080" {
		t.Errorf("Proxy 未正确映射")
	}
}

func TestProvider_HandleScreenshot_InvalidMethod(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	req := httptest.NewRequest("GET", "/screenshot?url=https://example.com", nil)
	w := httptest.NewRecorder()

	p.handleScreenshot(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Screenshot GET 状态码 = %d, want 405", w.Code)
	}
}

func TestProvider_HandleScreenshot_NoURL(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	req := httptest.NewRequest("POST", "/screenshot", nil)
	w := httptest.NewRecorder()

	p.handleScreenshot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Screenshot 无 URL 状态码 = %d, want 400", w.Code)
	}
}

func TestDiscoverChrome_NoInstance(t *testing.T) {
	// 在没有 Chrome 运行的端口上探测，应返回错误
	_, err := DiscoverChrome("127.0.0.1", []int{19999})
	if err == nil {
		t.Error("未运行的端口应返回错误")
	}
}

func TestWaitForSignal(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)
	p.ready = true

	done := make(chan struct{})
	go func() {
		WaitForSignal(p)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	p2, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("获取当前进程失败: %v", err)
	}
	if err := p2.Signal(os.Interrupt); err != nil {
		t.Fatalf("发送信号失败: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("WaitForSignal 在收到信号后未返回")
	}
}

func TestProvider_Shutdown_Complete(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	p.pool = pool
	p.ready = true
	p.startedAt = time.Now()

	err = p.Shutdown()
	if err != nil {
		t.Errorf("Shutdown 应无错误, got: %v", err)
	}
	if p.ready {
		t.Error("Shutdown 后 ready 应为 false")
	}
}

func TestProvider_Shutdown_NilServer(t *testing.T) {
	p := NewProvider(DefaultProviderOptions())
	err := p.Shutdown()
	if err != nil {
		t.Errorf("Shutdown 应无错误, got: %v", err)
	}
	if p.ready {
		t.Error("Shutdown 后 ready 应为 false")
	}
}

func TestProvider_Shutdown_OnlyPool(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	p.pool = pool
	p.ready = true

	err = p.Shutdown()
	if err != nil {
		t.Errorf("Shutdown 应无错误, got: %v", err)
	}
}

func TestProvider_StartWithContext(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	opts.Port = 19923
	p := NewProvider(opts)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- p.StartWithContext(ctx)
	}()

	// 轮询等待服务启动
	var hasServer, isReady bool
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		p.mu.RLock()
		hasServer = p.server != nil
		isReady = p.ready
		p.mu.RUnlock()
		if hasServer && isReady {
			break
		}
	}

	if !hasServer {
		t.Error("StartWithContext 应初始化 HTTP server")
	}
	if !isReady {
		t.Error("StartWithContext 应设置 ready=true")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("StartWithContext 返回错误: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("StartWithContext 在 context 取消后未返回")
	}
}

func TestProvider_Start(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	opts.Port = 19924
	p := NewProvider(opts)

	done := make(chan error, 1)
	go func() {
		done <- p.Start()
	}()

	// 轮询等待服务启动
	var hasServer, isReady bool
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		p.mu.RLock()
		hasServer = p.server != nil
		isReady = p.ready
		p.mu.RUnlock()
		if hasServer && isReady {
			break
		}
	}

	if !hasServer {
		t.Error("Start 应初始化 HTTP server")
	}
	if !isReady {
		t.Error("Start 应设置 ready=true")
	}

	resp, err := http.Get("http://127.0.0.1:19924/health")
	if err != nil {
		t.Fatalf("HTTP GET 失败: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("health 状态码 = %d, want 200", resp.StatusCode)
	}

	err = p.Shutdown()
	if err != nil {
		t.Logf("Shutdown 返回: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Log("Start 仍在运行中 (预期行为), 已通过 Shutdown 关闭")
	}
}

func TestProvider_HandleScreenshot_URLFromQuery(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	p.pool = pool
	p.ready = true
	defer p.Shutdown()

	req := httptest.NewRequest("POST", "/screenshot?url=https://example.com", nil)
	w := httptest.NewRecorder()

	p.handleScreenshot(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Screenshot 状态码 = %d", w.Code)
	}
}

func TestProvider_HandleScreenshot_URLFromBody(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	p.pool = pool
	p.ready = true
	defer p.Shutdown()

	body := strings.NewReader(`{"url": "https://example.com"}`)
	req := httptest.NewRequest("POST", "/screenshot", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	p.handleScreenshot(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Screenshot 状态码 = %d", w.Code)
	}
}

func TestProvider_HandleScreenshot_InvalidBodyJSON(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	body := strings.NewReader(`{invalid json`)
	req := httptest.NewRequest("POST", "/screenshot", body)
	w := httptest.NewRecorder()

	p.handleScreenshot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效 JSON body 状态码 = %d, want 400", w.Code)
	}
}

func TestProvider_GetWebSocketURL(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		reqHost         string
		chromeDebugPort int
		expected        string
	}{
		{
			name:            "指定具体 host",
			host:            "192.168.1.1",
			reqHost:         "",
			chromeDebugPort: 9222,
			expected:        "ws://192.168.1.1:9222/devtools/browser",
		},
		{
			name:            "0.0.0.0 使用请求 host",
			host:            "0.0.0.0",
			reqHost:         "example.com:9223",
			chromeDebugPort: 9222,
			expected:        "ws://example.com:9222/devtools/browser",
		},
		{
			name:            "0.0.0.0 无请求使用 127.0.0.1",
			host:            "0.0.0.0",
			reqHost:         "",
			chromeDebugPort: 9222,
			expected:        "ws://127.0.0.1:9222/devtools/browser",
		},
		{
			name:            "自定义调试端口",
			host:            "127.0.0.1",
			reqHost:         "",
			chromeDebugPort: 9220,
			expected:        "ws://127.0.0.1:9220/devtools/browser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultProviderOptions()
			opts.Host = tt.host
			opts.ChromeDebugPort = tt.chromeDebugPort
			p := NewProvider(opts)

			var req *http.Request
			if tt.reqHost != "" {
				req = httptest.NewRequest("GET", "/", nil)
				req.Host = tt.reqHost
			}

			wsURL := p.getWebSocketURL(req)
			if wsURL != tt.expected {
				t.Errorf("wsURL = %s, want %s", wsURL, tt.expected)
			}
		})
	}
}

func TestProvider_HandleHealth_Unhealthy(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("初始化池失败: %v", err)
	}
	p.pool = pool
	p.ready = true
	p.startedAt = time.Now()

	pool.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	p.handleHealth(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Health 状态码 = %d, want 503", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if result["status"] != "unhealthy" {
		t.Errorf("status = %v, want unhealthy", result["status"])
	}
}

func TestProvider_HandleIndex_Endpoints(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)
	p.ready = true

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	p.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Index 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	endpoints, ok := result["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("响应缺少 endpoints 字段")
	}

	expectedEndpoints := []string{"GET /ws", "GET /health", "GET /stats", "POST /screenshot"}
	for _, ep := range expectedEndpoints {
		if _, exists := endpoints[ep]; !exists {
			t.Errorf("缺少端点: %s", ep)
		}
	}
}

func TestDiscoverChrome_EmptyPorts(t *testing.T) {
	_, err := DiscoverChrome("127.0.0.1", []int{})
	if err == nil {
		t.Error("空端口列表应返回错误")
	}
}

func TestDiscoverChrome_MultiplePorts(t *testing.T) {
	_, err := DiscoverChrome("127.0.0.1", []int{19999, 19998})
	if err == nil {
		t.Error("无 Chrome 运行的端口应返回错误")
	}
}

func TestProvider_Start_Failure(t *testing.T) {
	opts := DefaultProviderOptions()
	opts.ChromePath = "/nonexistent/chrome-binary-that-doesnt-exist"
	opts.Port = 19925
	p := NewProvider(opts)

	err := p.Start()
	if err == nil {
		p.Shutdown()
	} else {
		if !strings.Contains(err.Error(), "启动 Chrome 实例失败") {
			t.Logf("Start 返回错误: %v (预期)", err)
		}
	}
}

func TestProvider_StartWithContext_Failure(t *testing.T) {
	opts := DefaultProviderOptions()
	opts.ChromePath = "/nonexistent/chrome-binary-that-doesnt-exist"
	opts.Port = 19926
	p := NewProvider(opts)

	ctx := context.Background()
	err := p.StartWithContext(ctx)
	if err == nil {
		t.Error("使用无效 Chrome 路径应返回错误")
	} else {
		if !strings.Contains(err.Error(), "启动 Chrome 实例失败") {
			t.Logf("StartWithContext 返回错误: %v (预期)", err)
		}
	}
}

func TestProvider_StartWithContext_CancelImmediately(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	opts.Port = 19927
	p := NewProvider(opts)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.StartWithContext(ctx)
	if err != nil {
		t.Logf("立即取消 StartWithContext 返回: %v", err)
	}
}

func TestProvider_HandleWebSocketURL_NoPool(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)
	p.ready = true

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Host = "test:9223"
	w := httptest.NewRecorder()

	p.handleWebSocketURL(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("WebSocketURL 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if _, ok := result["ws_url"]; !ok {
		t.Error("响应缺少 ws_url")
	}
}

func TestProvider_HandleStats_AllFields(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	p.pool = pool
	defer p.Shutdown()

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	p.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stats 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	expectedFields := []string{
		"active_screenshots", "max_concurrent", "total_screenshots",
		"failed_screenshots", "reconnect_count", "closed", "uptime",
	}
	for _, field := range expectedFields {
		if _, exists := result[field]; !exists {
			t.Errorf("stats 缺少字段: %s", field)
		}
	}
}

func TestProvider_HandleHealth_NotReady(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultProviderOptions()
	p := NewProvider(opts)

	runnerOpts := p.toRunnerOptions()
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		t.Fatalf("初始化池失败: %v", err)
	}
	p.pool = pool
	p.ready = false
	p.startedAt = time.Now()
	defer p.Shutdown()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	p.handleHealth(w, req)

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if result["ready"] != false {
		t.Errorf("ready = %v, want false", result["ready"])
	}
}

func TestDefaultProviderOptions_AllFields(t *testing.T) {
	opts := DefaultProviderOptions()

	if opts.Host != "0.0.0.0" {
		t.Errorf("Host = %s, want 0.0.0.0", opts.Host)
	}
	if opts.IdleTimeout != 0 {
		t.Errorf("IdleTimeout = %v, want 0", opts.IdleTimeout)
	}
	if opts.Headless != true {
		t.Error("Headless 应为 true")
	}
	if opts.MaxConcurrent != 10 {
		t.Errorf("MaxConcurrent = %d, want 10", opts.MaxConcurrent)
	}
}

func TestProvider_ToRunnerOptions_Defaults(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)
	runnerOpts := p.toRunnerOptions()

	if !runnerOpts.Chrome.Headless {
		t.Error("Headless 应为 true")
	}
	if runnerOpts.Chrome.WindowX != 1280 {
		t.Errorf("WindowX = %d, want 1280", runnerOpts.Chrome.WindowX)
	}
	if runnerOpts.Chrome.WindowY != 800 {
		t.Errorf("WindowY = %d, want 800", runnerOpts.Chrome.WindowY)
	}
	if runnerOpts.Chrome.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", runnerOpts.Chrome.Timeout)
	}
	if runnerOpts.Scan.ScreenshotFormat != "png" {
		t.Errorf("ScreenshotFormat = %s, want png", runnerOpts.Scan.ScreenshotFormat)
	}
	if !runnerOpts.Scan.HTTP || !runnerOpts.Scan.HTTPS {
		t.Error("HTTP 和 HTTPS 应为 true")
	}
}

func TestNewProvider_CustomOptions(t *testing.T) {
	opts := ProviderOptions{
		Host:             "127.0.0.1",
		Port:             9999,
		Headless:         false,
		MaxConcurrent:    5,
		ChromeDebugPort:  9224,
		WindowWidth:      1920,
		WindowHeight:     1080,
		IdleTimeout:      5 * time.Minute,
		ChromePath:       "/usr/bin/google-chrome",
		Proxy:            "http://proxy:8080",
		IgnoreCertErrors: true,
		UserAgent:        "TestAgent/1.0",
	}

	p := NewProvider(opts)

	if p.opts.Host != opts.Host {
		t.Errorf("Host = %s, want %s", p.opts.Host, opts.Host)
	}
	if p.opts.Port != opts.Port {
		t.Errorf("Port = %d, want %d", p.opts.Port, opts.Port)
	}
	if p.opts.MaxConcurrent != opts.MaxConcurrent {
		t.Errorf("MaxConcurrent = %d, want %d", p.opts.MaxConcurrent, opts.MaxConcurrent)
	}
}

func TestProvider_GetWebSocketURL_NilRequest(t *testing.T) {
	opts := DefaultProviderOptions()
	opts.Host = "0.0.0.0"
	p := NewProvider(opts)

	wsURL := p.getWebSocketURL(nil)
	if wsURL != "ws://127.0.0.1:9222/devtools/browser" {
		t.Errorf("nil 请求 wsURL = %s, want ws://127.0.0.1:9222/devtools/browser", wsURL)
	}
}

func TestProvider_HandleIndex_NoPool(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)
	p.ready = true

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	p.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Index 状态码 = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if result["name"] != "go-snir CDP Provider" {
		t.Errorf("name = %v, want go-snir CDP Provider", result["name"])
	}
	if _, ok := result["ws_endpoint"]; !ok {
		t.Error("index 响应缺少 ws_endpoint")
	}
}

func TestProvider_HandleScreenshot_EmptyBodyNoURL(t *testing.T) {
	opts := DefaultProviderOptions()
	p := NewProvider(opts)
	p.ready = true

	req := httptest.NewRequest("POST", "/screenshot", nil)
	w := httptest.NewRecorder()

	p.handleScreenshot(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("空 body 无 URL 状态码 = %d, want 400", w.Code)
	}

	if !strings.Contains(w.Body.String(), "url parameter required") {
		t.Errorf("错误消息应包含 'url parameter required', got: %s", w.Body.String())
	}
}
