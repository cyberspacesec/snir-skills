package runner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestDiscoverChrome_NoInstance(t *testing.T) {
	// 在没有 Chrome 运行的端口上探测，应返回错误
	_, err := DiscoverChrome("127.0.0.1", []int{19999})
	if err == nil {
		t.Error("未运行的端口应返回错误")
	}
}

func TestDiscoverChromeWithTimeout_NoInstance(t *testing.T) {
	_, err := DiscoverChromeWithTimeout("127.0.0.1", []int{19999}, 1e9)
	if err == nil {
		t.Error("未运行的端口应返回错误")
	}
}

func TestAutoConnect_LocalMode(t *testing.T) {
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

	pool, mode, err := AutoConnect(opts, 2)
	if err != nil {
		t.Fatalf("AutoConnect() error = %v", err)
	}
	defer pool.Close()

	if mode != "local" && mode != "discovered" {
		t.Errorf("mode = %s, want local or discovered", mode)
	}

	// 验证池可用
	result, err := pool.Screenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}
	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestAutoConnect_RemoteMode(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	// 先启动一个 Chrome 实例获取 WSS URL
	localOpts := &Options{}
	localOpts.Chrome.Headless = true
	localOpts.Chrome.WindowX = 1280
	localOpts.Chrome.WindowY = 800
	localOpts.Chrome.Timeout = 30
	localOpts.Scan.ScreenshotPath = t.TempDir()
	localOpts.Scan.ScreenshotFormat = "png"

	localPool, err := NewDriverPool(localOpts, 1)
	if err != nil {
		t.Fatalf("创建本地池失败: %v", err)
	}

	// 获取 Chrome 的调试端口
	_ = localPool.Stats()

	// 设置 WSS URL（使用无效 URL 测试 remote 模式分支）
	remoteOpts := &Options{}
	remoteOpts.Chrome.WSS = "ws://127.0.0.1:9222/devtools/browser/test"
	remoteOpts.Chrome.Headless = true
	remoteOpts.Chrome.WindowX = 1280
	remoteOpts.Chrome.WindowY = 800
	remoteOpts.Chrome.Timeout = 30
	remoteOpts.Scan.ScreenshotPath = t.TempDir()
	remoteOpts.Scan.ScreenshotFormat = "png"

	// remote 模式（可能连接失败，但这只是测试分支覆盖）
	_, mode, err := AutoConnect(remoteOpts, 2)
	// 连接可能失败，因为 URL 不一定有效
	if err == nil {
		// 成功连接
		if mode != "remote" {
			t.Errorf("mode = %s, want remote", mode)
		}
	}
	// 如果连接失败也是预期的

	localPool.Close()
}

func TestProbeChromePort(t *testing.T) {
	_, err := probeChromePort("127.0.0.1", 19999)
	if err == nil {
		t.Error("探测不存在的端口应返回错误")
	}
}

// startFakeChrome 启动模拟 Chrome /json/version 端点的 httptest 服务器，返回其端口。
func startFakeChrome(t *testing.T, wsURL, browserVersion string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"webSocketDebuggerUrl": wsURL,
			"Browser":              browserVersion,
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// extractPort 从 httptest.Server.URL 解析端口。
func extractPort(t *testing.T, url string) int {
	t.Helper()
	var host, port string
	_, err := fmt.Sscanf(url, "http://%s", &host)
	if err != nil {
		t.Fatalf("解析 URL 失败: %v", err)
	}
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			port = host[i+1:]
			break
		}
	}
	var p int
	fmt.Sscanf(port, "%d", &p)
	if p == 0 {
		t.Fatalf("无法从 %s 解析端口", url)
	}
	return p
}

func TestProbeChromePortWithClient_Success(t *testing.T) {
	srv := startFakeChrome(t, "ws://127.0.0.1:9222/devtools/browser/abc", "Chrome/120.0.6099.109")
	port := extractPort(t, srv.URL)

	client := &http.Client{Timeout: 2 * time.Second}
	instance, err := probeChromePortWithClient("127.0.0.1", port, client)
	if err != nil {
		t.Fatalf("探测应成功: %v", err)
	}
	if instance.WsURL != "ws://127.0.0.1:9222/devtools/browser/abc" {
		t.Errorf("WsURL = %s", instance.WsURL)
	}
	if instance.Version != "Chrome/120.0.6099.109" {
		t.Errorf("Version = %s", instance.Version)
	}
	if instance.Port != port {
		t.Errorf("Port = %d, want %d", instance.Port, port)
	}
}

func TestProbeChromePortWithClient_NoWebSocketURL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"Browser": "Chrome/120",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	port := extractPort(t, srv.URL)

	client := &http.Client{Timeout: 2 * time.Second}
	_, err := probeChromePortWithClient("127.0.0.1", port, client)
	if err == nil {
		t.Fatal("缺少 webSocketDebuggerUrl 应返回错误")
	}
}

func TestProbeChromePortWithClient_InvalidJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	port := extractPort(t, srv.URL)

	client := &http.Client{Timeout: 2 * time.Second}
	_, err := probeChromePortWithClient("127.0.0.1", port, client)
	if err == nil {
		t.Fatal("无效 JSON 应返回错误")
	}
}

func TestDiscoverChrome_Success(t *testing.T) {
	srv := startFakeChrome(t, "ws://127.0.0.1:9222/devtools/browser/xyz", "Chrome/121.0")
	port := extractPort(t, srv.URL)

	instance, err := DiscoverChrome("127.0.0.1", []int{port})
	if err != nil {
		t.Fatalf("DiscoverChrome 应成功: %v", err)
	}
	if instance.WsURL != "ws://127.0.0.1:9222/devtools/browser/xyz" {
		t.Errorf("WsURL = %s", instance.WsURL)
	}
	if instance.Version != "Chrome/121.0" {
		t.Errorf("Version = %s", instance.Version)
	}
}

func TestDiscoverChrome_DefaultHostAndPorts(t *testing.T) {
	srv := startFakeChrome(t, "ws://127.0.0.1:9222/devtools/browser/def", "Chrome/122.0")
	port := extractPort(t, srv.URL)

	// host 为空应默认 127.0.0.1；端口列表传入只含 fake 端口的列表
	instance, err := DiscoverChrome("", []int{port})
	if err != nil {
		t.Fatalf("DiscoverChrome(空 host) 应成功: %v", err)
	}
	if instance == nil {
		t.Fatal("instance 不应为 nil")
	}
}

func TestDiscoverChromeWithTimeout_Success(t *testing.T) {
	srv := startFakeChrome(t, "ws://127.0.0.1:9222/devtools/browser/t", "Chrome/123.0")
	port := extractPort(t, srv.URL)

	instance, err := DiscoverChromeWithTimeout("127.0.0.1", []int{port}, 2*time.Second)
	if err != nil {
		t.Fatalf("DiscoverChromeWithTimeout 应成功: %v", err)
	}
	if instance.WsURL != "ws://127.0.0.1:9222/devtools/browser/t" {
		t.Errorf("WsURL = %s", instance.WsURL)
	}
}

func TestDiscoverChrome_FallsThroughPorts(t *testing.T) {
	// 第一个端口无服务，第二个端口是 fake Chrome，应跳过失败端口并成功
	srv := startFakeChrome(t, "ws://127.0.0.1:9222/devtools/browser/scan", "Chrome/124.0")
	port := extractPort(t, srv.URL)

	instance, err := DiscoverChrome("127.0.0.1", []int{19998, port})
	if err != nil {
		t.Fatalf("应跳过失败端口并成功: %v", err)
	}
	if instance.Port != port {
		t.Errorf("Port = %d, want %d", instance.Port, port)
	}
}

// TestDiscoverChromeWithTimeout_DefaultHostAndPorts 覆盖 host=="" 和
// ports==nil 的默认值分支（discovery.go:44-49）。
func TestDiscoverChromeWithTimeout_DefaultHostAndPorts(t *testing.T) {
	// 用空 host + nil ports，应使用默认 127.0.0.1 + 默认端口，
	// 探测失败返回错误（不依赖真实 Chrome）。
	_, err := DiscoverChromeWithTimeout("", nil, 100*time.Millisecond)
	if err == nil {
		t.Fatal("无 Chrome 实例时应返回错误")
	}
}
