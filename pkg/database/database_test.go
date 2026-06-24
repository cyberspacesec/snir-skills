package database

import (
	"os"
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/cyberspacesec/snir-skills/pkg/log"
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
		PerceptionHash: "cafe1234",
		TLS: models.TLS{
			Version:         "TLS 1.3",
			CipherSuite:     "TLS_AES_128_GCM_SHA256",
			Issuer:          "Example CA",
			Subject:         "example.com",
			SANs:            "example.com,www.example.com",
			FingerprintSHA1: "001122",
		},
		Technologies: []models.Technology{
			{Name: "nginx", Version: "1.24"},
		},
		Headers: []models.Header{
			{Name: "Server", Value: "nginx"},
			{Name: "Content-Type", Value: "text/html"},
		},
		Network: []models.NetworkLog{
			{Type: models.HTTP, URL: "https://example.com/app.js", Method: "GET", StatusCode: 200, ContentType: "application/javascript"},
		},
		Console: []models.ConsoleLog{
			{Level: "info", Message: "ready"},
		},
		Cookies: []models.Cookie{
			{Name: "sid", Value: "abc", Domain: "example.com", Path: "/"},
		},
		ProbedAt:     time.Now(),
		Failed:       false,
		FailedReason: "",
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

	if screenshot.Endpoint != "https://example.com:443" {
		t.Errorf("Endpoint 不匹配, 期望 https://example.com:443, 得到 %s", screenshot.Endpoint)
	}

	if screenshot.HeadersJSON == "" || screenshot.TechnologiesJSON == "" || screenshot.TLSJSON == "" {
		t.Errorf("复杂证据 JSON 不应为空: headers=%q technologies=%q tls=%q", screenshot.HeadersJSON, screenshot.TechnologiesJSON, screenshot.TLSJSON)
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

func TestGetScreenshot(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 保存一个结果
	result := createTestResult()
	err := db.SaveResult(result)
	if err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	// 获取所有截图以获得 ID
	screenshots, err := db.GetAllScreenshots()
	if err != nil {
		t.Fatalf("获取所有截图失败: %v", err)
	}

	if len(screenshots) == 0 {
		t.Fatalf("期望至少有1个截图")
	}

	// 按ID获取截图
	screenshot, err := db.GetScreenshot(screenshots[0].ID)
	if err != nil {
		t.Fatalf("按ID获取截图失败: %v", err)
	}

	if screenshot.URL != result.URL {
		t.Errorf("URL不匹配, 期望 %s, 得到 %s", result.URL, screenshot.URL)
	}

	// 测试不存在ID的情况
	_, err = db.GetScreenshot(99999)
	if err == nil {
		t.Error("查询不存在ID应该返回错误")
	}
}

func TestGetScreenshotByURL_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 测试不存在URL的情况
	_, err := db.GetScreenshotByURL("https://not-exist.com")
	if err == nil {
		t.Error("查询不存在URL应该返回错误")
	}
}

func TestNewDB_InvalidPath(t *testing.T) {
	// 测试无法创建目录的情况（例如在只读文件系统下）
	_, err := NewDB(Options{
		Path: "/proc/test_impossible.db",
	})
	if err == nil {
		t.Error("在无法创建目录的位置应该返回错误")
	}
}

func TestDBWriterWrite_Error(t *testing.T) {
	// 测试 DBWriter.Write 错误路径
	// 创建一个已关闭的数据库
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	writer := NewDBWriter(db)

	// 关闭数据库后再写入
	db.Close()

	result := &models.Result{
		URL:      "https://example.com",
		Title:    "Test",
		ProbedAt: time.Now(),
	}

	err = writer.Write(result)
	if err == nil {
		t.Error("在关闭的数据库上写入应该返回错误")
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

	if exportedResult.SchemaVersion != models.ResultSchemaVersion {
		t.Errorf("SchemaVersion 不匹配, 期望 %s, 得到 %s", models.ResultSchemaVersion, exportedResult.SchemaVersion)
	}

	if exportedResult.Endpoint != "https://example.com:443" {
		t.Errorf("Endpoint 不匹配, 期望 https://example.com:443, 得到 %s", exportedResult.Endpoint)
	}

	if exportedResult.PerceptionHash != initialResult.PerceptionHash {
		t.Errorf("PerceptionHash 不匹配, 期望 %s, 得到 %s", initialResult.PerceptionHash, exportedResult.PerceptionHash)
	}

	if !reflect.DeepEqual(exportedResult.TLS, initialResult.TLS) {
		t.Errorf("TLS 不匹配, 期望 %#v, 得到 %#v", initialResult.TLS, exportedResult.TLS)
	}

	if !reflect.DeepEqual(exportedResult.Technologies, initialResult.Technologies) {
		t.Errorf("Technologies 不匹配, 期望 %#v, 得到 %#v", initialResult.Technologies, exportedResult.Technologies)
	}

	if !reflect.DeepEqual(exportedResult.Headers, initialResult.Headers) {
		t.Errorf("Headers 不匹配, 期望 %#v, 得到 %#v", initialResult.Headers, exportedResult.Headers)
	}

	if !reflect.DeepEqual(exportedResult.Network, initialResult.Network) {
		t.Errorf("Network 不匹配, 期望 %#v, 得到 %#v", initialResult.Network, exportedResult.Network)
	}

	if !reflect.DeepEqual(exportedResult.Console, initialResult.Console) {
		t.Errorf("Console 不匹配, 期望 %#v, 得到 %#v", initialResult.Console, exportedResult.Console)
	}

	if !reflect.DeepEqual(exportedResult.Cookies, initialResult.Cookies) {
		t.Errorf("Cookies 不匹配, 期望 %#v, 得到 %#v", initialResult.Cookies, exportedResult.Cookies)
	}
}

// TestNewDB_CreateDirError 测试创建目录失败的错误路径
func TestNewDB_CreateDirError(t *testing.T) {
	// /dev/null 是一个文件，不是目录，所以 filepath.Dir("/dev/null/test.db") = "/dev/null"
	// islazy.CreateDir("/dev/null") 会因为 /dev/null 已作为文件存在而失败
	_, err := NewDB(Options{
		Path: "/dev/null/test.db",
	})
	if err == nil {
		t.Error("在无法创建目录的位置应该返回错误")
	}
}

// TestNewDB_InitDBError 测试 initDB 失败的路径
// 使用只读内存数据库：gorm.Open 成功但 AutoMigrate 失败
func TestNewDB_InitDBError(t *testing.T) {
	_, err := NewDB(Options{
		Path: "file::memory:?mode=ro",
	})
	if err == nil {
		t.Error("在只读数据库中创建表应该返回错误")
	}
}

// TestNewDB_DebugLogging 测试启用调试日志时的 NewDB 路径
func TestNewDB_DebugLogging(t *testing.T) {
	// 启用调试日志以覆盖 log.IsDebugEnabled() == true 分支
	log.EnableDebug()
	defer log.EnableSilence() // 恢复静默模式以免影响其他测试

	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("在调试模式下创建内存数据库失败: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Error("数据库实例不应为 nil")
	}
}

// TestClose_ErrorPath 测试关闭无效数据库连接的错误路径
func TestClose_ErrorPath(t *testing.T) {
	// 创建一个没有有效连接池的 gorm.DB 来触发 d.db.DB() 的错误路径
	invalidDB := &DB{
		db: &gorm.DB{
			Config: &gorm.Config{},
		},
	}

	err := invalidDB.Close()
	if err == nil {
		t.Error("关闭无效数据库应该返回错误")
	}
}

// TestGetAllScreenshots_ClosedDB 测试在已关闭的数据库上获取所有截图
func TestGetAllScreenshots_ClosedDB(t *testing.T) {
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建内存数据库失败: %v", err)
	}

	// 先保存一些结果
	result := createTestResult()
	err = db.SaveResult(result)
	if err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	// 关闭数据库
	err = db.Close()
	if err != nil {
		t.Fatalf("关闭数据库失败: %v", err)
	}

	// 在已关闭的数据库上获取所有截图
	_, err = db.GetAllScreenshots()
	if err == nil {
		t.Error("在已关闭的数据库上获取所有截图应该返回错误")
	}
}

// TestGetAllScreenshots_EmptyTable 测试从空表获取所有截图
func TestGetAllScreenshots_EmptyTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 从空数据库获取所有截图
	screenshots, err := db.GetAllScreenshots()
	if err != nil {
		t.Fatalf("从空表获取截图失败: %v", err)
	}

	if len(screenshots) != 0 {
		t.Errorf("空表应该返回 0 个截图，但得到 %d 个", len(screenshots))
	}
}

// TestCreateScanSession_ClosedDB 测试在已关闭的数据库上创建扫描会话
func TestCreateScanSession_ClosedDB(t *testing.T) {
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建内存数据库失败: %v", err)
	}

	// 关闭数据库
	err = db.Close()
	if err != nil {
		t.Fatalf("关闭数据库失败: %v", err)
	}

	// 在已关闭的数据库上创建会话
	_, err = db.CreateScanSession("test_session")
	if err == nil {
		t.Error("在已关闭的数据库上创建扫描会话应该返回错误")
	}
}

// TestAddTag_ErrorPath 测试 AddTag 的错误路径 - 在已关闭的数据库上创建新标签
func TestAddTag_ErrorPath(t *testing.T) {
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建内存数据库失败: %v", err)
	}

	// 关闭数据库
	err = db.Close()
	if err != nil {
		t.Fatalf("关闭数据库失败: %v", err)
	}

	// 在已关闭的数据库上添加标签
	_, err = db.AddTag("test_tag")
	if err == nil {
		t.Error("在已关闭的数据库上添加标签应该返回错误")
	}
}

// TestGetAllTags_ClosedDB 测试在已关闭的数据库上获取所有标签
func TestGetAllTags_ClosedDB(t *testing.T) {
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建内存数据库失败: %v", err)
	}

	// 关闭数据库
	err = db.Close()
	if err != nil {
		t.Fatalf("关闭数据库失败: %v", err)
	}

	// 在已关闭的数据库上获取所有标签
	_, err = db.GetAllTags()
	if err == nil {
		t.Error("在已关闭的数据库上获取所有标签应该返回错误")
	}
}

// TestGetAllTags_EmptyTable 测试从空表获取所有标签
func TestGetAllTags_EmptyTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 从空数据库获取所有标签
	tags, err := db.GetAllTags()
	if err != nil {
		t.Fatalf("从空表获取标签失败: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("空表应该返回 0 个标签，但得到 %d 个", len(tags))
	}
}

// TestExportResults_ClosedDB 测试在已关闭的数据库上导出结果
func TestExportResults_ClosedDB(t *testing.T) {
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建内存数据库失败: %v", err)
	}

	// 关闭数据库
	err = db.Close()
	if err != nil {
		t.Fatalf("关闭数据库失败: %v", err)
	}

	// 在已关闭的数据库上导出结果
	_, err = db.ExportResults()
	if err == nil {
		t.Error("在已关闭的数据库上导出结果应该返回错误")
	}
}

// TestExportResults_EmptyTable 测试从空表导出结果
func TestExportResults_EmptyTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 从空数据库导出结果
	results, err := db.ExportResults()
	if err != nil {
		t.Fatalf("从空表导出结果失败: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("空表应该返回 0 个结果，但得到 %d 个", len(results))
	}
}
