package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/cyberspacesec/snir-skills/pkg/database"
	"github.com/cyberspacesec/snir-skills/pkg/models"

	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			SendJSONResponse(w, http.StatusNotFound, APIResponse{Success: false, Error: "未找到 id=" + idStr + " 的结果"})
		} else {
			SendJSONResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "查询数据库失败: " + err.Error()})
		}
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
