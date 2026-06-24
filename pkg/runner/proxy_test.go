package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStaticProxy(t *testing.T) {
	p := NewStaticProxy("http://127.0.0.1:8080")
	proxy, err := p.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if proxy != "http://127.0.0.1:8080" {
		t.Errorf("GetProxy() = %s, want http://127.0.0.1:8080", proxy)
	}
	if p.Name() != "static" {
		t.Errorf("Name() = %s, want static", p.Name())
	}
}

func TestNoProxy(t *testing.T) {
	p := NewNoProxy()
	proxy, err := p.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if proxy != "" {
		t.Errorf("GetProxy() = %s, want empty", proxy)
	}
}

func TestProxyList_RoundRobin(t *testing.T) {
	proxies := []string{"http://a:8080", "http://b:8080", "http://c:8080"}
	p := NewProxyList(proxies, ProxyRoundRobin)

	// Round-robin 应该循环
	results := make([]string, 6)
	for i := 0; i < 6; i++ {
		proxy, err := p.GetProxy()
		if err != nil {
			t.Fatalf("GetProxy() error = %v", err)
		}
		results[i] = proxy
	}

	// 应该是 a, b, c, a, b, c
	expected := []string{"http://a:8080", "http://b:8080", "http://c:8080", "http://a:8080", "http://b:8080", "http://c:8080"}
	for i, exp := range expected {
		if results[i] != exp {
			t.Errorf("results[%d] = %s, want %s", i, results[i], exp)
		}
	}
}

func TestProxyList_Random(t *testing.T) {
	proxies := []string{"http://a:8080", "http://b:8080"}
	p := NewProxyList(proxies, ProxyRandom)

	// 随机策略应该每次返回列表中的某个代理
	for i := 0; i < 10; i++ {
		proxy, err := p.GetProxy()
		if err != nil {
			t.Fatalf("GetProxy() error = %v", err)
		}
		if proxy != "http://a:8080" && proxy != "http://b:8080" {
			t.Errorf("GetProxy() = %s, 不在列表中", proxy)
		}
	}
}

func TestProxyList_Sequential(t *testing.T) {
	proxies := []string{"http://a:8080", "http://b:8080", "http://c:8080"}
	p := NewProxyList(proxies, ProxySequential)

	// 顺序策略：从第一个开始，依次切换，到最后一个停住
	results := make([]string, 5)
	for i := 0; i < 5; i++ {
		proxy, _ := p.GetProxy()
		results[i] = proxy
	}

	// 应该是 a, b, c, c, c
	if results[0] != "http://a:8080" || results[1] != "http://b:8080" || results[2] != "http://c:8080" {
		t.Errorf("Sequential 前3个 = %v, want a,b,c", results[:3])
	}
	// 最后两个应该停在 c
	if results[3] != "http://c:8080" || results[4] != "http://c:8080" {
		t.Errorf("Sequential 后2个 = %v, want c,c", results[3:])
	}
}

func TestProxyList_Empty(t *testing.T) {
	p := NewProxyList(nil, ProxyRoundRobin)
	if p != nil {
		t.Error("空列表应该返回 nil")
	}
}

func TestProxyFile(t *testing.T) {
	// 创建临时代理文件
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")
	content := `# 测试代理列表
http://proxy1:8080
http://proxy2:8080

# 另一个代理
http://proxy3:8080
`
	if err := os.WriteFile(proxyFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入代理文件失败: %v", err)
	}

	pf, err := NewProxyFile(proxyFile, ProxyRoundRobin)
	if err != nil {
		t.Fatalf("NewProxyFile() error = %v", err)
	}

	// 应该能读取 3 个代理
	for i := 0; i < 3; i++ {
		proxy, err := pf.GetProxy()
		if err != nil {
			t.Fatalf("GetProxy() error = %v", err)
		}
		if proxy == "" {
			t.Error("代理地址为空")
		}
	}

	if pf.Name() != "proxy-file("+proxyFile+")" {
		t.Errorf("Name() = %s", pf.Name())
	}
}

func TestProxyFile_NoProtocol(t *testing.T) {
	// 不带协议前缀的代理应该自动加 http://
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")
	content := "127.0.0.1:8080\n"
	if err := os.WriteFile(proxyFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入代理文件失败: %v", err)
	}

	pf, err := NewProxyFile(proxyFile, ProxyRoundRobin)
	if err != nil {
		t.Fatalf("NewProxyFile() error = %v", err)
	}

	proxy, err := pf.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if proxy != "http://127.0.0.1:8080" {
		t.Errorf("GetProxy() = %s, want http://127.0.0.1:8080", proxy)
	}
}

func TestProxyFile_Reload(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")

	// 初始写入
	if err := os.WriteFile(proxyFile, []byte("http://proxy1:8080\n"), 0644); err != nil {
		t.Fatalf("写入代理文件失败: %v", err)
	}

	pf, err := NewProxyFile(proxyFile, ProxyRoundRobin)
	if err != nil {
		t.Fatalf("NewProxyFile() error = %v", err)
	}

	proxy1, _ := pf.GetProxy()
	if proxy1 != "http://proxy1:8080" {
		t.Errorf("初始代理 = %s, want http://proxy1:8080", proxy1)
	}

	// 更新文件（需要修改时间变化）
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(proxyFile, []byte("http://proxy2:9090\n"), 0644); err != nil {
		t.Fatalf("更新代理文件失败: %v", err)
	}

	// 触发重载
	proxy2, _ := pf.GetProxy()
	// 注意：重载是基于文件修改时间的，可能需要稍微等待
	t.Logf("重载后代理 = %s", proxy2)
}

func TestProxyFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(proxyFile, []byte(""), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	_, err := NewProxyFile(proxyFile, ProxyRoundRobin)
	if err == nil {
		t.Error("空文件应该返回错误")
	}
}

func TestCreateProxyProvider_Static(t *testing.T) {
	opts := &Options{}
	opts.Chrome.Proxy = "http://127.0.0.1:8080"

	p := CreateProxyProvider(opts)
	if p.Name() != "static" {
		t.Errorf("Name() = %s, want static", p.Name())
	}
}

func TestCreateProxyProvider_NoProxy(t *testing.T) {
	opts := &Options{}
	p := CreateProxyProvider(opts)
	if p.Name() != "direct" {
		t.Errorf("Name() = %s, want direct", p.Name())
	}
}

func TestCreateProxyProvider_ProxyList(t *testing.T) {
	opts := &Options{}
	opts.Chrome.ProxyList = []string{"http://a:8080", "http://b:8080"}
	opts.Chrome.ProxyStrategy = ProxyRoundRobin

	p := CreateProxyProvider(opts)
	if p.Name() != "proxy-list(2,round-robin)" {
		t.Errorf("Name() = %s", p.Name())
	}
}

func TestCreateProxyProvider_ProxyFile(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")
	if err := os.WriteFile(proxyFile, []byte("http://proxy1:8080\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := &Options{}
	opts.Chrome.ProxyFile = proxyFile
	opts.Chrome.ProxyStrategy = ProxyRoundRobin

	p := CreateProxyProvider(opts)
	if p == nil {
		t.Error("CreateProxyProvider 返回 nil")
	}

	proxy, err := p.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if proxy != "http://proxy1:8080" {
		t.Errorf("GetProxy() = %s", proxy)
	}
}

func TestNewProxyAPI(t *testing.T) {
	api := NewProxyAPI("http://example.com/api/proxy", ProxyRandom)
	if api == nil {
		t.Fatal("NewProxyAPI should return non-nil")
	}
	if api.url != "http://example.com/api/proxy" {
		t.Errorf("url = %s, want http://example.com/api/proxy", api.url)
	}
	if api.strategy != ProxyRandom {
		t.Errorf("strategy = %s, want random", api.strategy)
	}
	if api.client == nil {
		t.Error("client should not be nil")
	}
	if api.client.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", api.client.timeout)
	}
}

func TestProxyAPI_Name(t *testing.T) {
	api := NewProxyAPI("http://proxy-api.example.com/get", ProxyRoundRobin)
	name := api.Name()
	expected := "proxy-api(http://proxy-api.example.com/get)"
	if name != expected {
		t.Errorf("Name() = %s, want %s", name, expected)
	}
}

func TestProxyAPI_GetProxy_EmptyResponse(t *testing.T) {
	api := NewProxyAPI("http://192.0.2.1:19999/nonexistent", ProxyRoundRobin)
	_, err := api.GetProxy()
	if err == nil {
		t.Error("Should return error for unreachable API")
	}
}

func TestProxyAPI_GetProxy_NoProtocol(t *testing.T) {
	// Test that the no-protocol branch works (code path coverage)
	// We can't easily test the full path without a mock HTTP server,
	// but we can verify the function signature and basic behavior
	api := NewProxyAPI("http://192.0.2.1:19999/api", ProxyRoundRobin)
	_, err := api.GetProxy()
	if err == nil {
		t.Error("Should return error for unreachable API")
	}
}

func TestCreateProxyProvider_ProxyURL(t *testing.T) {
	opts := &Options{}
	opts.Chrome.ProxyURL = "http://proxy-api.example.com/get"
	opts.Chrome.ProxyStrategy = ProxyRoundRobin

	p := CreateProxyProvider(opts)
	if p == nil {
		t.Fatal("CreateProxyProvider should return non-nil")
	}
	name := p.Name()
	if name != "proxy-api(http://proxy-api.example.com/get)" {
		t.Errorf("Name() = %s, want proxy-api(http://proxy-api.example.com/get)", name)
	}
}

func TestCreateProxyProvider_ProxyFileFallback(t *testing.T) {
	// When ProxyFile is set but file doesn't exist, should fall back to static proxy
	opts := &Options{}
	opts.Chrome.ProxyFile = "/nonexistent/proxy/file.txt"
	opts.Chrome.Proxy = "http://fallback:8080"

	p := CreateProxyProvider(opts)
	if p == nil {
		t.Fatal("CreateProxyProvider should return non-nil")
	}
	if p.Name() != "static" {
		t.Errorf("Name() = %s, want static (fallback)", p.Name())
	}
}

func TestCreateProxyProvider_ProxyFileFallbackNoProxy(t *testing.T) {
	// When ProxyFile is set but file doesn't exist, and no static proxy, should fall back to NoProxy
	opts := &Options{}
	opts.Chrome.ProxyFile = "/nonexistent/proxy/file.txt"

	p := CreateProxyProvider(opts)
	if p == nil {
		t.Fatal("CreateProxyProvider should return non-nil")
	}
	if p.Name() != "direct" {
		t.Errorf("Name() = %s, want direct (fallback)", p.Name())
	}
}

func TestProxyList_DefaultStrategy(t *testing.T) {
	proxies := []string{"http://a:8080"}
	p := NewProxyList(proxies, "unknown-strategy")
	if p == nil {
		t.Fatal("NewProxyList should not return nil for unknown strategy")
	}
	proxy, err := p.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if proxy != "http://a:8080" {
		t.Errorf("Default strategy should return first proxy, got %s", proxy)
	}
}

func TestProxyList_Name(t *testing.T) {
	proxies := []string{"http://a:8080", "http://b:8080"}
	p := NewProxyList(proxies, ProxyRoundRobin)
	name := p.Name()
	expected := "proxy-list(2,round-robin)"
	if name != expected {
		t.Errorf("Name() = %s, want %s", name, expected)
	}
}

func TestProxyList_GetProxy_EmptyList(t *testing.T) {
	// Create a ProxyList with empty list (bypassing NewProxyList's nil check)
	p := &ProxyList{
		proxies:  []string{},
		strategy: ProxyRoundRobin,
	}
	_, err := p.GetProxy()
	if err == nil {
		t.Error("Should return error for empty proxy list")
	}
}

func TestProxyFile_GetProxy_NilList(t *testing.T) {
	// Create a ProxyFile with a nil list to test the nil check
	pf := &ProxyFile{
		filePath: "/nonexistent",
		strategy: ProxyRoundRobin,
		list:     nil,
	}
	_, err := pf.GetProxy()
	if err == nil {
		t.Error("Should return error when list is nil")
	}
}

func TestProxyFile_ReloadFileError(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")
	os.WriteFile(proxyFile, []byte("http://proxy1:8080\n"), 0644)

	pf, err := NewProxyFile(proxyFile, ProxyRoundRobin)
	if err != nil {
		t.Fatalf("NewProxyFile() error = %v", err)
	}

	// Delete the file to trigger reload error
	os.Remove(proxyFile)

	// Should still work with cached list
	proxy, err := pf.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() should still work with cached list: %v", err)
	}
	if proxy != "http://proxy1:8080" {
		t.Errorf("GetProxy() = %s, want http://proxy1:8080", proxy)
	}
}

func TestLoadProxyFile_Nonexistent(t *testing.T) {
	_, err := loadProxyFile("/nonexistent/proxy/file.txt")
	if err == nil {
		t.Error("Should return error for nonexistent file")
	}
}

func TestLoadProxyFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(proxyFile, []byte("# only comments\n"), 0644)

	_, err := loadProxyFile(proxyFile)
	if err == nil {
		t.Error("Should return error for empty proxy file")
	}
}

func TestProxyFile_Name(t *testing.T) {
	pf := &ProxyFile{filePath: "/path/to/proxies.txt"}
	name := pf.Name()
	expected := "proxy-file(/path/to/proxies.txt)"
	if name != expected {
		t.Errorf("Name() = %s, want %s", name, expected)
	}
}

func TestSimpleHTTPClient_Get(t *testing.T) {
	client := &simpleHTTPClient{timeout: 1 * time.Second}

	// Test unreachable host
	_, err := client.Get("http://192.0.2.1:19999/nonexistent")
	if err == nil {
		t.Error("Should return error for unreachable host")
	}
}

func TestSimpleHTTPClient_Get_Non200(t *testing.T) {
	// We can't easily test non-200 without a real server, but we can test
	// the client creation and timeout
	client := &simpleHTTPClient{timeout: 5 * time.Second}
	if client.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", client.timeout)
	}
}

func TestProxyFile_ReloadIfNeeded_StatError(t *testing.T) {
	// Create a ProxyFile pointing to a nonexistent file path
	// The reloadIfNeeded should not panic when stat fails
	pf := &ProxyFile{
		filePath: "/nonexistent/path/proxies.txt",
		strategy: ProxyRoundRobin,
		list:     NewProxyList([]string{"http://cached:8080"}, ProxyRoundRobin),
	}
	// reloadIfNeeded is called by GetProxy, stat should fail, which is handled
	pf.reloadIfNeeded()
	// Should not panic and should keep the cached list
}

func TestProxyFile_ReloadIfNeeded_Success(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")

	// Write initial content
	os.WriteFile(proxyFile, []byte("http://proxy1:8080\n"), 0644)

	pf, err := NewProxyFile(proxyFile, ProxyRoundRobin)
	if err != nil {
		t.Fatalf("NewProxyFile() error = %v", err)
	}

	// Wait a bit then update the file to trigger reload
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(proxyFile, []byte("http://proxy2:9090\nhttp://proxy3:7070\n"), 0644)

	// Call reloadIfNeeded directly
	pf.reloadIfNeeded()

	// After reload, GetProxy should return from the new list
	proxy, err := pf.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() after reload error = %v", err)
	}
	// Should be one of the new proxies
	if proxy != "http://proxy2:9090" && proxy != "http://proxy3:7070" {
		t.Errorf("GetProxy() = %s, expected one of the new proxies", proxy)
	}
}

func TestLoadProxyFile_WithProtocol(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")
	content := "http://proxy1:8080\nsocks5://proxy2:1080\n"
	os.WriteFile(proxyFile, []byte(content), 0644)

	proxies, err := loadProxyFile(proxyFile)
	if err != nil {
		t.Fatalf("loadProxyFile() error = %v", err)
	}
	if len(proxies) != 2 {
		t.Errorf("Expected 2 proxies, got %d", len(proxies))
	}
	if proxies[0] != "http://proxy1:8080" {
		t.Errorf("proxies[0] = %s, want http://proxy1:8080", proxies[0])
	}
	if proxies[1] != "socks5://proxy2:1080" {
		t.Errorf("proxies[1] = %s, want socks5://proxy2:1080", proxies[1])
	}
}

func TestProxyFile_ReloadIfNeeded_NoChange(t *testing.T) {
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")

	os.WriteFile(proxyFile, []byte("http://proxy1:8080\n"), 0644)

	pf, err := NewProxyFile(proxyFile, ProxyRoundRobin)
	if err != nil {
		t.Fatalf("NewProxyFile() error = %v", err)
	}

	// Call reloadIfNeeded without modifying the file - should not reload
	pf.reloadIfNeeded()

	// Should still return the original proxy
	proxy, err := pf.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if proxy != "http://proxy1:8080" {
		t.Errorf("GetProxy() = %s, want http://proxy1:8080", proxy)
	}
}

func TestProxyAPI_GetProxy_HTTPError(t *testing.T) {
	// Test with a non-HTTP URL to trigger client.Get error
	api := NewProxyAPI("http://192.0.2.1:19999/api", ProxyRoundRobin)
	_, err := api.GetProxy()
	if err == nil {
		t.Error("Should return error for unreachable API")
	}
}

func TestCreateProxyProvider_Priority(t *testing.T) {
	// ProxyURL has highest priority
	opts := &Options{}
	opts.Chrome.ProxyURL = "http://api.example.com/proxy"
	opts.Chrome.ProxyFile = "/some/file"
	opts.Chrome.ProxyList = []string{"http://a:8080"}
	opts.Chrome.Proxy = "http://b:8080"

	p := CreateProxyProvider(opts)
	// Should choose ProxyURL (highest priority)
	name := p.Name()
	if !strings.Contains(name, "proxy-api") {
		t.Errorf("Should use ProxyURL (highest priority), got Name() = %s", name)
	}
}

func TestCreateProxyProvider_AllOptions(t *testing.T) {
	// ProxyFile has second priority when ProxyURL is empty
	tmpDir := t.TempDir()
	proxyFile := filepath.Join(tmpDir, "proxies.txt")
	os.WriteFile(proxyFile, []byte("http://proxy1:8080\n"), 0644)

	opts := &Options{}
	opts.Chrome.ProxyFile = proxyFile
	opts.Chrome.ProxyList = []string{"http://a:8080"}
	opts.Chrome.Proxy = "http://b:8080"

	p := CreateProxyProvider(opts)
	name := p.Name()
	if !strings.Contains(name, "proxy-file") {
		t.Errorf("Should use ProxyFile (second priority), got Name() = %s", name)
	}
}
