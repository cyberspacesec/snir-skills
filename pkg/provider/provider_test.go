package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/runner"
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
