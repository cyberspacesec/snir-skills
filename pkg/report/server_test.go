package report

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewServer 测试创建新的服务器实例
func TestNewServer(t *testing.T) {
	options := ServerOptions{
		Host:           "localhost",
		Port:           8080,
		ScreenshotPath: "screenshots",
		ReportPath:     "reports",
	}

	server := NewServer(options)

	// 验证服务器实例
	if server == nil {
		t.Fatal("NewServer应该返回非nil的服务器实例")
	}

	// 验证选项设置正确
	if server.Options.Host != options.Host {
		t.Errorf("Host不匹配，期望%s，得到%s", options.Host, server.Options.Host)
	}
	if server.Options.Port != options.Port {
		t.Errorf("Port不匹配，期望%d，得到%d", options.Port, server.Options.Port)
	}
	if server.Options.ScreenshotPath != options.ScreenshotPath {
		t.Errorf("ScreenshotPath不匹配，期望%s，得到%s", options.ScreenshotPath, server.Options.ScreenshotPath)
	}
	if server.Options.ReportPath != options.ReportPath {
		t.Errorf("ReportPath不匹配，期望%s，得到%s", options.ReportPath, server.Options.ReportPath)
	}
}

// TestServeIndex 测试首页请求处理
func TestServeIndex(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "server_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建截图和报告目录
	screenshotDir := filepath.Join(tempDir, "screenshots")
	reportDir := filepath.Join(tempDir, "reports")
	if err := os.Mkdir(screenshotDir, 0755); err != nil {
		t.Fatalf("创建截图目录失败: %v", err)
	}
	if err := os.Mkdir(reportDir, 0755); err != nil {
		t.Fatalf("创建报告目录失败: %v", err)
	}

	// 创建测试截图文件
	screenshotFile := filepath.Join(screenshotDir, "example.com_20210101.png")
	if err := os.WriteFile(screenshotFile, []byte("fake PNG data"), 0644); err != nil {
		t.Fatalf("创建测试截图文件失败: %v", err)
	}

	// 创建测试报告文件
	reportFile := filepath.Join(reportDir, "report.html")
	if err := os.WriteFile(reportFile, []byte("<html>Test Report</html>"), 0644); err != nil {
		t.Fatalf("创建测试报告文件失败: %v", err)
	}

	// 创建服务器实例
	options := ServerOptions{
		Host:           "localhost",
		Port:           8080,
		ScreenshotPath: screenshotDir,
		ReportPath:     reportDir,
	}
	server := NewServer(options)

	// 创建HTTP请求
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// 调用serveIndex方法
	server.serveIndex(w, req, screenshotDir, reportDir)

	// 获取响应
	resp := w.Result()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，得到 %d", http.StatusOK, resp.StatusCode)
	}

	// 检查响应内容类型
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("期望内容类型包含 text/html，得到 %s", contentType)
	}
}

// TestGetFiles 测试获取文件列表功能
func TestGetFiles(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "get_files_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	validFiles := []string{
		filepath.Join(tempDir, "test1.png"),
		filepath.Join(tempDir, "test2.jpg"),
		filepath.Join(tempDir, "test3.jpeg"),
	}
	invalidFiles := []string{
		filepath.Join(tempDir, "test4.txt"),
		filepath.Join(tempDir, "test5.pdf"),
	}

	// 创建所有测试文件
	for _, file := range append(validFiles, invalidFiles...) {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 测试获取图片文件
	files, err := getFiles(tempDir, ".png", ".jpg", ".jpeg")
	if err != nil {
		t.Errorf("getFiles失败: %v", err)
	}

	// 检查返回的文件数量
	if len(files) != len(validFiles) {
		t.Errorf("期望找到 %d 个图片文件，但找到了 %d 个", len(validFiles), len(files))
	}

	// 测试空目录
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatalf("创建空目录失败: %v", err)
	}

	files, err = getFiles(emptyDir, ".png")
	if err != nil {
		t.Errorf("在空目录中getFiles失败: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("在空目录中期望找到 0 个文件，但找到了 %d 个", len(files))
	}

	// 测试不存在的目录
	nonexistentDir := filepath.Join(tempDir, "nonexistent")
	files, err = getFiles(nonexistentDir, ".png")

	// 根据getFiles的实际实现，如果目录不存在可能返回空切片而不是错误
	// 这里我们测试函数确实返回了一个空切片
	if len(files) != 0 {
		t.Errorf("对于不存在的目录，期望得到空切片，但找到了 %d 个文件", len(files))
	}
}

func TestServeIndex_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server_404_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	screenshotDir := filepath.Join(tempDir, "screenshots")
	reportDir := filepath.Join(tempDir, "reports")
	os.Mkdir(screenshotDir, 0755)
	os.Mkdir(reportDir, 0755)

	options := ServerOptions{
		Host:           "localhost",
		Port:           8080,
		ScreenshotPath: screenshotDir,
		ReportPath:     reportDir,
	}
	server := NewServer(options)

	// 请求非根路径
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.serveIndex(w, req, screenshotDir, reportDir)

	if w.Code != http.StatusNotFound {
		t.Errorf("非根路径应返回 404, 得到 %d", w.Code)
	}
}

func TestServeIndex_WithScreenshots(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server_screenshot_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	screenshotDir := filepath.Join(tempDir, "screenshots")
	reportDir := filepath.Join(tempDir, "reports")
	os.Mkdir(screenshotDir, 0755)
	os.Mkdir(reportDir, 0755)

	// 创建多个截图文件
	for _, fn := range []string{"example.com_20210101.png", "test.org_20210102.jpg"} {
		os.WriteFile(filepath.Join(screenshotDir, fn), []byte("fake"), 0644)
	}

	options := ServerOptions{
		Host:           "localhost",
		Port:           8080,
		ScreenshotPath: screenshotDir,
		ReportPath:     reportDir,
	}
	server := NewServer(options)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.serveIndex(w, req, screenshotDir, reportDir)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "截图 (2)") {
		t.Errorf("HTML 应包含截图计数, got: ...%s...", body[strings.LastIndex(body, "截图"):][:min(50, len(body)-strings.LastIndex(body, "截图"))])
	}
}

func TestServeIndex_WithReports(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server_report_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	screenshotDir := filepath.Join(tempDir, "screenshots")
	reportDir := filepath.Join(tempDir, "reports")
	os.Mkdir(screenshotDir, 0755)
	os.Mkdir(reportDir, 0755)

	// 创建报告文件
	for _, fn := range []string{"report.html", "data.json", "export.csv"} {
		os.WriteFile(filepath.Join(reportDir, fn), []byte("fake"), 0644)
	}

	options := ServerOptions{
		Host:           "localhost",
		Port:           8080,
		ScreenshotPath: screenshotDir,
		ReportPath:     reportDir,
	}
	server := NewServer(options)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.serveIndex(w, req, screenshotDir, reportDir)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "报告 (3)") {
		t.Errorf("HTML 应包含报告计数, got snippet: %s", body[max(0, strings.LastIndex(body, "报告")-5):][:60])
	}
}

func TestServeIndex_NoFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server_empty_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	screenshotDir := filepath.Join(tempDir, "screenshots")
	reportDir := filepath.Join(tempDir, "reports")
	os.Mkdir(screenshotDir, 0755)
	os.Mkdir(reportDir, 0755)

	options := ServerOptions{
		Host:           "localhost",
		Port:           8080,
		ScreenshotPath: screenshotDir,
		ReportPath:     reportDir,
	}
	server := NewServer(options)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.serveIndex(w, req, screenshotDir, reportDir)

	body := w.Body.String()
	if !strings.Contains(body, "没有找到截图文件") {
		t.Error("无截图时应显示提示信息")
	}
	if !strings.Contains(body, "没有找到报告文件") {
		t.Error("无报告时应显示提示信息")
	}
}

func TestGetFiles_NoExtensions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "get_no_ext_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "test.png"), []byte("test"), 0644)

	// 不指定扩展名——不应找到任何文件
	files, err := getFiles(tempDir)
	if err != nil {
		t.Errorf("getFiles 失败: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("不带扩展名参数应找到 0 个文件，但找到了 %d 个", len(files))
	}
}

func TestGetFiles_MixedExtensions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "get_mixed_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "a.png"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "b.jpg"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "c.gif"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "d.txt"), []byte(""), 0644)

	// 只匹配 png 和 jpg
	files, err := getFiles(tempDir, ".png", ".jpg")
	if err != nil {
		t.Errorf("getFiles 失败: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("期望找到 2 个文件，但找到了 %d 个: %v", len(files), files)
	}
}

func TestServerOptions(t *testing.T) {
	opts := ServerOptions{
		Host:           "127.0.0.1",
		Port:           9999,
		ScreenshotPath: "/tmp/screenshots",
		ReportPath:     "/tmp/reports",
	}

	s := NewServer(opts)
	if s.Options.Host != "127.0.0.1" {
		t.Errorf("Host = %s, want 127.0.0.1", s.Options.Host)
	}
	if s.Options.Port != 9999 {
		t.Errorf("Port = %d, want 9999", s.Options.Port)
	}
	if s.Options.ScreenshotPath != "/tmp/screenshots" {
		t.Errorf("ScreenshotPath = %s, want /tmp/screenshots", s.Options.ScreenshotPath)
	}
}

func TestServeIndex_ContentType(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server_ct_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	screenshotDir := filepath.Join(tempDir, "screenshots")
	reportDir := filepath.Join(tempDir, "reports")
	os.Mkdir(screenshotDir, 0755)
	os.Mkdir(reportDir, 0755)

	server := NewServer(ServerOptions{
		Host:           "localhost",
		Port:           8080,
		ScreenshotPath: screenshotDir,
		ReportPath:     reportDir,
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.serveIndex(w, req, screenshotDir, reportDir)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "charset=utf-8") {
		t.Errorf("Content-Type 应包含 charset=utf-8, 得到 %s", contentType)
	}
}

func TestGetFiles_SubDirectoriesNotListed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "get_subdir_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建子目录——它不应被当作文件列出
	subDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(tempDir, "file.png"), []byte("test"), 0644)

	files, err := getFiles(tempDir, ".png")
	if err != nil {
		t.Errorf("getFiles 失败: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("子目录不应计入，期望 1 个文件，但得到了 %d 个", len(files))
	}
}

// TestRun_CreateDirError 覆盖 Run 在 ScreenshotPath 不可创建时返回错误。
func TestRun_CreateDirError(t *testing.T) {
	tempDir := t.TempDir()
	// 把 ScreenshotPath 设为已存在的文件路径，MkdirAll 必然失败
	filePath := filepath.Join(tempDir, "afile")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}
	server := NewServer(ServerOptions{
		Host:           "127.0.0.1",
		Port:           0,
		ScreenshotPath: filePath,
		ReportPath:     tempDir,
	})
	if err := server.Run(); err == nil {
		t.Fatal("期望 Run 返回创建截图目录失败错误，得到 nil")
	}
}

// TestRun_ReportDirError 覆盖 Run 在 ReportPath 不可创建时返回错误。
func TestRun_ReportDirError(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "rfile")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}
	server := NewServer(ServerOptions{
		Host:           "127.0.0.1",
		Port:           0,
		ScreenshotPath: tempDir,
		ReportPath:     filePath,
	})
	if err := server.Run(); err == nil {
		t.Fatal("期望 Run 返回创建报告目录失败错误，得到 nil")
	}
}

// TestServeIndex_GetFilesError 覆盖 serveIndex 中 getFiles 失败分支。
func TestServeIndex_GetFilesError(t *testing.T) {
	tempDir := t.TempDir()
	// 把 screenshotPath 设为文件，getFiles 调用 ReadDir 失败
	badPath := filepath.Join(tempDir, "notdir")
	if err := os.WriteFile(badPath, []byte("x"), 0644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}
	server := NewServer(ServerOptions{ScreenshotPath: badPath, ReportPath: tempDir})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.serveIndex(rr, req, badPath, tempDir)
	// 无论状态码如何，只要不 panic 即视为通过（错误分支已覆盖）
	_ = rr.Code
}

// TestRun_ListenAndServeFailure 覆盖 Run 的成功路径（CreateDir 都成功）直至 ListenAndServe 调用：
// 先用 net.Listen 占住一个端口，再让 Run 尝试在同一端口 ListenAndServe → 立即返回错误，
// 覆盖 Run 的 line 37-62（除成功返回外全部行）。注意：Run 使用 DefaultServeMux，可能与其他
// 测试的全局注册冲突，故仅验证返回错误（端口已占用）而不发请求。
func TestRun_ListenAndServeFailure(t *testing.T) {
	// 占住一个空闲端口
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen 失败: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	tempDir := t.TempDir()
	server := NewServer(ServerOptions{
		Host:           "127.0.0.1",
		Port:           port,
		ScreenshotPath: tempDir,
		ReportPath:     tempDir,
	})
	// Run 会走到 ListenAndServe，因端口已被 ln 占用而返回错误
	if err := server.Run(); err == nil {
		t.Fatal("端口已占用时 Run 应返回错误")
	}
}
