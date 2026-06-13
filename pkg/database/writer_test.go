package database

import (
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

func TestNewDBWriter(t *testing.T) {
	// 使用内存数据库进行测试
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	defer db.Close()

	writer := NewDBWriter(db)

	// 验证Writer是否成功创建
	if writer == nil {
		t.Fatal("NewDBWriter应返回非nil的Writer")
	}
}

func TestDBWriterWrite(t *testing.T) {
	// 使用内存数据库进行测试
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	defer db.Close()

	// 创建DBWriter
	writer := NewDBWriter(db)

	// 创建测试结果
	result := &models.Result{
		URL:            "https://example.com",
		Title:          "Example Domain",
		Filename:       "test_screenshot.png",
		ResponseCode:   200,
		ResponseReason: "OK",
		ProbedAt:       time.Now(),
	}

	// 调用Write方法
	err = writer.Write(result)
	if err != nil {
		t.Fatalf("DBWriter.Write返回错误: %v", err)
	}

	// 验证结果是否已保存到数据库
	screenshots, err := db.GetAllScreenshots()
	if err != nil {
		t.Fatalf("获取所有截图失败: %v", err)
	}

	if len(screenshots) != 1 {
		t.Fatalf("期望保存1个结果，但保存了%d个", len(screenshots))
	}

	savedScreenshot := screenshots[0]
	if savedScreenshot.URL != result.URL {
		t.Errorf("保存的URL不匹配, 期望: %s, 实际: %s", result.URL, savedScreenshot.URL)
	}
	if savedScreenshot.Title != result.Title {
		t.Errorf("保存的Title不匹配, 期望: %s, 实际: %s", result.Title, savedScreenshot.Title)
	}
}

func TestDBWriterClose(t *testing.T) {
	// 使用内存数据库进行测试
	db, err := NewDB(Options{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	defer db.Close()

	writer := NewDBWriter(db)

	// 调用Close方法
	err = writer.Close()
	if err != nil {
		t.Fatalf("DBWriter.Close返回错误: %v", err)
	}
}
