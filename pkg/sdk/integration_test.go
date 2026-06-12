package sdk

import (
	"os"
	"testing"
	"time"
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
	opts.Timeout = 30 * time.Second

	// 步骤 1: 创建客户端（内部启动一个 Chrome 进程）
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("步骤1 - 创建客户端失败: %v", err)
	}
	defer client.Close()

	// 步骤 2: 第一次截图（百度首页较稳定）
	result1, err := client.Screenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("步骤2 - 第一次截图失败: %v", err)
	}
	if result1.Title == "" {
		t.Error("步骤2 - 截图结果缺少页面标题")
	}

	// 步骤 3: 第二次截图（复用同一浏览器进程）
	result2, err := client.Screenshot("https://www.baidu.com", nil)
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
