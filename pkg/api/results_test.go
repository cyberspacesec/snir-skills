package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/cyberspacesec/snir-skills/pkg/database"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// newServerWithTestDB 构造带内存 DB 的 Server（不启动 Chrome 池）
func newServerWithTestDB(t *testing.T) *Server {
	t.Helper()
	db, err := database.NewDB(database.Options{Path: t.TempDir() + "/api_test.db"})
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rec := &models.Result{
		URL:           "https://example.com/",
		Title:         "Example",
		ResponseCode:  200,
		SchemaVersion: models.ResultSchemaVersion,
		ProbedAt:      models.Now().Add(-1 * time.Hour),
	}
	if err := db.SaveResult(rec); err != nil {
		t.Fatalf("SaveResult failed: %v", err)
	}

	s := &Server{
		Options: ServerOptions{},
		Router:  mux.NewRouter(),
	}
	s.SetDB(db)
	s.SetupRoutes()
	return s
}

func TestHandleListResults_HappyPath(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results?limit=10", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal err: %v", err)
	}
	if !resp.Success {
		t.Error("expected Success=true")
	}
	if resp.Data == nil {
		t.Error("expected non-nil Data")
	}
}

func TestHandleGetResult_HappyPath(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results/1", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetResult_NotFound(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results/9999", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleGetResultByURL_HappyPath(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results/by-url?url=https://example.com/", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetResultByHost_HappyPath(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results/by-host?host=example.com", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

// TestHandleListResults_DBNotConfigured — 未注入 DB 时返回 503
func TestHandleListResults_DBNotConfigured(t *testing.T) {
	s := &Server{Options: ServerOptions{}, Router: mux.NewRouter()}
	s.SetupRoutes()
	req := httptest.NewRequest(http.MethodGet, "/results", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

// TestHandleGetResult_InvalidID — id 非数字时返回 400
func TestHandleGetResult_InvalidID(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results/abc", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestHandleListResults_InvalidLimit — 非法 limit 回退默认值，均返回 200
func TestHandleListResults_InvalidLimit(t *testing.T) {
	cases := []string{
		"/results?limit=abc",
		"/results?limit=-5",
		"/results?limit=99999",
	}
	for _, path := range cases {
		path := path
		t.Run(path, func(t *testing.T) {
			s := newServerWithTestDB(t)
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			s.Router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

// TestHandleGetResultByURL_MissingParam — 缺少 url 参数返回 400
func TestHandleGetResultByURL_MissingParam(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results/by-url", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestHandleGetResultByHost_MissingParam — 缺少 host 参数返回 400
func TestHandleGetResultByHost_MissingParam(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results/by-host", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// TestHandleListResults_VerifyDataContent — 验证 Data 反序列化后回传的 URL 与写入一致
func TestHandleListResults_VerifyDataContent(t *testing.T) {
	s := newServerWithTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/results?limit=10", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// APIResponse.Data 在 JSON 反序列化后类型为 interface{}，需二次解析出具体数组
	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response err: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected Success=true")
	}

	var results []models.Result
	if err := json.Unmarshal(resp.Data, &results); err != nil {
		t.Fatalf("unmarshal Data err: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].URL != "https://example.com/" {
		t.Errorf("expected URL=https://example.com/, got %q", results[0].URL)
	}
}

// TestHandleListResults_TruncationByLimit 覆盖 HandleListResults 的
// screenshots>limit 截断分支（line 57-59）：插入多条数据，limit 小于总数。
func TestHandleListResults_TruncationByLimit(t *testing.T) {
	db, err := database.NewDB(database.Options{Path: t.TempDir() + "/trunc.db"})
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// 插入 5 条不同 URL 的记录
	for i := 0; i < 5; i++ {
		rec := &models.Result{
			URL:           "https://example" + string(rune('a'+i)) + ".com/",
			Title:         "Example",
			ResponseCode:  200,
			SchemaVersion: models.ResultSchemaVersion,
			ProbedAt:      models.Now().Add(-time.Duration(i) * time.Hour),
		}
		if err := db.SaveResult(rec); err != nil {
			t.Fatalf("SaveResult failed: %v", err)
		}
	}

	s := &Server{Options: ServerOptions{}, Router: mux.NewRouter()}
	s.SetDB(db)
	s.SetupRoutes()

	req := httptest.NewRequest(http.MethodGet, "/results?limit=2", nil)
	rec := httptest.NewRecorder()
	s.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var resp APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	results, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Data 类型错误: %T", resp.Data)
	}
	// limit=2 应截断为 2 条
	if len(results) != 2 {
		t.Errorf("limit=2 应返回 2 条结果, got %d", len(results))
	}
}
