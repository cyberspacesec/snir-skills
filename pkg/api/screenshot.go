package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/runner"
	"github.com/gorilla/mux"
)

// HandleScreenshot 处理单个URL的截图请求
func (s *Server) HandleScreenshot(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var req ScreenshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("解析请求失败", "error", err)
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "无效的请求格式",
		})
		return
	}

	// 验证URL
	if req.URL == "" {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "URL不能为空",
		})
		return
	}

	// 确保URL包含协议前缀
	req.URL = ensureProtocol(req.URL, req.HTTPS, req.HTTP)

	// 验证URL格式
	_, err := url.Parse(req.URL)
	if err != nil {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "无效的URL格式",
		})
		return
	}

	// 创建Runner选项
	opts := createRunnerOptions(req, s.Options)

	// 处理截图请求
	log.Info("处理截图请求", "url", req.URL)
	result, err := s.ProcessScreenshot(req, opts)
	if err != nil {
		log.Error("截图失败", "url", req.URL, "error", err)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("截图失败: %v", err),
		})
		return
	}

	// 返回结果
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "截图成功",
		Data:    result,
	})
}

// HandleGetScreenshot 获取已保存的截图
func (s *Server) HandleGetScreenshot(w http.ResponseWriter, r *http.Request) {
	// 获取文件名参数
	vars := mux.Vars(r)
	filename := vars["filename"]

	// 验证文件名
	if filename == "" || strings.Contains(filename, "..") {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "无效的文件名",
		})
		return
	}

	// 构建文件路径
	filepath := filepath.Join(s.Options.ScreenshotPath, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		SendJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "文件不存在",
		})
		return
	}

	// 读取文件
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Error("读取文件失败", "filepath", filepath, "error", err)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "读取文件失败",
		})
		return
	}

	// 设置内容类型
	contentType := "image/png"
	if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		contentType = "image/jpeg"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	w.Write(content)
}

// createRunnerOptions 从请求中创建Runner选项
func createRunnerOptions(req ScreenshotRequest, serverOpts ServerOptions) runner.Options {
	opts := runner.Options{}

	// 设置Chrome选项
	opts.Chrome.UserAgent = req.UserAgent
	opts.Chrome.Proxy = req.Proxy
	opts.Chrome.Headless = true

	// 设置超时和延迟
	if req.Timeout > 0 {
		opts.Chrome.Timeout = req.Timeout
	} else {
		opts.Chrome.Timeout = 30 // 默认30秒
	}

	if req.Delay > 0 {
		opts.Chrome.Delay = req.Delay
	}

	// 设置截图选项
	opts.Scan.ScreenshotPath = serverOpts.ScreenshotPath
	opts.Scan.ScreenshotFormat = "png"
	opts.Scan.ScreenshotQuality = 90

	// 设置黑名单选项
	opts.Scan.EnableBlacklist = serverOpts.EnableBlacklist
	opts.Scan.DefaultBlacklist = serverOpts.DefaultBlacklist
	opts.Scan.BlacklistPatterns = serverOpts.BlacklistPatterns
	opts.Scan.BlacklistFile = serverOpts.BlacklistFile

	// 设置JavaScript选项
	opts.Scan.JavaScript = req.JavaScript

	// 忽略证书错误
	opts.Chrome.IgnoreCertErrors = req.IgnoreCertErrors

	return opts
}

// ensureProtocol 确保URL包含协议前缀
func ensureProtocol(url string, https, http bool) string {
	// 如果URL为空，直接返回
	if url == "" {
		return ""
	}

	// 如果已经包含协议，则直接返回
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.Contains(url, "://") {
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

// HandleListScreenshots 列出保存的截图
func (s *Server) HandleListScreenshots(w http.ResponseWriter, r *http.Request) {
	// 检查截图目录是否存在
	if _, err := os.Stat(s.Options.ScreenshotPath); os.IsNotExist(err) {
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "截图目录不存在",
		})
		return
	}

	// 列出目录中的所有文件
	files, err := ioutil.ReadDir(s.Options.ScreenshotPath)
	if err != nil {
		log.Error("读取截图目录失败", "path", s.Options.ScreenshotPath, "error", err)
		SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "读取截图目录失败",
		})
		return
	}

	// 过滤出图片文件
	var screenshots []ScreenshotInfo
	for _, file := range files {
		// 跳过目录和非图片文件
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		// 提取URL信息（假设文件名格式为：域名_时间戳.png）
		url := strings.TrimSuffix(filename, ext)
		parts := strings.Split(url, "_")
		if len(parts) > 1 {
			url = parts[0]
		}

		// 添加到结果列表
		screenshots = append(screenshots, ScreenshotInfo{
			Filename:  filename,
			URL:       url,
			Size:      file.Size(),
			CreatedAt: file.ModTime(),
		})
	}

	// 返回结果
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    screenshots,
	})
}
