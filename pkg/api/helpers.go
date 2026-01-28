package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cyberspacesec/go-snir/pkg/islazy"
	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/models"
)

// CreateScreenshotDir 创建截图目录
func CreateScreenshotDir(path string) (string, error) {
	return islazy.CreateDir(path)
}

// GetImageContentType 获取图像文件的内容类型
func GetImageContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

// IsImageFile 检查文件是否为图像文件
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif"
}

// SendJSONResponse 发送JSON响应
func SendJSONResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
	w.WriteHeader(statusCode)

	// 序列化响应
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Error("序列化JSON响应失败", "error", err)
		w.Write([]byte(`{"success":false,"error":"内部服务器错误"}`))
		return
	}

	w.Write(responseJSON)
}

// UrlWithProtocol 确保URL包含协议前缀
func UrlWithProtocol(url string, https, http bool) string {
	if url == "" {
		return ""
	}

	// 如果已经包含协议，则直接返回
	if UrlHasProtocol(url) {
		return url
	}

	// 根据配置添加协议前缀
	if https {
		return "https://" + url
	} else if http {
		return "http://" + url
	}

	// 默认使用HTTPS
	return "https://" + url
}

// UrlHasProtocol 检查URL是否已包含协议前缀
func UrlHasProtocol(url string) bool {
	return len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://")
}

// Write 实现 runner.Writer 接口的写入方法
func (w *MemoryWriter) Write(result *interface{}) error {
	if r, ok := (*result).(*models.Result); ok {
		w.mu.Lock()
		w.Results = append(w.Results, r)
		w.mu.Unlock()
	}
	return nil
}

// Close 实现 runner.Writer 接口的关闭方法
func (w *MemoryWriter) Close() error {
	return nil
}
