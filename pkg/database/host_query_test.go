package database

import (
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// newTestDBForHostQuery 创建内存数据库实例，专用于 host/URL 列表查询测试。
func newTestDBForHostQuery(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(Options{Path: ":memory:"})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// makeResult 构造仅含列表查询所需字段的扫描结果。
func makeResult(url, host string, probedAt time.Time) *models.Result {
	return &models.Result{
		URL:      url,
		Host:     host,
		Title:    "test",
		ProbedAt: probedAt,
	}
}

// TestGetScreenshotsByHost_HappyPath 验证按 host 前缀查询返回匹配记录且按 probed_at 倒序。
func TestGetScreenshotsByHost_HappyPath(t *testing.T) {
	db := newTestDBForHostQuery(t)

	// 时间锚点，确保前两条记录的 probed_at 严格小于第三条以验证倒序。
	base := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	earlier := base.Add(-2 * time.Hour)
	later := base.Add(-1 * time.Hour)

	// 三条记录：两条属于 example.com，一条属于 other.com。
	// 注意插入顺序故意与期望返回顺序相反，以验证 ORDER BY probed_at DESC 的排序逻辑。
	results := []*models.Result{
		makeResult("https://example.com/older", "example.com", earlier),
		makeResult("https://other.com/", "other.com", base),
		makeResult("https://example.com/newer", "example.com", later),
	}

	for _, r := range results {
		if err := db.SaveResult(r); err != nil {
			t.Fatalf("保存结果失败: %v", err)
		}
	}

	got, err := db.GetScreenshotsByHost("example.com")
	if err != nil {
		t.Fatalf("GetScreenshotsByHost 返回错误: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("期望返回 2 条记录，实际得到 %d 条", len(got))
	}

	// 验证倒序：later 应在前，earlier 应在后。
	if got[0].URL != "https://example.com/newer" {
		t.Errorf("首条记录 URL 不匹配，期望 https://example.com/newer，得到 %s", got[0].URL)
	}
	if got[1].URL != "https://example.com/older" {
		t.Errorf("次条记录 URL 不匹配，期望 https://example.com/older，得到 %s", got[1].URL)
	}
	if !got[0].ProbedAt.After(got[1].ProbedAt) {
		t.Errorf("期望按 probed_at 倒序排列，首条 probed_at=%v 应晚于次条 probed_at=%v", got[0].ProbedAt, got[1].ProbedAt)
	}

	// 确保返回结果中不含 other.com。
	for _, s := range got {
		if s.Host != "example.com" {
			t.Errorf("不应返回 host=%s 的记录，只期望 example.com", s.Host)
		}
	}
}

// TestGetScreenshotsByHost_NoMatch 验证无匹配 host 时返回空切片且无 error。
func TestGetScreenshotsByHost_NoMatch(t *testing.T) {
	db := newTestDBForHostQuery(t)

	if err := db.SaveResult(makeResult("https://example.com/", "example.com", time.Now())); err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	got, err := db.GetScreenshotsByHost("nonexistent.com")
	if err != nil {
		t.Fatalf("无匹配查询不应返回错误，得到: %v", err)
	}
	if got == nil {
		t.Fatal("无匹配查询应返回非 nil 空切片，得到 nil")
	}
	if len(got) != 0 {
		t.Errorf("无匹配查询应返回空切片，实际得到 %d 条", len(got))
	}
}

// TestGetScreenshotsByHost_EmptyInput 验证空字符串输入返回 error 且不查询数据库。
func TestGetScreenshotsByHost_EmptyInput(t *testing.T) {
	db := newTestDBForHostQuery(t)

	// 输入空格字符串也应被视为空输入。
	for _, in := range []string{"", "   "} {
		got, err := db.GetScreenshotsByHost(in)
		if err == nil {
			t.Errorf("空输入 host=%q 应返回 error，实际无 error", in)
		}
		if got != nil {
			t.Errorf("空输入 host=%q 不应返回记录，实际得到 %d 条", in, len(got))
		}
	}
}

// TestGetScreenshotsByURL_ReturnsAllHistory 验证同一 URL 多次扫描全部返回且按 probed_at 倒序。
func TestGetScreenshotsByURL_ReturnsAllHistory(t *testing.T) {
	db := newTestDBForHostQuery(t)

	target := "https://example.com/history"
	base := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)

	// 插入顺序故意与期望返回顺序相反。
	results := []*models.Result{
		makeResult(target, "example.com", base.Add(-2*time.Hour)),
		makeResult(target, "example.com", base.Add(-1*time.Hour)),
	}
	for _, r := range results {
		if err := db.SaveResult(r); err != nil {
			t.Fatalf("保存结果失败: %v", err)
		}
	}

	got, err := db.GetScreenshotsByURL(target)
	if err != nil {
		t.Fatalf("GetScreenshotsByURL 返回错误: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("期望返回 2 条历史记录，实际得到 %d 条", len(got))
	}

	// 验证倒序。
	if !got[0].ProbedAt.After(got[1].ProbedAt) {
		t.Errorf("期望按 probed_at 倒序排列，首条 probed_at=%v 应晚于次条 probed_at=%v", got[0].ProbedAt, got[1].ProbedAt)
	}
	// 全部记录应属于同一 URL。
	for _, s := range got {
		if s.URL != target {
			t.Errorf("URL 不匹配，期望 %s，得到 %s", target, s.URL)
		}
	}
}

// TestGetScreenshotsByURL_NoMatch 验证不存在的 URL 返回空切片无 error。
func TestGetScreenshotsByURL_NoMatch(t *testing.T) {
	db := newTestDBForHostQuery(t)

	if err := db.SaveResult(makeResult("https://example.com/", "example.com", time.Now())); err != nil {
		t.Fatalf("保存结果失败: %v", err)
	}

	got, err := db.GetScreenshotsByURL("https://nonexistent.com/")
	if err != nil {
		t.Fatalf("无匹配查询不应返回错误，得到: %v", err)
	}
	if got == nil {
		t.Fatal("无匹配查询应返回非 nil 空切片，得到 nil")
	}
	if len(got) != 0 {
		t.Errorf("无匹配查询应返回空切片，实际得到 %d 条", len(got))
	}
}
