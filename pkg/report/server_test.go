package report

import (
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
