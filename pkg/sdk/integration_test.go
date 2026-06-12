package sdk

import (
	"context"
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

// TestSDKScreenshotBytes 集成测试：验证截图字节数据获取
func TestSDKScreenshotBytes(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	imgBytes, result, err := client.ScreenshotBytes("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("ScreenshotBytes() 失败: %v", err)
	}

	if len(imgBytes) == 0 {
		t.Error("截图字节数据为空")
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}

	// PNG 文件头检查
	if len(imgBytes) >= 8 {
		if imgBytes[0] != 0x89 || imgBytes[1] != 'P' || imgBytes[2] != 'N' || imgBytes[3] != 'G' {
			t.Errorf("返回的数据不是 PNG 格式 (前4字节: %x)", imgBytes[:4])
		}
	}
}

// TestSDKBatchScreenshot 集成测试：批量截图
func TestSDKBatchScreenshot(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.MaxConcurrent = 2
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	urls := []string{
		"https://www.baidu.com",
		"https://www.bing.com",
	}

	results := client.BatchScreenshot(urls, nil)
	if len(results) != len(urls) {
		t.Fatalf("BatchScreenshot 返回 %d 个结果, 期望 %d", len(results), len(urls))
	}

	successCount := 0
	for _, r := range results {
		if r.Error == nil && r.Result != nil && !r.Result.Failed {
			successCount++
		}
	}

	if successCount == 0 {
		t.Error("批量截图全部失败")
	}
}

// TestSDKWithContext 集成测试：带 context 的截图
func TestSDKWithContext(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.ScreenshotWithContext(ctx, "https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("ScreenshotWithContext() 失败: %v", err)
	}

	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

// TestSDKStats 集成测试：统计信息
func TestSDKStats(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 执行一次截图
	_, _ = client.Screenshot("https://www.baidu.com", nil)

	stats := client.Stats()
	if stats.TotalScreenshots < 1 {
		t.Errorf("TotalScreenshots = %d, want >= 1", stats.TotalScreenshots)
	}
	if stats.Closed {
		t.Error("客户端不应标记为关闭")
	}
}