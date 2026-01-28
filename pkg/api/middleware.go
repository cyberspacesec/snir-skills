package api

import (
	"net/http"
	"strings"

	"github.com/cyberspacesec/go-snir/pkg/log"
)

// CreateAuthMiddleware 创建API认证中间件
func (s *Server) CreateAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过特定路径的认证
			if r.URL.Path == "/" || r.URL.Path == "/health" || r.URL.Path == "/stats" ||
				strings.HasPrefix(r.URL.Path, "/screenshots/") || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// 没有设置API密钥时跳过认证
			if s.Options.APIKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// 从请求头中获取API密钥
			apiKey := r.Header.Get("X-API-Key")

			// 如果请求头中没有API密钥，则从URL参数中获取
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			// 检查API密钥是否正确
			if apiKey != s.Options.APIKey {
				log.Warn("API认证失败", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
				SendJSONResponse(w, http.StatusUnauthorized, APIResponse{
					Success: false,
					Error:   "无效的API密钥",
				})
				return
			}

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}
