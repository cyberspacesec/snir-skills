package database

import (
	"os"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// 创建测试用的数据库
func setupTestDB(t *testing.T) (*DB, func()) {
	// 使用内存数据库进行测试
	tempDBPath := ":memory:"

	// 创建数据库
	db, err := NewDB(Options{
		Path: tempDBPath,
	})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 返回清理函数
	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func createTestResult() *models.Result {
	return &models.Result{
		URL:            "https://example.com",
		Title:          "Example Domain",
		Filename:       "test_screenshot.png",
		FinalURL:       "https://example.com/",
		ResponseCode:   200,
		ResponseReason: "OK",
		Protocol:       "HTTP/2",
		ContentLength:  1256,
		HTML:           "<html><body>Example</body></html>",
		ProbedAt:       time.Now(),
		Failed:         false,
		FailedReason:   "",
	}
}

func TestNewDB(t *testing.T) {
	// 创建一个测试目录
	testDir := "test_db_dir"
	defer os.RemoveAll(testDir)

	// 测试创建数据库
	db, err := NewDB(Options{
		Path: testDir + "/test.db",
	})
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	defer db.Close()

	// 验证数据库已创建
	if db == nil {
		t.Error("数据库实例不应为 nil")
	}
}

func TestSaveAndGetResult(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建测试数据
	result := createTestResult()

	// 保存结果
	err := db.SaveResult(result)
	if err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	// 获取结果
	screenshots, err := db.GetAllScreenshots()
	if err != nil {
		t.Fatalf("获取所有截图失败: %v", err)
	}

	// 验证结果
	if len(screenshots) != 1 {
		t.Errorf("期望获得 1 个结果，但得到 %d 个", len(screenshots))
	}

	screenshot := screenshots[0]
	if screenshot.URL != result.URL {
		t.Errorf("URL 不匹配, 期望 %s, 得到 %s", result.URL, screenshot.URL)
	}

	if screenshot.Title != result.Title {
		t.Errorf("Title 不匹配, 期望 %s, 得到 %s", result.Title, screenshot.Title)
	}

	if screenshot.ResponseCode != result.ResponseCode {
		t.Errorf("ResponseCode 不匹配, 期望 %d, 得到 %d", result.ResponseCode, screenshot.ResponseCode)
	}
}

func TestSaveResults(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建测试数据
	results := []*models.Result{
		createTestResult(),
		{
			URL:            "https://example.org",
			Title:          "Another Example",
			Filename:       "test_screenshot2.png",
			ResponseCode:   200,
			ResponseReason: "OK",
			ProbedAt:       time.Now(),
		},
	}

	// 保存结果
	err := db.SaveResults(results)
	if err != nil {
		t.Fatalf("批量保存结果失败: %v", err)
	}

	// 获取结果
	screenshots, err := db.GetAllScreenshots()
	if err != nil {
		t.Fatalf("获取所有截图失败: %v", err)
	}

	// 验证结果
	if len(screenshots) != len(results) {
		t.Errorf("期望获得 %d 个结果，但得到 %d 个", len(results), len(screenshots))
	}
}

func TestGetScreenshotByURL(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建测试数据
	result := createTestResult()

	// 保存结果
	err := db.SaveResult(result)
	if err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	// 按URL获取结果
	screenshot, err := db.GetScreenshotByURL(result.URL)
	if err != nil {
		t.Fatalf("按URL获取截图失败: %v", err)
	}

	// 验证结果
	if screenshot.URL != result.URL {
		t.Errorf("URL 不匹配, 期望 %s, 得到 %s", result.URL, screenshot.URL)
	}
}

func TestScanSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建扫描会话
	sessionName := "Test Session"
	session, err := db.CreateScanSession(sessionName)
	if err != nil {
		t.Fatalf("创建扫描会话失败: %v", err)
	}

	// 验证会话
	if session.Name != sessionName {
		t.Errorf("会话名称不匹配, 期望 %s, 得到 %s", sessionName, session.Name)
	}

	// 结束会话
	err = db.EndScanSession(session.ID)
	if err != nil {
		t.Fatalf("结束扫描会话失败: %v", err)
	}
}

func TestTagOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 保存一个结果用于添加标签
	result := createTestResult()
	err := db.SaveResult(result)
	if err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	screenshots, err := db.GetAllScreenshots()
	if err != nil {
		t.Fatalf("获取所有截图失败: %v", err)
	}

	if len(screenshots) == 0 {
		t.Fatalf("期望至少有 1 个截图，但得到 0 个")
	}

	screenshotID := screenshots[0].ID

	// 创建标签
	tagName := "TestTag"
	tag, err := db.AddTag(tagName)
	if err != nil {
		t.Fatalf("添加标签失败: %v", err)
	}

	// 验证标签
	if tag.Name != tagName {
		t.Errorf("标签名称不匹配, 期望 %s, 得到 %s", tagName, tag.Name)
	}

	// 将标签添加到截图
	err = db.AddTagToScreenshot(screenshotID, tag.ID)
	if err != nil {
		t.Fatalf("给截图添加标签失败: %v", err)
	}

	// 获取所有标签
	tags, err := db.GetAllTags()
	if err != nil {
		t.Fatalf("获取所有标签失败: %v", err)
	}

	// 验证标签数量
	if len(tags) != 1 {
		t.Errorf("期望获得 1 个标签，但得到 %d 个", len(tags))
	}

	// 再次添加相同的标签，应该不会创建新标签
	duplicateTag, err := db.AddTag(tagName)
	if err != nil {
		t.Fatalf("添加重复标签失败: %v", err)
	}

	if duplicateTag.ID != tag.ID {
		t.Errorf("重复标签应该具有相同的 ID，期望 %d, 得到 %d", tag.ID, duplicateTag.ID)
	}

	// 获取所有标签，数量应该仍然是1
	tags, err = db.GetAllTags()
	if err != nil {
		t.Fatalf("获取所有标签失败: %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("添加重复标签后，期望只有 1 个标签，但得到 %d 个", len(tags))
	}
}

func TestExportResults(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 创建测试数据
	initialResult := createTestResult()

	// 保存结果
	err := db.SaveResult(initialResult)
	if err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	// 导出结果
	results, err := db.ExportResults()
	if err != nil {
		t.Fatalf("导出结果失败: %v", err)
	}

	// 验证结果
	if len(results) != 1 {
		t.Errorf("期望导出 1 个结果，但得到 %d 个", len(results))
	}

	exportedResult := results[0]
	if exportedResult.URL != initialResult.URL {
		t.Errorf("URL 不匹配, 期望 %s, 得到 %s", initialResult.URL, exportedResult.URL)
	}

	if exportedResult.Title != initialResult.Title {
		t.Errorf("Title 不匹配, 期望 %s, 得到 %s", initialResult.Title, exportedResult.Title)
	}
}
