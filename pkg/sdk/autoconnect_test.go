package sdk

import (
	"os"
	"testing"
	"time"
)

func TestAutoConnectClient(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, mode, err := AutoConnectClient(opts)
	if err != nil {
		t.Fatalf("AutoConnectClient() error = %v", err)
	}
	defer client.Close()

	if mode != AutoConnectLocal && mode != AutoConnectDiscovered {
		t.Errorf("mode = %s, want local or discovered", mode)
	}

	// 验证客户端可用
	result, err := client.Screenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}
