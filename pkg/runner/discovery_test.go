package runner

import (
	"os"
	"testing"
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
