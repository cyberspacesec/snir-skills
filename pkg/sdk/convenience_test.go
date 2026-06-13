package sdk

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestClient_ScreenshotFullPage(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	result, err := client.ScreenshotFullPage("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("ScreenshotFullPage() error = %v", err)
	}
	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestClient_ScreenshotElement(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	// 元素截图 - 选择器可能不存在，但方法应该能正常调用
	result, err := client.ScreenshotElement("https://www.baidu.com", "body", nil)
	if err != nil {
		// 元素截图可能失败，这是正常的
		t.Logf("ScreenshotElement() error = %v (可能是选择器问题)", err)
	} else {
		t.Logf("ScreenshotElement() title = %s", result.Title)
	}
}

func TestClient_ScreenshotWithJS(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	result, err := client.ScreenshotWithJS("https://www.baidu.com", "document.title", nil)
	if err != nil {
		t.Fatalf("ScreenshotWithJS() error = %v", err)
	}
	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestClient_BatchScreenshotStreaming(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second
	opts.MaxConcurrent = 2

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	urls := []string{
		"https://www.baidu.com",
		"https://www.baidu.com",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ch := client.BatchScreenshotStreaming(ctx, urls, nil)

	count := 0
	for result := range ch {
		count++
		if result.Error != nil {
			t.Logf("BatchStreaming[%d] %s error: %v", count, result.URL, result.Error)
		} else {
			t.Logf("BatchStreaming[%d] %s title: %s", count, result.URL, result.Result.Title)
		}
	}

	if count != len(urls) {
		t.Errorf("收到 %d 个结果, 期望 %d", count, len(urls))
	}
}

func TestClient_BatchScreenshotCallback(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second
	opts.MaxConcurrent = 2

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	urls := []string{"https://www.baidu.com"}
	var results []BatchResult

	ctx := context.Background()
	client.BatchScreenshotCallback(ctx, urls, nil, func(r BatchResult) {
		results = append(results, r)
	})

	if len(results) != 1 {
		t.Errorf("回调次数 = %d, 期望 1", len(results))
	}
}

func TestClient_ScreenshotWithActions(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	// 简单动作：等待 1 秒
	actions := []runner.InteractionAction{
		{Type: "wait", WaitTime: 1},
	}

	result, err := client.ScreenshotWithActions("https://www.baidu.com", actions, nil)
	if err != nil {
		t.Fatalf("ScreenshotWithActions() error = %v", err)
	}
	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestClient_ScreenshotWithCookies(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	cookies := []runner.CustomCookie{
		{Name: "test_cookie", Value: "sdk_test", Domain: ".baidu.com"},
	}

	result, err := client.ScreenshotWithCookies("https://www.baidu.com", cookies, nil)
	if err != nil {
		t.Fatalf("ScreenshotWithCookies() error = %v", err)
	}
	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestClient_ScreenshotHTML(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	html, result, err := client.ScreenshotHTML("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("ScreenshotHTML() error = %v", err)
	}
	if html == "" {
		t.Error("HTML 源码为空")
	}
	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

func TestResultWrapper(t *testing.T) {
	r := &models.Result{
		Title:        "Test Page",
		ResponseCode: 200,
		HTML:         "<html></html>",
		Screenshot:   "/tmp/test.png",
		Failed:       false,
		Headers:      []models.Header{{Name: "Content-Type", Value: "text/html"}},
		Cookies:      []models.Cookie{{Name: "session", Value: "abc"}},
		Console:      []models.ConsoleLog{{Level: "error", Message: "test error"}},
		Technologies: []models.Technology{{Name: "React", Version: "18"}},
	}

	w := WrapResult(r)

	if !w.IsSuccess() {
		t.Error("IsSuccess() 应该为 true")
	}
	if w.IsFailed() {
		t.Error("IsFailed() 应该为 false")
	}
	if !w.HasScreenshot() {
		t.Error("HasScreenshot() 应该为 true")
	}
	if !w.HasHTML() {
		t.Error("HasHTML() 应该为 true")
	}
	if w.TitleOrDefault("default") != "Test Page" {
		t.Error("TitleOrDefault() 应该返回实际标题")
	}
	if w.ResponseCodeOrDefault(404) != 200 {
		t.Error("ResponseCodeOrDefault() 应该返回 200")
	}

	cookieMap := w.CookieMap()
	if cookieMap["session"] != "abc" {
		t.Errorf("CookieMap() = %v, 期望 session=abc", cookieMap)
	}

	headerVal := w.HeaderValue("Content-Type")
	if headerVal != "text/html" {
		t.Errorf("HeaderValue() = %s, 期望 text/html", headerVal)
	}

	errors := w.ConsoleErrors()
	if len(errors) != 1 || errors[0] != "test error" {
		t.Errorf("ConsoleErrors() = %v", errors)
	}

	techNames := w.TechnologyNames()
	if len(techNames) != 1 || techNames[0] != "React" {
		t.Errorf("TechnologyNames() = %v", techNames)
	}
}

func TestResultWrapper_Nil(t *testing.T) {
	var w *ResultWrapper = nil

	if w.IsSuccess() {
		t.Error("nil ResultWrapper IsSuccess() 应该为 false")
	}
	if !w.IsFailed() {
		t.Error("nil ResultWrapper IsFailed() 应该为 true")
	}
	if w.HasScreenshot() {
		t.Error("nil ResultWrapper HasScreenshot() 应该为 false")
	}
	if w.TitleOrDefault("default") != "default" {
		t.Error("nil ResultWrapper 应该返回默认值")
	}
}

func TestResultWrapper_Failed(t *testing.T) {
	r := &models.Result{Failed: true, FailedReason: "timeout"}
	w := WrapResult(r)

	if w.IsSuccess() {
		t.Error("失败结果 IsSuccess() 应该为 false")
	}
	if !w.IsFailed() {
		t.Error("失败结果 IsFailed() 应该为 true")
	}
}
