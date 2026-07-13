# AI Agent 历史结果检索 API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 让 AI Agent 能通过 HTTP API 和 Go SDK 检索已扫描的历史 Result（按 id / url / host / 列表），而非每次只能重新触发扫描。释放 `pkg/database` 已实现但未暴露的查询能力。

**Architecture:** 数据流：Agent 发起 `GET /results?...` → `pkg/api/results.go` handler → `Server.db`（`*database.DB`）→ database 查询方法返回 `*Screenshot` → `Screenshot.ToResult()` 转成 `*models.Result` → 经 `SendJSONResponse` 包成 `APIResponse{Data: results}` 返回。关键组件：`pkg/database/database.go` 新增 `GetScreenshotsByHost` + `GetScreenshotsByURL`（列表版）；`pkg/api/types.go` 给 `Server` 加 `db` 字段、给 `ServerOptions` 加 `DBPath`；`pkg/api/results.go` 新增 4 个 handler；`cmd/api.go` 注入 `--db-path` 打开 DB。为什么这样做：`database` 层已有 `GetScreenshot`/`GetScreenshotByURL`/`GetAllScreenshots`/`ExportResults` 但 `pkg/api` 从未引用 database 包（grep 0 命中），三层（SDK/API/CLI）全无检索入口——这是调研确认的最大结构性缺口。

**Tech Stack:** Go 1.23, chromedp v0.13.0, gorilla/mux v1.8.1, GORM v1.25.12, cobra v1.9.1

**Risks:**
- Task 2 修改共享的 `Server` 结构体（`pkg/api/types.go:175`）和 `NewServer`（`server_methods.go:15`），可能影响现有 API 启动 → 缓解：`db` 字段设为可选，未配置 DBPath 时新端点返回 503 `{"error":"未启用数据库"}`，不阻塞 `NewServer` 也不影响现有 `/screenshot`/`/batch`
- Task 2 的 `GetScreenshotByURL` 只取一条（`database.go:110` 用 `First`），但 Agent 可能要"某 URL 所有历史扫描记录" → 缓解：新增 `GetScreenshotsByURL` 返回列表（保留旧的单条方法不破坏兼容）
- Task 2 行号会因新增方法偏移，影响 Task 3/4 的行号引用 → 缓解：Task 3/4 用函数名 + 上下文描述定位，不硬依赖 Task 2 后的行号
- Task 4 SDK 方法需与 API 端点契约完全一致 → 缓解：Task 4 在 Task 2/3 端点契约固化后编写，先固化响应 JSON 形状再写 SDK

---

### Task 1: database 层新增 host 模糊查询与 URL 列表查询

**Depends on:** None
**Files:**
- Modify: `pkg/database/database.go:119`（在 `GetAllScreenshots` 函数之后插入两个新方法）
- Test: `pkg/database/host_query_test.go`（新建）

- [ ] **Step 1: 新增 GetScreenshotsByHost 方法 — 按 host 前缀模糊查询历史结果**

文件: `pkg/database/database.go:119`（在 `GetAllScreenshots` 函数闭合 `}` 之后、`CreateScanSession` 函数之前插入）

```go
// GetScreenshotsByHost 通过 host 前缀模糊查询截图记录
// host 不区分大小写，匹配 host 字段前缀（如 "example.com" 匹配 "example.com" 及其子域不匹配，仅 host 字段精确或前缀）
func (d *DB) GetScreenshotsByHost(host string) ([]*Screenshot, error) {
	var screenshots []*Screenshot
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("host 不能为空")
	}
	if err := d.db.Where("host LIKE ?", host+"%").Order("probed_at DESC").Find(&screenshots).Error; err != nil {
		return nil, err
	}
	return screenshots, nil
}

// GetScreenshotsByURL 通过 URL 查询该 URL 的所有历史扫描记录（按时间倒序）
// 与 GetScreenshotByURL 不同，本方法返回列表而非单条最新
func (d *DB) GetScreenshotsByURL(url string) ([]*Screenshot, error) {
	var screenshots []*Screenshot
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("url 不能为空")
	}
	if err := d.db.Where("url = ?", url).Order("probed_at DESC").Find(&screenshots).Error; err != nil {
		return nil, err
	}
	return screenshots, nil
}
```

- [ ] **Step 2: 检查 database.go 是否已 import strings — 若无则添加**

文件: `pkg/database/database.go:1-15`（import 块）

```go
import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/models"

	"gorm.io/gorm"
)
```

- [ ] **Step 3: 创建 host 模糊查询单元测试 — 覆盖 host 匹配、URL 列表、空输入边界**

```go
// pkg/database/host_query_test.go
package database

import (
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// newTestDBForResultQuery 构造内存 DB 并插入测试记录
func newTestDBForResultQuery(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(Options{Path: t.TempDir() + "/test_results.db"})
	if err != nil {
		t.Fatalf("NewDB failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// 通过公开的 SaveResult 插入记录（内部调 FromResult + EnrichEndpoint）
	now := models.Now()
	records := []*models.Result{
		{URL: "https://example.com/", Title: "Example", ResponseCode: 200, ProbedAt: now.Add(-2 * time.Hour)},
		{URL: "https://example.com/", Title: "Example v2", ResponseCode: 301, ProbedAt: now.Add(-1 * time.Hour)},
		{URL: "https://other.com/", Title: "Other", ResponseCode: 200, ProbedAt: now},
	}
	for _, r := range records {
		r.SchemaVersion = models.ResultSchemaVersion
		if err := db.SaveResult(r); err != nil {
			t.Fatalf("SaveResult failed: %v", err)
		}
	}
	return db
}

// TestGetScreenshotsByHost_HappyPath — 按 host 前缀匹配返回对应记录
func TestGetScreenshotsByHost_HappyPath(t *testing.T) {
	db := newTestDBForResultQuery(t)
	got, err := db.GetScreenshotsByHost("example.com")
	if err != nil {
		t.Fatalf("GetScreenshotsByHost err: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d records for example.com, want 2", len(got))
	}
}

// TestGetScreenshotsByHost_NoMatch — 不存在的 host 返回空切片无错误
func TestGetScreenshotsByHost_NoMatch(t *testing.T) {
	db := newTestDBForResultQuery(t)
	got, err := db.GetScreenshotsByHost("nonexistent.com")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d records for nonexistent host, want 0", len(got))
	}
}

// TestGetScreenshotsByHost_EmptyInput — 空 host 返回错误不查询
func TestGetScreenshotsByHost_EmptyInput(t *testing.T) {
	db := newTestDBForResultQuery(t)
	if _, err := db.GetScreenshotsByHost("  "); err == nil {
		t.Error("expected error for empty host, got nil")
	}
}

// TestGetScreenshotsByURL_ReturnsAllHistory — 同一 URL 的所有历史记录按时间倒序返回
func TestGetScreenshotsByURL_ReturnsAllHistory(t *testing.T) {
	db := newTestDBForResultQuery(t)
	got, err := db.GetScreenshotsByURL("https://example.com/")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
	// 时间倒序：第一条应是最近扫描的（title=Example v2，probed_at 较晚）
	if got[0].Title != "Example v2" {
		t.Errorf("first record title = %q, want %q (latest by probed_at DESC)", got[0].Title, "Example v2")
	}
}

// TestGetScreenshotsByURL_NoMatch — 不存在的 URL 返回空切片无错误
func TestGetScreenshotsByURL_NoMatch(t *testing.T) {
	db := newTestDBForResultQuery(t)
	got, err := db.GetScreenshotsByURL("https://nope.com/")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d, want 0", len(got))
	}
}
```

- [ ] **Step 4: 验证 database 层新增查询方法**
Run: `go test ./pkg/database/ -run "TestGetScreenshotsByHost|TestGetScreenshotsByURL" -count=1 -v`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

- [ ] **Step 5: 提交**
Run: `git add pkg/database/database.go pkg/database/host_query_test.go && git commit -m "feat(database): add GetScreenshotsByHost and GetScreenshotsByURL list queries"`

---

### Task 2: API 层注入 DB 并新增 4 个结果检索端点

**Depends on:** Task 1
**Files:**
- Modify: `pkg/api/types.go:154-182`（ServerOptions 加 DBPath，Server 加 db 字段）
- Modify: `pkg/api/server_methods.go:15-30`（NewServer 注入 DB 初始化）
- Create: `pkg/api/results.go`（4 个 handler + SetDB 方法）
- Modify: `pkg/api/server_methods.go:115`（SetupRoutes 注册新路由）
- Test: `pkg/api/results_test.go`（新建）

- [ ] **Step 1: 扩展 ServerOptions 和 Server 结构体 — 增加 DBPath 与 db 字段**

文件: `pkg/api/types.go:154-182`（替换 ServerOptions 与 Server 两个 struct）

```go
type ServerOptions struct {
	// 服务器配置
	Port           int    `json:"port"`
	Host           string `json:"host"`
	APIKey         string `json:"api_key"`
	ScreenshotPath string `json:"screenshot_path"`
	DBPath         string `json:"db_path"` // 数据库文件路径，非空则启用历史结果检索端点

	// 批处理配置
	MaxBatchSize          int `json:"max_batch_size"`
	MaxConcurrency        int `json:"max_concurrency"`
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
	RequestQueueSize      int `json:"request_queue_size"`

	// 黑名单配置
	EnableBlacklist   bool     `json:"enable_blacklist"`
	DefaultBlacklist  bool     `json:"default_blacklist"`
	BlacklistPatterns []string `json:"blacklist_patterns"`
	BlacklistFile     string   `json:"blacklist_file"`
}

// Server 表示API服务器
type Server struct {
	Options          ServerOptions
	Router           *mux.Router
	concurrencyLimit interface{}        // 并发限制器
	shutdownCh       chan struct{}      // 关闭通道
	serverStartTime  time.Time          // 服务器启动时间
	pool             *runner.DriverPool // 浏览器连接池，复用 Chrome 进程
	db               *database.DB       // 数据库，为 nil 时结果检索端点返回 503
}
```

- [ ] **Step 2: 创建 results.go — 实现 4 个结果检索 handler 与 SetDB 方法**

```go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/cyberspacesec/snir-skills/pkg/database"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// SetDB 注入数据库实例，启用历史结果检索端点。
// 必须在 SetupRoutes 之前调用。传入 nil 或未调用时，检索端点返回 503。
func (s *Server) SetDB(db *database.DB) {
	s.db = db
}

// dbReady 检查数据库是否可用，不可用时写 503 响应并返回 false。
func (s *Server) dbReady(w http.ResponseWriter) bool {
	if s.db == nil {
		SendJSONResponse(w, http.StatusServiceUnavailable, APIResponse{
			Success: false,
			Error:   "未启用数据库：请使用 --db-path 指定数据库文件路径以启用历史结果检索",
		})
		return false
	}
	return true
}

// HandleListResults GET /results — 列出所有历史扫描结果（按 probed_at 倒序）
// 查询参数: ?limit=N（默认 100，上限 1000）
func (s *Server) HandleListResults(w http.ResponseWriter, r *http.Request) {
	if !s.dbReady(w) {
		return
	}
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 1000 {
		limit = 1000
	}
	screenshots, err := s.db.GetAllScreenshots()
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "查询数据库失败: " + err.Error(),
		})
		return
	}
	// 截断到 limit（GetAllScreenshots 无分页，这里在内存截断）
	if len(screenshots) > limit {
		screenshots = screenshots[:limit]
	}
	results := make([]*models.Result, 0, len(screenshots))
	for _, sc := range screenshots {
		results = append(results, sc.ToResult())
	}
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "成功获取历史结果列表",
		Data:    results,
	})
}

// HandleGetResult GET /results/{id} — 按主键 id 检索单个历史结果
func (s *Server) HandleGetResult(w http.ResponseWriter, r *http.Request) {
	if !s.dbReady(w) {
		return
	}
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "缺少结果 id"})
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "id 必须为正整数"})
		return
	}
	sc, err := s.db.GetScreenshot(uint(id))
	if err != nil {
		SendJSONResponse(w, http.StatusNotFound, APIResponse{Success: false, Error: "未找到 id=" + idStr + " 的结果"})
		return
	}
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "成功获取结果",
		Data:    sc.ToResult(),
	})
}

// HandleGetResultByURL GET /results/by-url — 按精确 URL 查询该 URL 的所有历史扫描记录
// 查询参数: ?url=...（必填）
func (s *Server) HandleGetResultByURL(w http.ResponseWriter, r *http.Request) {
	if !s.dbReady(w) {
		return
	}
	url := r.URL.Query().Get("url")
	if url == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "缺少 url 查询参数"})
		return
	}
	screenshots, err := s.db.GetScreenshotsByURL(url)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "查询失败: " + err.Error()})
		return
	}
	results := make([]*models.Result, 0, len(screenshots))
	for _, sc := range screenshots {
		results = append(results, sc.ToResult())
	}
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "成功获取 URL 历史结果",
		Data:    results,
	})
}

// HandleGetResultByHost GET /results/by-host — 按 host 前缀模糊查询历史结果
// 查询参数: ?host=...（必填）
func (s *Server) HandleGetResultByHost(w http.ResponseWriter, r *http.Request) {
	if !s.dbReady(w) {
		return
	}
	host := r.URL.Query().Get("host")
	if host == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "缺少 host 查询参数"})
		return
	}
	screenshots, err := s.db.GetScreenshotsByHost(host)
	if err != nil {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "查询失败: " + err.Error()})
		return
	}
	results := make([]*models.Result, 0, len(screenshots))
	for _, sc := range screenshots {
		results = append(results, sc.ToResult())
	}
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "成功获取 host 历史结果",
		Data:    results,
	})
}

// 保留一个未使用的引用以防 go fmt 报 json 导入未用（如未来扩展错误细节用 json.RawMessage）
var _ = json.RawMessage("")
```

- [ ] **Step 3: 在 SetupRoutes 中注册 4 个新路由 — 在 /health 之后追加**

文件: `pkg/api/server_methods.go:115`（在 `s.Router.HandleFunc("/health", HandleHealth)...` 这一行之后、函数闭合 `}` 之前插入）

```go
	s.Router.HandleFunc("/health", HandleHealth).Methods("GET", "OPTIONS")
	// 历史结果检索端点（需配合 --db-path 启用）
	s.Router.HandleFunc("/results", s.HandleListResults).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/results/{id}", s.HandleGetResult).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/results/by-url", s.HandleGetResultByURL).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/results/by-host", s.HandleGetResultByHost).Methods("GET", "OPTIONS")
```

- [ ] **Step 4: 在 server_methods.go 顶部 import 块添加 database 包**

文件: `pkg/api/server_methods.go:1-15`（import 块）

```go
import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cyberspacesec/snir-skills/pkg/database"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)
```

- [ ] **Step 5: 创建 results_test.go — 验证 4 个端点的 happy path 与 503 兜底**

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	// 通过公开的 SaveResult 插入测试数据（内部调 FromResult + EnrichEndpoint）
	rec := &models.Result{
		URL:           "https://example.com/",
		Title:         "Example",
		ResponseCode:  200,
		ProbedAt:      models.Now().Add(-1 * time.Hour),
		SchemaVersion: models.ResultSchemaVersion,
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
```

- [ ] **Step 6: 验证 API 层新增端点测试通过**
Run: `go test ./pkg/api/ -run "TestHandleListResults|TestHandleGetResult" -count=1 -v`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

- [ ] **Step 7: 提交**
Run: `git add pkg/api/types.go pkg/api/server_methods.go pkg/api/results.go pkg/api/results_test.go && git commit -m "feat(api): add /results endpoints for historical Result retrieval"`

---

### Task 3: cmd/api.go 接入 --db-path flag 并端到端验证

**Depends on:** Task 2
**Files:**
- Modify: `cmd/api.go:40-50`（构造 ServerOptions 时传 DBPath + 初始化 DB）
- Modify: `cmd/api.go:75-85`（init 函数新增 --db-path flag）

- [ ] **Step 1: cmd/api.go 构造 ServerOptions 时传入 DBPath 并初始化 DB**

文件: `cmd/api.go:40-55`（在 `server := api.NewServer(apiOptions)` 这一行之前插入 DB 初始化，并在 apiOptions 字面量中加 DBPath 字段）

替换 apiOptions 字面量结尾与 NewServer 调用之间的区块：

```go
			MaxConcurrentRequests: opts.API.MaxConcurrent,
			RequestQueueSize:       opts.API.QueueSize,
			DBPath:                 opts.DB.Path,
		}

		// 创建API服务
		server := api.NewServer(apiOptions)

		// 若指定了 db-path，打开数据库以启用历史结果检索端点
		if opts.DB.Path != "" {
			db, err := database.NewDB(database.Options{Path: opts.DB.Path})
			if err != nil {
				log.Error("打开数据库失败，历史结果检索端点不可用", "error", err)
			} else {
				server.SetDB(db)
				log.Info("历史结果检索已启用", "db_path", log.Cyan(opts.DB.Path))
			}
		}
```

- [ ] **Step 2: cmd/api.go init 函数新增 --db-path flag**

文件: `cmd/api.go:75-95`（init 函数中，在 `--api-key` flag 注册之后添加）

```go
	apiCmd.Flags().StringVar(&opts.API.APIKey, "api-key", "", log.Cyan("API密钥，用于API鉴权，如不指定则自动生成"))
	apiCmd.Flags().StringVar(&opts.DB.Path, "db-path", "", log.Cyan("数据库文件路径，启用后 /results 系列端点可检索历史扫描结果"))
```

- [ ] **Step 3: cmd/api.go 顶部 import 块添加 database 包**

文件: `cmd/api.go:1-20`（import 块）

```go
import (
	"fmt"

	"github.com/cyberspacesec/snir-skills/internal/options"
	"github.com/cyberspacesec/snir-skills/pkg/api"
	"github.com/cyberspacesec/snir-skills/pkg/database"
	"github.com/cyberspacesec/snir-skills/pkg/log"

	"github.com/spf13/cobra"
)
```

- [ ] **Step 4: 编译验证 — 确认 cmd/api.go 引用正确**
Run: `go build ./cmd/ 2>&1 | head -20`
Expected:
  - Exit code: 0
  - Output does NOT contain: "error" or "undefined"

- [ ] **Step 5: 端到端验证 — 先扫一个 URL 入库，再用 /results 检索**
Run: `make build && cd /tmp && rm -rf snir-results && mkdir snir-results && cd snir-results && /home/cc11001100/github/cyberspacesec/snir-skills/snir scan example.com --db --db-path results.db --write-jsonl >/dev/null 2>&1 && /home/cc11001100/github/cyberspacesec/snir-skills/snir api --host 127.0.0.1 --port 19095 --api-key rtest --db-path /tmp/snir-results/results.db >/tmp/results-api.log 2>&1 & sleep 5 && curl -s http://127.0.0.1:19095/results -H "X-API-Key: rtest" | python3 -c "import sys,json; r=json.load(sys.stdin); d=r.get('data') or []; print('results count:', len(d)); print('first url:', d[0].get('url') if d else 'NONE'); assert len(d)>=1, 'should have at least 1 result'; assert d[0].get('response_code')==200, 'first result code should be 200'" && curl -s "http://127.0.0.1:19095/results/by-host?host=example.com" -H "X-API-Key: rtest" | python3 -c "import sys,json; r=json.load(sys.stdin); d=r.get('data') or []; print('by-host count:', len(d)); assert len(d)>=1" && pkill -f "snir.*api.*19095" 2>/dev/null; echo done`
Expected:
  - Exit code: 0
  - Output contains: "results count: 1"
  - Output contains: "first url: https://example.com"
  - Output contains: "by-host count: 1"

- [ ] **Step 6: 提交**
Run: `git add cmd/api.go && git commit -m "feat(cmd): add --db-path flag to api command for results retrieval"`

---


### Task 4: SDK 新增 HTTPClient 结果检索方法与文档更新

**Depends on:** Task 3
**Files:**
- Create: `pkg/sdk/http_client.go`（独立 HTTPClient 类型 + 3 个检索方法）
- Test: `pkg/sdk/http_client_test.go`（新建）
- Create: `website/api/endpoints-results.md`（端点文档）
- Modify: `website/.vitepress/config.ts`（sidebar 挂接新文档）

> **设计说明：** 现有 `pkg/sdk/Client`（client.go:54）是进程内 driver 封装（`driverPool` 直连 Chrome），整个包无 `net/http` 引用。结果检索必须走 HTTP API，因此新增与 `Client` 平行的独立 `HTTPClient` 类型，不污染现有 driver 路径。这符合现有架构——SDK 已有 `Client`（driver）、`NewRemoteClient`（ws），再加 `HTTPClient`（http）是自然扩展。

- [ ] **Step 1: 创建 pkg/sdk/http_client.go — 独立 HTTP 客户端 + 3 个结果检索方法**

```go
package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// HTTPClient 是 snir HTTP API 的轻量客户端，用于结果检索等只读端点。
// 与 Client（进程内 driver）不同，HTTPClient 不启动 Chrome，仅发 HTTP 请求。
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// HTTPClientOptions 配置 HTTPClient
type HTTPClientOptions struct {
	BaseURL string        // snir api 地址，如 "http://127.0.0.1:8080"
	APIKey  string        // X-API-Key 鉴权密钥
	Timeout time.Duration
}

// NewHTTPClient 创建 HTTP API 客户端
func NewHTTPClient(opts HTTPClientOptions) *HTTPClient {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPClient{
		baseURL:    opts.BaseURL,
		apiKey:     opts.APIKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// addAuth 注入鉴权头
func (c *HTTPClient) addAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}

// doRaw 执行请求返回 body + status
func (c *HTTPClient) doRaw(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// doSingleResult 发请求并解析 APIResponse 信封到单个 Result
func (c *HTTPClient) doSingleResult(req *http.Request) (*models.Result, error) {
	body, status, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}
	if status == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("服务端未启用数据库：请用 --db-path 启动 snir api")
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(body))
	}
	var apiResp struct {
		Success bool            `json:"success"`
		Error   string          `json:"error,omitempty"`
		Data    *models.Result  `json:"data,omitempty"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if !apiResp.Success {
		return nil, fmt.Errorf("%s", apiResp.Error)
	}
	return apiResp.Data, nil
}

// doListResult 发请求并解析 APIResponse 信封到 Result 切片
func (c *HTTPClient) doListResult(req *http.Request) ([]*models.Result, error) {
	body, status, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}
	if status == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("服务端未启用数据库：请用 --db-path 启动 snir api")
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(body))
	}
	var apiResp struct {
		Success bool             `json:"success"`
		Error   string           `json:"error,omitempty"`
		Data    []*models.Result `json:"data,omitempty"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if !apiResp.Success {
		return nil, fmt.Errorf("%s", apiResp.Error)
	}
	return apiResp.Data, nil
}

// GetResult 按主键 id 检索单个历史扫描结果。
func (c *HTTPClient) GetResult(id uint64) (*models.Result, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/results/%d", c.baseURL, id), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	return c.doSingleResult(req)
}

// GetResultByURL 按精确 URL 查询该 URL 的所有历史扫描记录（按 probed_at 倒序）。
func (c *HTTPClient) GetResultByURL(rawURL string) ([]*models.Result, error) {
	q := url.Values{}
	q.Set("url", rawURL)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/results/by-url?%s", c.baseURL, q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	return c.doListResult(req)
}

// ListResults 列出所有历史扫描结果（按 probed_at 倒序）。
// limit<=0 用服务端默认 100，>1000 截断为 1000。
func (c *HTTPClient) ListResults(limit int) ([]*models.Result, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/results?%s", c.baseURL, q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	return c.doListResult(req)
}
```

- [ ] **Step 2: 创建 pkg/sdk/http_client_test.go — 验证 3 个方法的 happy path / 503 / 404**

```go
package sdk

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newMockResultsServer(t *testing.T, handler http.HandlerFunc) (*HTTPClient, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewHTTPClient(HTTPClientOptions{BaseURL: srv.URL, APIKey: "test"})
	return c, srv.Close
}

func TestHTTPGetResult_HappyPath(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/results/1" {
			t.Errorf("path = %s, want /results/1", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test" {
			t.Errorf("api key header = %q", r.Header.Get("X-API-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":{"id":1,"url":"https://example.com/","response_code":200}}`))
	})
	defer cleanup()
	res, err := c.GetResult(1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res == nil || res.ResponseCode != 200 {
		t.Errorf("got %+v, want response_code=200", res)
	}
}

func TestHTTPGetResult_DBNotConfigured(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"success":false,"error":"未启用数据库"}`))
	})
	defer cleanup()
	if _, err := c.GetResult(1); err == nil {
		t.Error("expected error for 503, got nil")
	}
}

func TestHTTPGetResult_NotFound(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":"未找到 id=9999 的结果"}`))
	})
	defer cleanup()
	if _, err := c.GetResult(9999); err == nil {
		t.Error("expected error for 404, got nil")
	}
}

func TestHTTPGetResultByURL_HappyPath(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/results/by-url" {
			t.Errorf("path = %s, want /results/by-url", r.URL.Path)
		}
		if r.URL.Query().Get("url") != "https://example.com/" {
			t.Errorf("url param = %q", r.URL.Query().Get("url"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":[{"id":1,"url":"https://example.com/","response_code":200}]}`))
	})
	defer cleanup()
	results, err := c.GetResultByURL("https://example.com/")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ResponseCode != 200 {
		t.Errorf("response_code = %d, want 200", results[0].ResponseCode)
	}
}

func TestHTTPListResults_HappyPath(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/results" {
			t.Errorf("path = %s, want /results", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":[{"id":1},{"id":2}]}`))
	})
	defer cleanup()
	results, err := c.ListResults(0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}
```

- [ ] **Step 3: 验证 HTTPClient 测试通过**
Run: `go test ./pkg/sdk/ -run "TestHTTP" -count=1 -v`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

- [ ] **Step 4: 创建端点文档 website/api/endpoints-results.md — 让 AI Agent 自发现检索能力**

内容为 Markdown 文档，含端点表、鉴权说明、响应格式、curl 示例、Go SDK 示例（`sdk.NewHTTPClient`）。文档需包含：

- 顶部启用方式说明：`snir api --db-path <file>`，否则返回 503
- 端点表 4 行：`GET /results?limit=N`、`GET /results/{id}`、`GET /results/by-url?url=...`、`GET /results/by-host?host=...`
- 鉴权：`X-API-Key` 头或 `?api_key=` 参数
- 响应格式：统一 `APIResponse` 信封，`data` 为 `Result` 或 `Result` 数组
- curl 示例 3 条：`/results?limit=50`、`/results/by-url?url=...`、`/results/by-host?host=...`，均用 `jq` 管道
- Go SDK 示例：`sdk.NewHTTPClient(sdk.HTTPClientOptions{...})` + `GetResult` / `GetResultByURL` / `ListResults`
- 与扫描端点区别：`/screenshot` `/batch` 触发新扫描，`/results` 检索已存储历史，不发网络请求

- [ ] **Step 5: 在 website/.vitepress/config.ts 的 api sidebar 挂接新文档**

文件: `website/.vitepress/config.ts`（找到 `api` 侧边栏 items 数组，在现有 endpoint-* 项之后追加一项）

```typescript
        { text: '结果检索端点', link: '/api/endpoints-results' },
```

- [ ] **Step 6: 验证文档站构建通过**
Run: `cd website && npx vitepress build 2>&1 | tail -10`
Expected:
  - Exit code: 0
  - Output does NOT contain: "error" or "unresolved"

- [ ] **Step 7: 提交**
Run: `git add pkg/sdk/http_client.go pkg/sdk/http_client_test.go website/api/endpoints-results.md website/.vitepress/config.ts && git commit -m "feat(sdk): add HTTPClient with GetResult/GetResultByURL/ListResults and results API docs"`
