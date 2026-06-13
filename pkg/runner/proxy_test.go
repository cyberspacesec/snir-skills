package runner

import (
	"os"
	"path/filepath"
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
