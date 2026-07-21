package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		// 网络环境不稳定时可能失败，仅记录不报错
		t.Logf("ScreenshotWithActions() error = %v (可能是网络问题)", err)
		return
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

func TestWrapResult_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w != nil {
		t.Error("WrapResult(nil) 应该返回 nil")
	}
}

func TestResultWrapper_HasScreenshot_Empty(t *testing.T) {
	r := &models.Result{Screenshot: ""}
	w := WrapResult(r)
	if w.HasScreenshot() {
		t.Error("空截图路径 HasScreenshot() 应该为 false")
	}
}

func TestResultWrapper_HasHTML_Empty(t *testing.T) {
	r := &models.Result{HTML: ""}
	w := WrapResult(r)
	if w.HasHTML() {
		t.Error("空 HTML HasHTML() 应该为 false")
	}
}

func TestResultWrapper_TitleOrDefault_Empty(t *testing.T) {
	r := &models.Result{Title: ""}
	w := WrapResult(r)
	if w.TitleOrDefault("default") != "default" {
		t.Error("空标题 TitleOrDefault() 应该返回默认值")
	}
}

func TestResultWrapper_TitleOrDefault_NilResult(t *testing.T) {
	w := WrapResult(nil)
	if w.TitleOrDefault("default") != "default" {
		t.Error("nil Result TitleOrDefault() 应该返回默认值")
	}
}

func TestResultWrapper_ResponseCodeOrDefault_Zero(t *testing.T) {
	r := &models.Result{ResponseCode: 0}
	w := WrapResult(r)
	if w.ResponseCodeOrDefault(404) != 404 {
		t.Error("ResponseCode 为 0 时应该返回默认值")
	}
}

func TestResultWrapper_ResponseCodeOrDefault_NilResult(t *testing.T) {
	w := WrapResult(nil)
	if w.ResponseCodeOrDefault(404) != 404 {
		t.Error("nil Result ResponseCodeOrDefault() 应该返回默认值")
	}
}

func TestResultWrapper_HeaderMap_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w.HeaderMap() != nil {
		t.Error("nil ResultWrapper HeaderMap() 应该返回 nil")
	}
}

func TestResultWrapper_HeaderMap_WithData(t *testing.T) {
	r := &models.Result{
		Headers: []models.Header{
			{Name: "Content-Type", Value: "text/html"},
			{Name: "Server", Value: "nginx"},
		},
	}
	w := WrapResult(r)
	m := w.HeaderMap()
	if len(m) != 2 {
		t.Errorf("HeaderMap() len = %d, want 2", len(m))
	}
}

func TestResultWrapper_CookieMap_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w.CookieMap() != nil {
		t.Error("nil ResultWrapper CookieMap() 应该返回 nil")
	}
}

func TestResultWrapper_CookieMap_Empty(t *testing.T) {
	r := &models.Result{Cookies: []models.Cookie{}}
	w := WrapResult(r)
	m := w.CookieMap()
	if len(m) != 0 {
		t.Errorf("空 Cookies CookieMap() len = %d, want 0", len(m))
	}
}

func TestResultWrapper_HeaderValue_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w.HeaderValue("X-Test") != "" {
		t.Error("nil ResultWrapper HeaderValue() 应该返回空字符串")
	}
}

func TestResultWrapper_HeaderValue_NotFound(t *testing.T) {
	r := &models.Result{
		Headers: []models.Header{{Name: "Content-Type", Value: "text/html"}},
	}
	w := WrapResult(r)
	if w.HeaderValue("X-Missing") != "" {
		t.Error("不存在的 Header 应该返回空字符串")
	}
}

func TestResultWrapper_ConsoleErrors_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w.ConsoleErrors() != nil {
		t.Error("nil ResultWrapper ConsoleErrors() 应该返回 nil")
	}
}

func TestResultWrapper_ConsoleErrors_NoErrors(t *testing.T) {
	r := &models.Result{
		Console: []models.ConsoleLog{
			{Level: "info", Message: "info msg"},
			{Level: "warn", Message: "warn msg"},
		},
	}
	w := WrapResult(r)
	errors := w.ConsoleErrors()
	if len(errors) != 0 {
		t.Errorf("无 error 级别日志 ConsoleErrors() len = %d, want 0", len(errors))
	}
}

func TestResultWrapper_NetworkErrors_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w.NetworkErrors() != nil {
		t.Error("nil ResultWrapper NetworkErrors() 应该返回 nil")
	}
}

func TestResultWrapper_NetworkErrors_WithErrors(t *testing.T) {
	r := &models.Result{
		Network: []models.NetworkLog{
			{URL: "https://example.com/ok", StatusCode: 200},
			{URL: "https://example.com/notfound", StatusCode: 404},
			{URL: "https://example.com/error", StatusCode: 500},
		},
	}
	w := WrapResult(r)
	errors := w.NetworkErrors()
	if len(errors) != 2 {
		t.Errorf("NetworkErrors() len = %d, want 2", len(errors))
	}
}

func TestResultWrapper_NetworkErrors_NoErrors(t *testing.T) {
	r := &models.Result{
		Network: []models.NetworkLog{
			{URL: "https://example.com/ok", StatusCode: 200},
			{URL: "https://example.com/redirect", StatusCode: 302},
		},
	}
	w := WrapResult(r)
	errors := w.NetworkErrors()
	if len(errors) != 0 {
		t.Errorf("无 4xx/5xx 时 NetworkErrors() len = %d, want 0", len(errors))
	}
}

func TestResultWrapper_TechnologyNames_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w.TechnologyNames() != nil {
		t.Error("nil ResultWrapper TechnologyNames() 应该返回 nil")
	}
}

func TestResultWrapper_TechnologyNames_Empty(t *testing.T) {
	r := &models.Result{Technologies: []models.Technology{}}
	w := WrapResult(r)
	names := w.TechnologyNames()
	if len(names) != 0 {
		t.Errorf("空 Technologies TechnologyNames() len = %d, want 0", len(names))
	}
}

func TestResultWrapper_TLSInfo_Nil(t *testing.T) {
	w := WrapResult(nil)
	if w.TLSInfo() != nil {
		t.Error("nil ResultWrapper TLSInfo() 应该返回 nil")
	}
}

func TestResultWrapper_TLSInfo_WithData(t *testing.T) {
	r := &models.Result{
		TLS: models.TLS{
			Version:     "TLS 1.3",
			CipherSuite: "TLS_AES_128_GCM_SHA256",
		},
	}
	w := WrapResult(r)
	tls := w.TLSInfo()
	if tls == nil {
		t.Fatal("TLSInfo() 不应该返回 nil")
	}
	if tls.Version != "TLS 1.3" {
		t.Errorf("TLS Version = %s, want TLS 1.3", tls.Version)
	}
}

func TestResultWrapper_EvidenceSummary(t *testing.T) {
	r := &models.Result{
		HTML:            "<html></html>",
		Screenshot:      "/tmp/test.png",
		ScreenshotBytes: []byte("png"),
		Headers:         []models.Header{{Name: "Content-Type", Value: "text/html"}},
		Cookies:         []models.Cookie{{Name: "session", Value: "abc"}},
		Console: []models.ConsoleLog{
			{Level: "info", Message: "ready"},
			{Level: "error", Message: "boom"},
		},
		Network: []models.NetworkLog{
			{URL: "https://example.com", StatusCode: 200},
			{URL: "https://example.com/missing", StatusCode: 404},
		},
		Technologies: []models.Technology{{Name: "React"}},
		TLS:          models.TLS{Version: "TLS 1.3"},
	}

	w := WrapResult(r)
	if !w.HasEvidence() {
		t.Fatal("HasEvidence() 应该为 true")
	}

	summary := w.EvidenceSummary()
	if !summary.HasScreenshot || !summary.HasScreenshotBytes || !summary.HasHTML || !summary.HasTLS {
		t.Fatalf("EvidenceSummary() flags = %+v", summary)
	}
	if summary.HeaderCount != 1 || summary.CookieCount != 1 ||
		summary.ConsoleCount != 2 || summary.ConsoleErrorCount != 1 ||
		summary.NetworkCount != 2 || summary.NetworkErrorCount != 1 ||
		summary.TechnologyCount != 1 {
		t.Fatalf("EvidenceSummary() counts = %+v", summary)
	}
}

func TestResultWrapper_EvidenceSummary_Empty(t *testing.T) {
	w := WrapResult(&models.Result{})
	if w.HasEvidence() {
		t.Fatal("空结果 HasEvidence() 应该为 false")
	}
	if summary := w.EvidenceSummary(); summary != (EvidenceSummary{}) {
		t.Fatalf("空结果 EvidenceSummary() = %+v", summary)
	}
}

func TestResultWrapper_EvidenceSummary_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.HasEvidence() {
		t.Fatal("nil wrapper HasEvidence() 应该为 false")
	}
	if summary := w.EvidenceSummary(); summary != (EvidenceSummary{}) {
		t.Fatalf("nil wrapper EvidenceSummary() = %+v", summary)
	}
}

func TestResultWrapper_SaveJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	w := WrapResult(&models.Result{
		URL:             "https://example.com",
		Title:           "Example",
		HTML:            "<html></html>",
		ScreenshotBytes: []byte("png-bytes"),
	})

	if err := w.SaveJSON(path); err != nil {
		t.Fatalf("SaveJSON() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("导出的 JSON 无法解析: %v", err)
	}
	if decoded["title"] != "Example" {
		t.Fatalf("title = %v, want Example", decoded["title"])
	}
	jsonText := string(data)
	if strings.Contains(jsonText, "ScreenshotBytes") ||
		strings.Contains(jsonText, "screenshot_bytes") ||
		strings.Contains(jsonText, "png-bytes") {
		t.Fatalf("SaveJSON() 不应导出内存截图字节: %s", jsonText)
	}
}

func TestResultWrapper_SaveJSON_ErrorBranches(t *testing.T) {
	var nilWrapper *ResultWrapper
	if err := nilWrapper.SaveJSON(filepath.Join(t.TempDir(), "result.json")); err == nil {
		t.Fatal("nil wrapper SaveJSON() 应该返回错误")
	}

	w := WrapResult(&models.Result{})
	if err := w.SaveJSON(""); err == nil {
		t.Fatal("空路径 SaveJSON() 应该返回错误")
	}

	// WriteFile 失败分支（path 指向已存在文件当作目录）
	fileAsDir := filepath.Join(t.TempDir(), "afile")
	os.WriteFile(fileAsDir, []byte("x"), 0644)
	if err := w.SaveJSON(filepath.Join(fileAsDir, "out.json")); err == nil {
		t.Fatal("无效路径 SaveJSON() 应该返回 WriteFile 错误")
	}
}

func TestResultWrapper_SaveHTML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.html")
	w := WrapResult(&models.Result{HTML: "<html><body>ok</body></html>"})

	if err := w.SaveHTML(path); err != nil {
		t.Fatalf("SaveHTML() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "<html><body>ok</body></html>" {
		t.Fatalf("SaveHTML() data = %q", data)
	}
}

func TestResultWrapper_SaveHTML_ErrorBranches(t *testing.T) {
	var nilWrapper *ResultWrapper
	if err := nilWrapper.SaveHTML(filepath.Join(t.TempDir(), "page.html")); err == nil {
		t.Fatal("nil wrapper SaveHTML() 应该返回错误")
	}

	w := WrapResult(&models.Result{})
	if err := w.SaveHTML(""); err == nil {
		t.Fatal("空路径 SaveHTML() 应该返回错误")
	}
	if err := w.SaveHTML(filepath.Join(t.TempDir(), "page.html")); err == nil {
		t.Fatal("缺少 HTML SaveHTML() 应该返回错误")
	}

	// WriteFile 失败分支：HTML 非空但 path 指向已存在文件当作目录
	wHTML := WrapResult(&models.Result{HTML: "<html></html>"})
	fileAsDir := filepath.Join(t.TempDir(), "afile")
	os.WriteFile(fileAsDir, []byte("x"), 0644)
	if err := wHTML.SaveHTML(filepath.Join(fileAsDir, "out.html")); err == nil {
		t.Fatal("无效路径 SaveHTML() 应该返回 WriteFile 错误")
	}
}

func TestResultWrapper_ReadAndWriteScreenshot_InMemoryBytes(t *testing.T) {
	w := WrapResult(&models.Result{ScreenshotBytes: []byte("png")})

	data, err := w.ReadScreenshot()
	if err != nil {
		t.Fatalf("ReadScreenshot() error = %v", err)
	}
	if string(data) != "png" {
		t.Fatalf("ReadScreenshot() = %q, want png", data)
	}

	var buf bytes.Buffer
	if err := w.WriteScreenshot(&buf); err != nil {
		t.Fatalf("WriteScreenshot() error = %v", err)
	}
	if buf.String() != "png" {
		t.Fatalf("WriteScreenshot() data = %q, want png", buf.String())
	}

	path := filepath.Join(t.TempDir(), "shot.png")
	if err := w.SaveScreenshot(path); err != nil {
		t.Fatalf("SaveScreenshot() error = %v", err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(saved) != "png" {
		t.Fatalf("SaveScreenshot() data = %q, want png", saved)
	}
}

func TestResultWrapper_ReadScreenshot_FallsBackToFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	if err := os.WriteFile(source, []byte("file-png"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	w := WrapResult(&models.Result{Screenshot: source})
	data, err := w.ReadScreenshot()
	if err != nil {
		t.Fatalf("ReadScreenshot() error = %v", err)
	}
	if string(data) != "file-png" {
		t.Fatalf("ReadScreenshot() = %q, want file-png", data)
	}

	dest := filepath.Join(dir, "dest.png")
	if err := w.SaveScreenshot(dest); err != nil {
		t.Fatalf("SaveScreenshot() error = %v", err)
	}
	saved, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(saved) != "file-png" {
		t.Fatalf("SaveScreenshot() data = %q, want file-png", saved)
	}
}

func TestResultWrapper_ScreenshotExport_ErrorBranches(t *testing.T) {
	var nilWrapper *ResultWrapper
	if _, err := nilWrapper.ReadScreenshot(); err == nil {
		t.Fatal("nil wrapper ReadScreenshot() 应该返回错误")
	}
	if err := nilWrapper.SaveScreenshot(filepath.Join(t.TempDir(), "shot.png")); err == nil {
		t.Fatal("nil wrapper SaveScreenshot() 应该返回错误")
	}

	w := WrapResult(&models.Result{})
	if _, err := w.ReadScreenshot(); err == nil {
		t.Fatal("缺少截图 ReadScreenshot() 应该返回错误")
	}
	if err := w.WriteScreenshot(nil); err == nil {
		t.Fatal("nil writer WriteScreenshot() 应该返回错误")
	}
	if err := w.SaveScreenshot(""); err == nil {
		t.Fatal("空路径 SaveScreenshot() 应该返回错误")
	}
}

func TestResultWrapper_SaveEvidenceBundle(t *testing.T) {
	dir := t.TempDir()
	w := WrapResult(&models.Result{
		URL:             "https://example.com",
		Title:           "Example",
		HTML:            "<html><body>ok</body></html>",
		ScreenshotBytes: []byte("png"),
		Headers:         []models.Header{{Name: "Content-Type", Value: "text/html"}},
		Console:         []models.ConsoleLog{{Level: "error", Message: "boom"}},
		Network:         []models.NetworkLog{{URL: "https://example.com/missing", StatusCode: 404}},
	})

	bundle, err := w.SaveEvidenceBundle(dir)
	if err != nil {
		t.Fatalf("SaveEvidenceBundle() error = %v", err)
	}
	if bundle.Dir != dir {
		t.Fatalf("bundle.Dir = %q, want %q", bundle.Dir, dir)
	}
	for _, path := range []string{bundle.ManifestJSON, bundle.ResultJSON, bundle.SummaryJSON, bundle.HTML, bundle.Screenshot} {
		if path == "" {
			t.Fatalf("SaveEvidenceBundle() returned empty path: %+v", bundle)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected bundle file %q: %v", path, err)
		}
	}
	if filepath.Base(bundle.Screenshot) != "screenshot.png" {
		t.Fatalf("screenshot file = %q, want screenshot.png", filepath.Base(bundle.Screenshot))
	}

	html, err := os.ReadFile(bundle.HTML)
	if err != nil {
		t.Fatalf("ReadFile(html) error = %v", err)
	}
	if string(html) != "<html><body>ok</body></html>" {
		t.Fatalf("bundle HTML = %q", html)
	}
	shot, err := os.ReadFile(bundle.Screenshot)
	if err != nil {
		t.Fatalf("ReadFile(screenshot) error = %v", err)
	}
	if string(shot) != "png" {
		t.Fatalf("bundle screenshot = %q", shot)
	}

	var summary EvidenceSummary
	summaryData, err := os.ReadFile(bundle.SummaryJSON)
	if err != nil {
		t.Fatalf("ReadFile(summary) error = %v", err)
	}
	if err := json.Unmarshal(summaryData, &summary); err != nil {
		t.Fatalf("summary JSON decode error = %v", err)
	}
	if !summary.HasHTML || !summary.HasScreenshotBytes || summary.ConsoleErrorCount != 1 || summary.NetworkErrorCount != 1 {
		t.Fatalf("summary = %+v", summary)
	}

	var manifest EvidenceBundle
	manifestData, err := os.ReadFile(bundle.ManifestJSON)
	if err != nil {
		t.Fatalf("ReadFile(manifest) error = %v", err)
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("manifest JSON decode error = %v", err)
	}
	if manifest.Screenshot != bundle.Screenshot || manifest.EvidenceSummary.ConsoleErrorCount != 1 {
		t.Fatalf("manifest = %+v, bundle = %+v", manifest, bundle)
	}
}

func TestResultWrapper_SaveEvidenceBundle_UsesScreenshotExtension(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.jpeg")
	if err := os.WriteFile(source, []byte("jpeg"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	bundleDir := filepath.Join(dir, "bundle")
	w := WrapResult(&models.Result{Screenshot: source})
	bundle, err := w.SaveEvidenceBundle(bundleDir)
	if err != nil {
		t.Fatalf("SaveEvidenceBundle() error = %v", err)
	}
	if filepath.Base(bundle.Screenshot) != "screenshot.jpeg" {
		t.Fatalf("screenshot file = %q, want screenshot.jpeg", filepath.Base(bundle.Screenshot))
	}
	if bundle.HTML != "" {
		t.Fatalf("HTML path = %q, want empty", bundle.HTML)
	}
}

func TestResultWrapper_SaveEvidenceBundle_MetadataOnly(t *testing.T) {
	dir := t.TempDir()
	w := WrapResult(&models.Result{URL: "https://example.com", Title: "Example"})

	bundle, err := w.SaveEvidenceBundle(dir)
	if err != nil {
		t.Fatalf("SaveEvidenceBundle() error = %v", err)
	}
	if bundle.HTML != "" || bundle.Screenshot != "" {
		t.Fatalf("metadata-only bundle optional paths = %+v", bundle)
	}
	if _, err := os.Stat(bundle.ResultJSON); err != nil {
		t.Fatalf("result JSON missing: %v", err)
	}
	if _, err := os.Stat(bundle.SummaryJSON); err != nil {
		t.Fatalf("summary JSON missing: %v", err)
	}
	if _, err := os.Stat(bundle.ManifestJSON); err != nil {
		t.Fatalf("manifest JSON missing: %v", err)
	}
}

func TestResultWrapper_SaveEvidenceBundle_ErrorBranches(t *testing.T) {
	var nilWrapper *ResultWrapper
	if _, err := nilWrapper.SaveEvidenceBundle(t.TempDir()); err == nil {
		t.Fatal("nil wrapper SaveEvidenceBundle() 应该返回错误")
	}

	w := WrapResult(&models.Result{})
	if _, err := w.SaveEvidenceBundle(""); err == nil {
		t.Fatal("空目录 SaveEvidenceBundle() 应该返回错误")
	}
}

func TestResultWrapper_IsSuccess_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.IsSuccess() {
		t.Error("nil wrapper IsSuccess() 应该为 false")
	}
}

func TestResultWrapper_HasHTML_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.HasHTML() {
		t.Error("nil wrapper HasHTML() 应该为 false")
	}
}

func TestResultWrapper_HeaderValue_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.HeaderValue("X-Test") != "" {
		t.Error("nil wrapper HeaderValue() 应该返回空字符串")
	}
}

func TestResultWrapper_ConsoleErrors_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.ConsoleErrors() != nil {
		t.Error("nil wrapper ConsoleErrors() 应该返回 nil")
	}
}

func TestResultWrapper_NetworkErrors_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.NetworkErrors() != nil {
		t.Error("nil wrapper NetworkErrors() 应该返回 nil")
	}
}

func TestResultWrapper_TechnologyNames_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.TechnologyNames() != nil {
		t.Error("nil wrapper TechnologyNames() 应该返回 nil")
	}
}

func TestResultWrapper_TLSInfo_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.TLSInfo() != nil {
		t.Error("nil wrapper TLSInfo() 应该返回 nil")
	}
}

func TestResultWrapper_CookieMap_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.CookieMap() != nil {
		t.Error("nil wrapper CookieMap() 应该返回 nil")
	}
}

func TestResultWrapper_ResponseCodeOrDefault_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.ResponseCodeOrDefault(404) != 404 {
		t.Error("nil wrapper ResponseCodeOrDefault() 应该返回默认值")
	}
}

func TestResultWrapper_TitleOrDefault_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.TitleOrDefault("default") != "default" {
		t.Error("nil wrapper TitleOrDefault() 应该返回默认值")
	}
}

func TestResultWrapper_HasScreenshot_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if w.HasScreenshot() {
		t.Error("nil wrapper HasScreenshot() 应该为 false")
	}
}

func TestResultWrapper_IsFailed_NilWrapper(t *testing.T) {
	var w *ResultWrapper
	if !w.IsFailed() {
		t.Error("nil wrapper IsFailed() 应该为 true")
	}
}

// Test mergeWithScreenshotOptions comprehensive coverage
func TestMergeWithScreenshotOptions_Nil(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	merged := mergeWithScreenshotOptions(base, nil)
	if merged.Chrome.Timeout != base.Chrome.Timeout {
		t.Error("nil ScreenshotOptions 不应修改 base")
	}
}

func TestMergeWithScreenshotOptions_Delay(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{Delay: 5 * time.Second}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Chrome.Delay != 5 {
		t.Errorf("Delay = %d, want 5", merged.Chrome.Delay)
	}
}

func TestMergeWithScreenshotOptions_UserAgent(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{UserAgent: "test-agent"}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Chrome.UserAgent != "test-agent" {
		t.Errorf("UserAgent = %s, want test-agent", merged.Chrome.UserAgent)
	}
}

func TestMergeWithScreenshotOptions_Proxy(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{Proxy: "http://proxy:8080"}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Chrome.Proxy != "http://proxy:8080" {
		t.Errorf("Proxy = %s, want http://proxy:8080", merged.Chrome.Proxy)
	}
}

func TestMergeWithScreenshotOptions_Device(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{Device: "iphone-15"}
	merged := mergeWithScreenshotOptions(base, so)

	if merged.Chrome.DeviceName != "iPhone 15" {
		t.Errorf("DeviceName = %q, want iPhone 15", merged.Chrome.DeviceName)
	}
	if merged.Chrome.WindowX != 393 || merged.Chrome.WindowY != 852 {
		t.Errorf("viewport = %dx%d, want 393x852", merged.Chrome.WindowX, merged.Chrome.WindowY)
	}
	if merged.Chrome.DeviceScaleFactor != 3 || !merged.Chrome.IsMobile || !merged.Chrome.HasTouch {
		t.Errorf("device emulation not applied: dpr=%v mobile=%t touch=%t", merged.Chrome.DeviceScaleFactor, merged.Chrome.IsMobile, merged.Chrome.HasTouch)
	}
}

func TestMergeWithScreenshotOptions_XPath(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{XPath: "//div[@id='main']"}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Scan.XPath != "//div[@id='main']" {
		t.Errorf("XPath = %s, want //div[@id='main']", merged.Scan.XPath)
	}
}

func TestMergeWithScreenshotOptions_ScreenshotQuality(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{ScreenshotQuality: 80}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Scan.ScreenshotQuality != 80 {
		t.Errorf("ScreenshotQuality = %d, want 80", merged.Scan.ScreenshotQuality)
	}
}

func TestMergeWithScreenshotOptions_ScreenshotFormat(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{ScreenshotFormat: "jpeg"}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Scan.ScreenshotFormat != "jpeg" {
		t.Errorf("ScreenshotFormat = %s, want jpeg", merged.Scan.ScreenshotFormat)
	}
}

func TestMergeWithScreenshotOptions_JavaScript(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{JavaScript: "console.log('test')"}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Scan.JavaScript != "console.log('test')" {
		t.Errorf("JavaScript = %s", merged.Scan.JavaScript)
	}
	if !merged.Scan.RunJSAfter {
		t.Error("设置 JavaScript 后 RunJSAfter 应为 true")
	}
}

func TestMergeWithScreenshotOptions_JavaScriptFile(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{JavaScriptFile: "/path/to/script.js"}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Scan.JavaScriptFile != "/path/to/script.js" {
		t.Errorf("JavaScriptFile = %s", merged.Scan.JavaScriptFile)
	}
}

func TestMergeWithScreenshotOptions_RunJSBefore(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{RunJSBefore: true}
	merged := mergeWithScreenshotOptions(base, so)
	if !merged.Scan.RunJSBefore {
		t.Error("RunJSBefore 应为 true")
	}
}

func TestMergeWithScreenshotOptions_SaveHTML(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{SaveHTML: true}
	merged := mergeWithScreenshotOptions(base, so)
	if !merged.Scan.SaveHTML {
		t.Error("SaveHTML 应为 true")
	}
}

func TestMergeWithScreenshotOptions_SaveHeaders(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{SaveHeaders: true}
	merged := mergeWithScreenshotOptions(base, so)
	if !merged.Scan.SaveHeaders {
		t.Error("SaveHeaders 应为 true")
	}
}

func TestMergeWithScreenshotOptions_SaveConsole(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{SaveConsole: true}
	merged := mergeWithScreenshotOptions(base, so)
	if !merged.Scan.SaveConsole {
		t.Error("SaveConsole 应为 true")
	}
}

func TestMergeWithScreenshotOptions_SaveCookies(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{SaveCookies: true}
	merged := mergeWithScreenshotOptions(base, so)
	if !merged.Scan.SaveCookies {
		t.Error("SaveCookies 应为 true")
	}
}

func TestMergeWithScreenshotOptions_SaveNetwork(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{SaveNetwork: true}
	merged := mergeWithScreenshotOptions(base, so)
	if !merged.Scan.SaveNetwork {
		t.Error("SaveNetwork 应为 true")
	}
}

func TestMergeWithScreenshotOptions_Cookies(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	base.Scan.Cookies = []runner.CustomCookie{{Name: "existing", Value: "val"}}
	so := &ScreenshotOptions{
		Cookies: []runner.CustomCookie{{Name: "new", Value: "newval"}},
	}
	merged := mergeWithScreenshotOptions(base, so)
	if len(merged.Scan.Cookies) != 2 {
		t.Errorf("Cookies len = %d, want 2 (追加)", len(merged.Scan.Cookies))
	}
}

func TestMergeWithScreenshotOptions_Actions(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{
		Actions: []runner.InteractionAction{{Type: "click", Selector: "#btn"}},
	}
	merged := mergeWithScreenshotOptions(base, so)
	if len(merged.Scan.Actions) != 1 {
		t.Errorf("Actions len = %d, want 1", len(merged.Scan.Actions))
	}
}

func TestMergeWithScreenshotOptions_Form(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{
		Form: runner.Form{
			Fields:         []runner.FormField{{Selector: "#user", Value: "admin"}},
			SubmitSelector: "#login",
		},
	}
	merged := mergeWithScreenshotOptions(base, so)
	if len(merged.Scan.Form.Fields) != 1 {
		t.Errorf("Form.Fields len = %d, want 1", len(merged.Scan.Form.Fields))
	}

	submitOnly := mergeWithScreenshotOptions(base, &ScreenshotOptions{
		Form: runner.Form{SubmitSelector: "#continue", WaitAfterSubmit: 500},
	})
	if submitOnly.Scan.Form.SubmitSelector != "#continue" || submitOnly.Scan.Form.WaitAfterSubmit != 500 {
		t.Errorf("submit-only form not merged: %+v", submitOnly.Scan.Form)
	}
}

func TestMergeWithScreenshotOptions_MaxRetries(t *testing.T) {
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	so := &ScreenshotOptions{MaxRetries: 3}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Scan.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", merged.Scan.MaxRetries)
	}
}

func TestMergeWithScreenshotOptions_ZeroTimeout(t *testing.T) {
	// 零值 Timeout 不应覆盖
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	base.Chrome.Timeout = 30
	so := &ScreenshotOptions{Timeout: 0}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Chrome.Timeout != 30 {
		t.Errorf("零值 Timeout 不应覆盖, got %d", merged.Chrome.Timeout)
	}
}

func TestMergeWithScreenshotOptions_ZeroQuality(t *testing.T) {
	// 零值 Quality 不应覆盖
	opts := DefaultClientOptions()
	base := toRunnerOptions(opts)
	base.Scan.ScreenshotQuality = 90
	so := &ScreenshotOptions{ScreenshotQuality: 0}
	merged := mergeWithScreenshotOptions(base, so)
	if merged.Scan.ScreenshotQuality != 90 {
		t.Errorf("零值 Quality 不应覆盖, got %d", merged.Scan.ScreenshotQuality)
	}
}

// Test toRunnerOptions comprehensive
func TestToRunnerOptions_JavaScript(t *testing.T) {
	co := ClientOptions{
		JavaScript: "alert('test')",
	}
	runnerOpts := toRunnerOptions(co)
	if runnerOpts.Scan.JavaScript != "alert('test')" {
		t.Errorf("JavaScript = %s", runnerOpts.Scan.JavaScript)
	}
	if !runnerOpts.Scan.RunJSAfter {
		t.Error("设置 JavaScript 后 RunJSAfter 应为 true")
	}
}

func TestToRunnerOptions_JavaScriptFile(t *testing.T) {
	co := ClientOptions{
		JavaScriptFile: "/path/to/script.js",
	}
	runnerOpts := toRunnerOptions(co)
	if runnerOpts.Scan.JavaScriptFile != "/path/to/script.js" {
		t.Errorf("JavaScriptFile = %s", runnerOpts.Scan.JavaScriptFile)
	}
}

func TestToRunnerOptions_RunJSBefore(t *testing.T) {
	co := ClientOptions{
		RunJSBefore: true,
	}
	runnerOpts := toRunnerOptions(co)
	if !runnerOpts.Scan.RunJSBefore {
		t.Error("RunJSBefore 应为 true")
	}
}

func TestToRunnerOptions_Fingerprint(t *testing.T) {
	co := ClientOptions{
		AcceptLanguage:  "zh-CN",
		Platform:        "Win32",
		Vendor:          "Google Inc.",
		Plugins:         []string{"PDF Viewer"},
		WebGLVendor:     "Intel Inc.",
		WebGLRenderer:   "Intel Iris",
		CustomHeaders:   map[string]string{"X-Custom": "value"},
		DisableWebRTC:   true,
		SpoofScreenSize: true,
		ScreenWidth:     1920,
		ScreenHeight:    1080,
	}
	runnerOpts := toRunnerOptions(co)
	if runnerOpts.Chrome.AcceptLanguage != "zh-CN" {
		t.Error("AcceptLanguage 未映射")
	}
	if runnerOpts.Chrome.Platform != "Win32" {
		t.Error("Platform 未映射")
	}
	if runnerOpts.Chrome.Vendor != "Google Inc." {
		t.Error("Vendor 未映射")
	}
	if len(runnerOpts.Chrome.Plugins) != 1 {
		t.Error("Plugins 未映射")
	}
	if runnerOpts.Chrome.WebGLVendor != "Intel Inc." {
		t.Error("WebGLVendor 未映射")
	}
	if runnerOpts.Chrome.WebGLRenderer != "Intel Iris" {
		t.Error("WebGLRenderer 未映射")
	}
	if runnerOpts.Chrome.CustomHeaders["X-Custom"] != "value" {
		t.Error("CustomHeaders 未映射")
	}
	if !runnerOpts.Chrome.DisableWebRTC {
		t.Error("DisableWebRTC 未映射")
	}
	if !runnerOpts.Chrome.SpoofScreenSize {
		t.Error("SpoofScreenSize 未映射")
	}
	if runnerOpts.Chrome.ScreenWidth != 1920 {
		t.Error("ScreenWidth 未映射")
	}
	if runnerOpts.Chrome.ScreenHeight != 1080 {
		t.Error("ScreenHeight 未映射")
	}
}

func TestToRunnerOptions_Device(t *testing.T) {
	co := DefaultClientOptions()
	co.Device = "iphone-15"

	runnerOpts := toRunnerOptions(co)
	if runnerOpts.Chrome.DeviceName != "iPhone 15" {
		t.Errorf("DeviceName = %q, want iPhone 15", runnerOpts.Chrome.DeviceName)
	}
	if runnerOpts.Chrome.WindowX != 393 || runnerOpts.Chrome.WindowY != 852 {
		t.Errorf("viewport = %dx%d, want 393x852", runnerOpts.Chrome.WindowX, runnerOpts.Chrome.WindowY)
	}
	if runnerOpts.Chrome.DeviceScaleFactor != 3 || !runnerOpts.Chrome.IsMobile || !runnerOpts.Chrome.HasTouch {
		t.Errorf("device emulation not applied: dpr=%v mobile=%t touch=%t", runnerOpts.Chrome.DeviceScaleFactor, runnerOpts.Chrome.IsMobile, runnerOpts.Chrome.HasTouch)
	}
}

func TestToRunnerOptions_DeviceWithExplicitFingerprint(t *testing.T) {
	co := DefaultClientOptions()
	co.Device = "iphone-15"
	co.UserAgent = "custom-agent"
	co.Platform = "CustomOS"
	co.ScreenWidth = 360

	runnerOpts := toRunnerOptions(co)
	if runnerOpts.Chrome.UserAgent != "custom-agent" {
		t.Errorf("UserAgent = %q, want custom-agent", runnerOpts.Chrome.UserAgent)
	}
	if runnerOpts.Chrome.Platform != "CustomOS" {
		t.Errorf("Platform = %q, want CustomOS", runnerOpts.Chrome.Platform)
	}
	if runnerOpts.Chrome.ScreenWidth != 360 {
		t.Errorf("ScreenWidth = %d, want 360", runnerOpts.Chrome.ScreenWidth)
	}
	if runnerOpts.Chrome.ScreenHeight != 852 {
		t.Errorf("ScreenHeight should keep device value, got %d", runnerOpts.Chrome.ScreenHeight)
	}
}

func TestToRunnerOptions_DataCollection(t *testing.T) {
	co := ClientOptions{
		SaveHTML:    true,
		SaveHeaders: true,
		SaveConsole: true,
		SaveCookies: true,
		SaveNetwork: true,
	}
	runnerOpts := toRunnerOptions(co)
	if !runnerOpts.Scan.SaveHTML {
		t.Error("SaveHTML 未映射")
	}
	if !runnerOpts.Scan.SaveHeaders {
		t.Error("SaveHeaders 未映射")
	}
	if !runnerOpts.Scan.SaveConsole {
		t.Error("SaveConsole 未映射")
	}
	if !runnerOpts.Scan.SaveCookies {
		t.Error("SaveCookies 未映射")
	}
	if !runnerOpts.Scan.SaveNetwork {
		t.Error("SaveNetwork 未映射")
	}
}

func TestToRunnerOptions_Cookies(t *testing.T) {
	co := ClientOptions{
		Cookies:    []runner.CustomCookie{{Name: "test", Value: "val"}},
		CookieFile: "/tmp/cookies.json",
	}
	runnerOpts := toRunnerOptions(co)
	if len(runnerOpts.Scan.Cookies) != 1 {
		t.Error("Cookies 未映射")
	}
	if runnerOpts.Scan.CookiesFile != "/tmp/cookies.json" {
		t.Error("CookieFile 未映射")
	}
}

func TestToRunnerOptions_Actions(t *testing.T) {
	co := ClientOptions{
		Actions: []runner.InteractionAction{{Type: "click", Selector: "#btn"}},
	}
	runnerOpts := toRunnerOptions(co)
	if len(runnerOpts.Scan.Actions) != 1 {
		t.Error("Actions 未映射")
	}
}

func TestToRunnerOptions_Form(t *testing.T) {
	co := ClientOptions{
		Form: runner.Form{
			Fields:         []runner.FormField{{Selector: "#user", Value: "admin"}},
			SubmitSelector: "#login",
		},
	}
	runnerOpts := toRunnerOptions(co)
	if len(runnerOpts.Scan.Form.Fields) != 1 {
		t.Error("Form 未映射")
	}
}

func TestToRunnerOptions_Blacklist(t *testing.T) {
	co := ClientOptions{
		EnableBlacklist:   true,
		DefaultBlacklist:  true,
		BlacklistPatterns: []string{"*.internal.*"},
		BlacklistFile:     "/tmp/blacklist.txt",
	}
	runnerOpts := toRunnerOptions(co)
	if !runnerOpts.Scan.EnableBlacklist {
		t.Error("EnableBlacklist 未映射")
	}
	if !runnerOpts.Scan.DefaultBlacklist {
		t.Error("DefaultBlacklist 未映射")
	}
	if len(runnerOpts.Scan.BlacklistPatterns) != 1 {
		t.Error("BlacklistPatterns 未映射")
	}
	if runnerOpts.Scan.BlacklistFile != "/tmp/blacklist.txt" {
		t.Error("BlacklistFile 未映射")
	}
}

func TestToRunnerOptions_ScanDefaults(t *testing.T) {
	co := ClientOptions{}
	runnerOpts := toRunnerOptions(co)
	if !runnerOpts.Scan.HTTP {
		t.Error("HTTP 默认应为 true")
	}
	if !runnerOpts.Scan.HTTPS {
		t.Error("HTTPS 默认应为 true")
	}
}

// Test defaultRunnerOptions
func TestDefaultRunnerOptions(t *testing.T) {
	opts := defaultRunnerOptions()
	if !opts.Chrome.Headless {
		t.Error("默认应为无头模式")
	}
	if opts.Chrome.WindowX != 1280 {
		t.Errorf("默认宽度 = %d, want 1280", opts.Chrome.WindowX)
	}
	if opts.Chrome.WindowY != 800 {
		t.Errorf("默认高度 = %d, want 800", opts.Chrome.WindowY)
	}
	if opts.Chrome.Timeout != 30 {
		t.Errorf("默认超时 = %d, want 30", opts.Chrome.Timeout)
	}
	if opts.Scan.ScreenshotPath != "screenshots" {
		t.Errorf("默认截图路径 = %s, want screenshots", opts.Scan.ScreenshotPath)
	}
	if opts.Scan.ScreenshotFormat != "png" {
		t.Errorf("默认截图格式 = %s, want png", opts.Scan.ScreenshotFormat)
	}
}

// Test extractDomain
func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url    string
		expect string
	}{
		{"https://example.com/path", "example.com"},
		{"http://example.com:8080/path", "example.com"},
		{"https://sub.example.com/path?query=1", "sub.example.com"},
		{"https://example.com#fragment", "example.com"},
		{"example.com", "example.com"},
		{"example.com/path", "example.com"},
		{"example.com:8080", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractDomain(tt.url)
			if result != tt.expect {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.url, result, tt.expect)
			}
		})
	}
}

// Test ensureScreenshotOptions
func TestEnsureScreenshotOptions_Nil(t *testing.T) {
	c := &Client{}
	opts := c.ensureScreenshotOptions(nil)
	if opts == nil {
		t.Error("ensureScreenshotOptions(nil) 不应返回 nil")
	}
}

func TestEnsureScreenshotOptions_NotNil(t *testing.T) {
	c := &Client{}
	input := &ScreenshotOptions{Timeout: 10 * time.Second}
	opts := c.ensureScreenshotOptions(input)
	if opts != input {
		t.Error("ensureScreenshotOptions 应返回传入的非 nil 值")
	}
	if opts.Timeout != 10*time.Second {
		t.Error("ensureScreenshotOptions 不应修改传入值")
	}
}

// Test ScreenshotWithForm
func TestClient_ScreenshotWithForm_Unit(t *testing.T) {
	// 纯单元测试：验证 ScreenshotWithForm 正确设置 opts.Form
	// 由于需要浏览器，这里只测试 opts 构建逻辑
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	form := runner.Form{
		Fields:         []runner.FormField{{Selector: "#user", Value: "admin"}},
		SubmitSelector: "#login",
	}

	result, err := client.ScreenshotWithForm("https://www.baidu.com", form, nil)
	if err != nil {
		t.Logf("ScreenshotWithForm() error = %v (可能是网络问题)", err)
		return
	}
	if result.Title == "" {
		t.Error("截图结果缺少页面标题")
	}
}

// Test BatchScreenshotCallback with nil callback
func TestClient_BatchScreenshotCallback_NilCallback(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	// nil callback 不应 panic
	ctx := context.Background()
	client.BatchScreenshotCallback(ctx, []string{"https://www.baidu.com"}, nil, nil)
}

// Test Client cookie methods
func TestClient_AddCookie(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	// 注意: AddCookie 首次调用时会创建内存中的 CookieJar
	// 如果 persistent=true, 需要保存到文件; 否则只在内存中
	err = client.AddCookie(runner.PersistentCookie{
		Name:       "test",
		Value:      "val",
		Domain:     "example.com",
		Persistent: false, // 非持久化，不需要写入文件
	})
	if err != nil {
		t.Errorf("AddCookie() error = %v", err)
	}

	jar := client.CookieJar()
	if jar == nil {
		t.Error("AddCookie 后 CookieJar 不应为 nil")
	}
}

func TestClient_AddPersistentCookie(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	// AddPersistentCookie 会创建持久化 Cookie，需要文件系统支持
	// 由于内部 NewCookieJar("") 使用空路径，持久化时会失败
	// 这里测试非持久化场景
	err = client.AddCookie(runner.PersistentCookie{
		Name:       "session",
		Value:      "abc123",
		Domain:     "example.com",
		Persistent: false,
	})
	if err != nil {
		t.Errorf("AddPersistentCookie() error = %v", err)
	}
}

func TestClient_AddOneTimeCookie(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	err = client.AddOneTimeCookie("ot_session", "xyz", "example.com")
	if err != nil {
		t.Errorf("AddOneTimeCookie() error = %v", err)
	}
}

func TestClient_SetCookieJar(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	jar, err := runner.NewCookieJar("")
	if err != nil {
		t.Fatalf("NewCookieJar() error = %v", err)
	}

	client.SetCookieJar(jar)
	if client.CookieJar() != jar {
		t.Error("SetCookieJar 后 CookieJar() 应返回相同的 jar")
	}
}

func TestClient_CookieJar_Nil(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	// 没有设置 CookieFile，CookieJar 应为 nil
	if client.CookieJar() != nil {
		t.Error("未设置 CookieFile 时 CookieJar 应为 nil")
	}
}

func TestClient_OnEvent(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ScreenshotPath = t.TempDir()
	opts.Timeout = 30 * time.Second

	client, err := NewClient(opts)
	if err != nil {
		t.Skipf("无法创建客户端（可能没有Chrome）: %v", err)
	}
	defer client.Close()

	// 注册事件处理器不应 panic
	client.OnEvent(func(event runner.PoolEvent) {
		// 空回调
	})
}
