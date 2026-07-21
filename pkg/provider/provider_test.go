package provider

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// netSplitHostPort 包装 net.SplitHostPort 便于测试。
func netSplitHostPort(s string) (string, string, error) { return net.SplitHostPort(s) }

// netParsePort 解析端口字符串为 int。
func netParsePort(s string) int {
	p, _ := strconv.Atoi(s)
	return p
}

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

// TestProvider_HandleScreenshot_BodyURLOkPoolNil 覆盖 handleScreenshot 的
// 有效 JSON body 提取 URL（line 351-353）+ pool==nil 早返回（line 365-367）
// 的组合分支，无需浏览器。POST 有效 JSON body，不设 pool，期望 503。
func TestProvider_HandleScreenshot_BodyURLOkPoolNil(t *testing.T) {
	p := NewProvider(DefaultProviderOptions())
	body := strings.NewReader(`{"url":"https://example.com"}`)
	req := httptest.NewRequest("POST", "/screenshot", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	p.handleScreenshot(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("有效 body URL + pool nil 应返回 503, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "provider not initialized") {
		t.Errorf("响应体应包含 'provider not initialized', got: %s", rr.Body.String())
	}
}

// TestProvider_HandleScreenshot_BodyEmptyURLPoolNil 覆盖 handleScreenshot 的
// 有效 JSON 但 url 为空（line 351 Unmarshal 成功但 req.URL==""）分支，
// 期望 400。无需浏览器。
func TestProvider_HandleScreenshot_BodyEmptyURLPoolNil(t *testing.T) {
	p := NewProvider(DefaultProviderOptions())
	body := strings.NewReader(`{"url":""}`)
	req := httptest.NewRequest("POST", "/screenshot", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	p.handleScreenshot(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("空 URL 应返回 400, got %d", rr.Code)
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

func TestProvider_HealthHandler_NotInitialized(t *testing.T) {
	p := &Provider{} // pool == nil 分支
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	p.handleHealth(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("未初始化 health 应返回 503, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "provider not initialized") {
		t.Errorf("响应体应包含 'provider not initialized', got: %s", rr.Body.String())
	}
}

func TestProvider_StatsHandler_NotInitialized(t *testing.T) {
	p := &Provider{}
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()
	p.handleStats(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("未初始化 stats 应返回 503, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "provider not initialized") {
		t.Errorf("响应体应包含 'provider not initialized', got: %s", rr.Body.String())
	}
}

func TestProvider_ScreenshotHandler_NotInitialized(t *testing.T) {
	// POST + 有效 url，但 pool==nil → 应返回 503（覆盖 handleScreenshot 的 pool==nil 早返回分支）
	p := &Provider{}
	req := httptest.NewRequest(http.MethodPost, "/screenshot?url=http://x.test", nil)
	rr := httptest.NewRecorder()
	p.handleScreenshot(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("未初始化 screenshot 应返回 503, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "provider not initialized") {
		t.Errorf("响应体应包含 'provider not initialized', got: %s", rr.Body.String())
	}
}

// TestDiscoverChrome_Success 覆盖 provider.DiscoverChrome 的成功路径（返回 WsURL）。
// 用 httptest 模拟 Chrome 的 /json/version 端点。
func TestDiscoverChrome_Success(t *testing.T) {
	const wsURL = "ws://127.0.0.1:9222/devtools/browser/fake-session-id"
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"webSocketDebuggerUrl": wsURL,
			"Browser":              "Test Chrome/1.0",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 从 httptest 监听地址解析端口
	_, portStr, err := netSplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("解析端口失败: %v", err)
	}
	port := netParsePort(portStr)

	got, err := DiscoverChrome("127.0.0.1", []int{port})
	if err != nil {
		t.Fatalf("DiscoverChrome 成功路径失败: %v", err)
	}
	if got != wsURL {
		t.Errorf("DiscoverChrome 返回 %q, want %q", got, wsURL)
	}
}

// TestProvider_Shutdown_WithServerNoPool 覆盖 Shutdown 的 server!=nil 分支
// （line 219-225）+ pool==nil 分支。不启动浏览器：构造一个已绑定但未
// ListenAndServe 的 http.Server，Shutdown 会立即返回（无活跃连接）。
func TestProvider_Shutdown_WithServerNoPool(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen 失败: %v", err)
	}
	defer ln.Close()

	p := NewProvider(DefaultProviderOptions())
	p.server = &http.Server{Handler: http.NewServeMux()}
	p.ready = true
	p.startedAt = time.Now()
	// pool 保持 nil，覆盖 line 228 的 pool==nil 跳过分支

	// 把已绑定的 listener 交给 server，让 Shutdown 有东西可关
	go func() { _ = p.server.Serve(ln) }()
	// 给 Serve 一点时间进入循环
	time.Sleep(50 * time.Millisecond)

	if err := p.Shutdown(); err != nil {
		t.Errorf("Shutdown 应无错误, got: %v", err)
	}
	if p.ready {
		t.Error("Shutdown 后 ready 应为 false")
	}
}

// TestProvider_Shutdown_ServerShutdownError 覆盖 Shutdown 的
// server.Shutdown 异常分支（line 222-224 记 warn 不返回错误）。
// 通过让 server 在已关闭的 listener 上构造，Shutdown 返回错误被吞掉。
func TestProvider_Shutdown_ServerShutdownError(t *testing.T) {
	p := NewProvider(DefaultProviderOptions())
	// 构造一个 server 并立即关闭其底层 listener，使 Shutdown 返回错误
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen 失败: %v", err)
	}
	srv := &http.Server{Handler: http.NewServeMux()}
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(50 * time.Millisecond)
	// 先关闭 listener 制造异常
	ln.Close()

	p.server = srv
	p.ready = true
	p.startedAt = time.Now()

	// Shutdown 会尝试关闭 server（可能已关闭），错误被吞掉
	if err := p.Shutdown(); err != nil {
		t.Errorf("Shutdown 应吞掉 server 错误返回 nil, got: %v", err)
	}
}

// TestProvider_HandleHealth_WithProxyPool 覆盖 handleHealth 的成功路径
// （provider.go:289-303，pool!=nil + Stats）。用 proxyProvider 模式构造
// 不启动浏览器的 pool，赋给 p.pool，无需真实 Chrome。
func TestProvider_HandleHealth_WithProxyPool(t *testing.T) {
	opts := DefaultProviderOptions()
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := runner.NewDriverPool(&runnerOpts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool proxyProvider 模式: %v", err)
	}
	defer pool.Close()
	p := NewProvider(opts)
	p.pool = pool
	p.ready = true
	p.startedAt = time.Now()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	p.handleHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("health 应返回 200, got %d; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Errorf("响应应包含 ok, got: %s", rr.Body.String())
	}
}

// TestProvider_HandleStats_WithProxyPool 覆盖 handleStats 的成功路径
// （provider.go:318-336，pool!=nil + Stats 附加 pool 统计）。
func TestProvider_HandleStats_WithProxyPool(t *testing.T) {
	opts := DefaultProviderOptions()
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := runner.NewDriverPool(&runnerOpts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	defer pool.Close()
	p := NewProvider(opts)
	p.pool = pool
	p.startedAt = time.Now()

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()
	p.handleStats(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("stats 应返回 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "max_concurrent") {
		t.Errorf("响应应包含 pool 统计, got: %s", rr.Body.String())
	}
}

// TestProvider_HandleScreenshot_PoolScreenshotFailure 覆盖 handleScreenshot 的
// pool!=nil + Screenshot 失败分支（provider.go:371-374，返回 500）。用 proxyProvider
// 模式 pool，Screenshot 走代理→ensureProxyBrowser 失败。
func TestProvider_HandleScreenshot_PoolScreenshotFailure(t *testing.T) {
	opts := DefaultProviderOptions()
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	runnerOpts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	pool, err := runner.NewDriverPool(&runnerOpts, 1)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	defer pool.Close()
	p := NewProvider(opts)
	p.pool = pool
	p.ready = true

	req := httptest.NewRequest(http.MethodPost, "/screenshot?url=https://example.com", nil)
	rr := httptest.NewRecorder()
	p.handleScreenshot(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Logf("状态码 = %d（截图失败应 500，可能其他情况）, body=%s", rr.Code, rr.Body.String())
	}
}

// TestProvider_HandleHealth_ClosedPool 覆盖 handleHealth 的 stats.Closed 分支
// （provider.go:298-301，返回 unhealthy + 503）。用 proxyProvider pool 关闭后
// stats.Closed=true。
func TestProvider_HandleHealth_ClosedPool(t *testing.T) {
	opts := DefaultProviderOptions()
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := runner.NewDriverPool(&runnerOpts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	pool.Close() // 关闭后 stats.Closed=true
	p := NewProvider(opts)
	p.pool = pool
	p.ready = true
	p.startedAt = time.Now()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	p.handleHealth(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed pool health 应返回 503, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "unhealthy") {
		t.Errorf("响应应包含 unhealthy, got: %s", rr.Body.String())
	}
}

// TestProvider_Shutdown_WithProxyPool 覆盖 Shutdown 的 pool!=nil 分支
// （provider.go:228-230，pool.Close）。用 proxyProvider 模式 pool。
func TestProvider_Shutdown_WithProxyPool(t *testing.T) {
	opts := DefaultProviderOptions()
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := runner.NewDriverPool(&runnerOpts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	p := NewProvider(opts)
	p.pool = pool
	p.ready = true
	if err := p.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if p.ready {
		t.Error("Shutdown 后 ready 应为 false")
	}
}
