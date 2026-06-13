package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/cyberspacesec/snir-skills/pkg/log"
)

// HandleBatchScreenshot 批量处理多个URL的截图请求
func (s *Server) HandleBatchScreenshot(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var req BatchScreenshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("解析请求失败", "error", err)
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "无效的请求格式",
		})
		return
	}

	// 验证URL列表
	if len(req.URLs) == 0 {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "URL列表不能为空",
		})
		return
	}

	// 检查批量大小限制
	if s.Options.MaxBatchSize > 0 && len(req.URLs) > s.Options.MaxBatchSize {
		SendJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("URL数量超过限制 (最大 %d 个)", s.Options.MaxBatchSize),
		})
		return
	}

	// 预处理所有URL，确保包含协议前缀
	for i, url := range req.URLs {
		req.URLs[i] = ensureProtocol(url, req.HTTPS, req.HTTP)
	}

	// 设置并发数
	concurrency := 2 // 默认值
	if req.Threads > 0 && req.Threads <= 20 {
		concurrency = req.Threads
	}

	// 创建共享的Runner选项
	opts := createRunnerOptions(ScreenshotRequest{
		UserAgent:        req.UserAgent,
		Proxy:            req.Proxy,
		Timeout:          req.Timeout,
		Delay:            req.Delay,
		IgnoreCertErrors: req.IgnoreCertErrors,
		JavaScript:       req.JavaScript,
		JavaScriptFile:   req.JavaScriptFile,
		RunJSBefore:      req.RunJSBefore,
		RunJSAfter:       req.RunJSAfter,
		Fingerprint:      req.Fingerprint,
		Cookies:          req.Cookies,
		Selector:         req.Selector,
		XPath:            req.XPath,
		CaptureFullPage:  req.CaptureFullPage,
		Actions:          req.Actions,
		Form:             req.Form,
	}, s.Options)

	// 创建请求列表
	requests := make([]ScreenshotRequest, len(req.URLs))
	for i, url := range req.URLs {
		requests[i] = ScreenshotRequest{
			URL:              url,
			UserAgent:        req.UserAgent,
			Proxy:            req.Proxy,
			Timeout:          req.Timeout,
			Delay:            req.Delay,
			IgnoreCertErrors: req.IgnoreCertErrors,
			JavaScript:       req.JavaScript,
			JavaScriptFile:   req.JavaScriptFile,
			RunJSBefore:      req.RunJSBefore,
			RunJSAfter:       req.RunJSAfter,
			Fingerprint:      req.Fingerprint,
			Cookies:          req.Cookies,
			Selector:         req.Selector,
			XPath:            req.XPath,
			CaptureFullPage:  req.CaptureFullPage,
			Actions:          req.Actions,
			Form:             req.Form,
		}
	}

	// 创建结果通道
	resultsChan := make(chan BatchResult, len(requests))

	// 处理函数
	processor := func(req ScreenshotRequest) BatchResult {
		log.Info("处理批量截图任务", "url", req.URL)
		result, err := s.ProcessScreenshot(req, opts)
		if err != nil {
			log.Error("截图失败", "url", req.URL, "error", err)
			return BatchResult{
				URL:   req.URL,
				Error: err.Error(),
			}
		}
		return BatchResult{
			URL:    req.URL,
			Result: result,
		}
	}

	// 启动并发处理
	go ProcessConcurrent(requests, concurrency, processor, resultsChan)

	// 收集结果
	var results []BatchResult
	var errors []BatchError

	// 等待所有结果
	for i := 0; i < len(requests); i++ {
		result := <-resultsChan
		results = append(results, result)
		if result.Error != "" {
			errors = append(errors, BatchError{
				URL:   result.URL,
				Error: result.Error,
			})
		}
	}

	// 返回结果
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("批量处理完成，成功 %d 个，失败 %d 个", len(results)-len(errors), len(errors)),
		Data: map[string]interface{}{
			"results": results,
			"errors":  errors,
		},
	})
}

// ProcessBatchConcurrent 并发处理批量请求
func ProcessBatchConcurrent(requests []ScreenshotRequest, concurrency int, processorFunc func(ScreenshotRequest) BatchResult, resultChan chan<- BatchResult) {
	// 创建工作池
	limiter := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	// 处理每个请求
	for _, req := range requests {
		wg.Add(1)
		go func(r ScreenshotRequest) {
			defer wg.Done()

			// 获取处理许可
			limiter <- struct{}{}
			defer func() { <-limiter }()

			// 处理请求并发送结果
			result := processorFunc(r)
			resultChan <- result
		}(req)
	}

	// 等待所有任务完成
	wg.Wait()
}
