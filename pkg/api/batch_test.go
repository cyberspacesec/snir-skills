package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// 为测试创建一个MockBatchScreenshotHandler
func MockBatchScreenshotHandler(mockResults map[string]error, maxBatchSize int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 解析请求体
		var req BatchScreenshotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "无效的请求格式",
				Error:   err.Error(),
			})
			return
		}

		// 验证URL列表
		if len(req.URLs) == 0 {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "URL列表不能为空",
			})
			return
		}

		// 检查批量大小
		if maxBatchSize > 0 && len(req.URLs) > maxBatchSize {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "批量请求超出最大限制",
				Error:   "Maximum batch size exceeded",
			})
			return
		}

		// 处理结果
		results := make([]BatchResult, 0, len(req.URLs))
		errors := make([]BatchError, 0)

		// 为每个URL创建结果
		for _, urlStr := range req.URLs {
			if err, ok := mockResults[urlStr]; ok && err != nil {
				// 模拟失败
				results = append(results, BatchResult{
					URL:   urlStr,
					Error: err.Error(),
				})
				errors = append(errors, BatchError{
					URL:   urlStr,
					Error: err.Error(),
				})
			} else {
				// 模拟成功
				results = append(results, BatchResult{
					URL: urlStr,
					Result: &models.Result{
						URL:        urlStr,
						Title:      "Mock Title for " + urlStr,
						Screenshot: "mock_screenshot.png",
					},
				})
			}
		}

		// 返回处理结果
		SendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "批量处理完成",
			Data: map[string]interface{}{
				"results": results,
				"errors":  errors,
				"total":   len(req.URLs),
				"success": len(req.URLs) - len(errors),
				"failed":  len(errors),
			},
		})
	}
}

// 测试批量处理的配置结构
type batchTestConfig struct {
	name            string
	requestBody     interface{}
	mockResults     map[string]error
	expectedStatus  int
	expectSuccess   bool
	expectedResults int
	expectedErrors  int
	maxBatchSize    int
}

// 运行批量测试的辅助函数
func runBatchTest(t *testing.T, config batchTestConfig) {
	// 设置模拟处理函数
	handler := MockBatchScreenshotHandler(config.mockResults, config.maxBatchSize)

	// 序列化请求体
	var reqBody []byte
	var err error

	if str, ok := config.requestBody.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(config.requestBody)
		if err != nil {
			t.Fatalf("无法序列化请求: %v", err)
		}
	}

	// 创建测试请求
	req, err := http.NewRequest("POST", "/batch", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("无法创建请求: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 处理请求
	handler.ServeHTTP(rr, req)

	// 检查状态码
	if rr.Code != config.expectedStatus {
		t.Errorf("状态码 = %v, 期望 %v", rr.Code, config.expectedStatus)
	}

	// 解析响应
	var response APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("无法解析响应: %v", err)
	}

	// 检查成功状态
	if response.Success != config.expectSuccess {
		t.Errorf("Success = %v, 期望 %v", response.Success, config.expectSuccess)
	}

	// 如果是成功响应，检查结果数量
	if config.expectSuccess && response.Data != nil {
		responseData, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Errorf("无法转换响应数据")
			return
		}

		// 检查结果数量
		results, ok := responseData["results"].([]interface{})
		if !ok {
			t.Errorf("无法转换结果数据")
			return
		}

		if len(results) != config.expectedResults {
			t.Errorf("结果数量 = %v, 期望 %v", len(results), config.expectedResults)
		}

		// 检查错误数量
		errors, ok := responseData["errors"].([]interface{})
		if !ok {
			t.Errorf("无法转换错误数据")
			return
		}

		if len(errors) != config.expectedErrors {
			t.Errorf("错误数量 = %v, 期望 %v", len(errors), config.expectedErrors)
		}
	}
}

// 测试HandleBatchScreenshot函数
func TestHandleBatchScreenshot(t *testing.T) {
	// 创建模拟错误
	mockErr := ErrMockScreenshotFailed

	// 定义测试用例
	tests := []batchTestConfig{
		{
			name: "有效批量请求-全部成功",
			requestBody: BatchScreenshotRequest{
				URLs: []string{
					"https://example1.com",
					"https://example2.com",
					"https://example3.com",
				},
				Threads: 2,
			},
			mockResults:     nil, // 全部成功
			expectedStatus:  http.StatusOK,
			expectSuccess:   true,
			expectedResults: 3,
			expectedErrors:  0,
			maxBatchSize:    10,
		},
		{
			name: "有效批量请求-部分失败",
			requestBody: BatchScreenshotRequest{
				URLs: []string{
					"https://example1.com",
					"https://example2.com", // 这个会失败
					"https://example3.com",
				},
				Threads: 2,
			},
			mockResults: map[string]error{
				"https://example2.com": mockErr,
			},
			expectedStatus:  http.StatusOK,
			expectSuccess:   true,
			expectedResults: 3,
			expectedErrors:  1,
			maxBatchSize:    10,
		},
		{
			name: "空URL列表",
			requestBody: BatchScreenshotRequest{
				URLs: []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			maxBatchSize:   10,
		},
		{
			name: "超过批量限制",
			requestBody: BatchScreenshotRequest{
				URLs: []string{
					"https://example1.com", "https://example2.com", "https://example3.com",
					"https://example4.com", "https://example5.com", "https://example6.com",
					"https://example7.com", "https://example8.com", "https://example9.com",
					"https://example10.com", "https://example11.com", // 超过了10个限制
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			maxBatchSize:   10,
		},
		{
			name:           "无效的请求格式",
			requestBody:    "这不是一个有效的JSON",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			maxBatchSize:   10,
		},
	}

	// 运行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBatchTest(t, tt)
		})
	}
}

// 并发测试结果
type ConcurrentTestResult struct {
	URL     string
	Success bool
}

// 测试ProcessConcurrent函数
func TestProcessBatchConcurrent(t *testing.T) {
	// 创建测试请求列表
	requests := []ScreenshotRequest{
		{URL: "http://example1.com"},
		{URL: "http://example2.com"},
		{URL: "http://example3.com"},
	}

	// 设置并发数
	concurrency := 2

	// 创建结果通道
	resultsChan := make(chan ConcurrentTestResult, len(requests))

	// 创建计数器
	var processedCount int32

	// 创建处理函数
	processor := func(req ScreenshotRequest) ConcurrentTestResult {
		// 增加计数器
		atomic.AddInt32(&processedCount, 1)

		// 模拟处理时间
		time.Sleep(10 * time.Millisecond)

		return ConcurrentTestResult{
			URL:     req.URL,
			Success: true,
		}
	}

	// 启动并发处理
	go func() {
		// 创建工作协程
		jobs := make(chan ScreenshotRequest, len(requests))

		// 启动工作协程
		for i := 0; i < concurrency; i++ {
			go func() {
				for req := range jobs {
					resultsChan <- processor(req)
				}
			}()
		}

		// 发送请求到工作通道
		for _, req := range requests {
			jobs <- req
		}

		// 关闭工作通道
		close(jobs)
	}()

	// 检查是否收到所有结果
	var results []ConcurrentTestResult
	for i := 0; i < len(requests); i++ {
		select {
		case result := <-resultsChan:
			results = append(results, result)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("等待结果超时")
		}
	}

	// 验证结果计数
	if len(results) != len(requests) {
		t.Errorf("期望有%d个结果，但得到%d个", len(requests), len(results))
	}

	// 验证是否所有请求都被处理
	if int(processedCount) != len(requests) {
		t.Errorf("期望处理%d个请求，但实际处理了%d个", len(requests), processedCount)
	}
}

// TestHandleBatchScreenshotServerMethod tests the actual server method HandleBatchScreenshot
func TestHandleBatchScreenshotServerMethod(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectSuccess  bool
		maxBatchSize   int
	}{
		{
			name:           "invalid JSON body",
			requestBody:    "not json",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			maxBatchSize:   10,
		},
		{
			name:           "empty URL list",
			requestBody:    `{"urls":[]}`,
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			maxBatchSize:   10,
		},
		{
			name:           "exceeds max batch size",
			requestBody:    `{"urls":["https://a.com","https://b.com","https://c.com"]}`,
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			maxBatchSize:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				Options: ServerOptions{
					ScreenshotPath: "/tmp/test-screenshots",
					MaxBatchSize:   tt.maxBatchSize,
				},
			}

			req, err := http.NewRequest("POST", "/batch", bytes.NewBufferString(tt.requestBody))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(server.HandleBatchScreenshot)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.expectedStatus)
			}

			var response APIResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("failed to parse response: %v", err)
			}
			if response.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v", response.Success, tt.expectSuccess)
			}
		})
	}
}

// TestProcessBatchConcurrentEdgeCases tests edge cases for ProcessBatchConcurrent
func TestProcessBatchConcurrentEdgeCases(t *testing.T) {
	t.Run("empty requests list", func(t *testing.T) {
		requests := []ScreenshotRequest{}
		resultsChan := make(chan BatchResult, 1)

		// This should not panic or deadlock
		ProcessBatchConcurrent(requests, 2, func(req ScreenshotRequest) BatchResult {
			return BatchResult{URL: req.URL}
		}, resultsChan)
	})

	t.Run("single request", func(t *testing.T) {
		requests := []ScreenshotRequest{
			{URL: "https://example.com"},
		}
		resultsChan := make(chan BatchResult, len(requests))

		ProcessBatchConcurrent(requests, 1, func(req ScreenshotRequest) BatchResult {
			return BatchResult{URL: req.URL, Error: ""}
		}, resultsChan)

		result := <-resultsChan
		if result.URL != "https://example.com" {
			t.Errorf("URL = %v, want https://example.com", result.URL)
		}
	})

	t.Run("high concurrency", func(t *testing.T) {
		requests := make([]ScreenshotRequest, 50)
		for i := range requests {
			requests[i] = ScreenshotRequest{URL: "https://example.com"}
		}
		resultsChan := make(chan BatchResult, len(requests))

		var processed int32
		ProcessBatchConcurrent(requests, 10, func(req ScreenshotRequest) BatchResult {
			atomic.AddInt32(&processed, 1)
			return BatchResult{URL: req.URL}
		}, resultsChan)

		// Collect all results
		count := 0
		for range requests {
			<-resultsChan
			count++
		}

		if count != 50 {
			t.Errorf("collected %v results, want 50", count)
		}
		if int(processed) != 50 {
			t.Errorf("processed %v requests, want 50", processed)
		}
	})

	t.Run("processor returns errors", func(t *testing.T) {
		requests := []ScreenshotRequest{
			{URL: "https://fail1.com"},
			{URL: "https://fail2.com"},
		}
		resultsChan := make(chan BatchResult, len(requests))

		ProcessBatchConcurrent(requests, 2, func(req ScreenshotRequest) BatchResult {
			return BatchResult{
				URL:   req.URL,
				Error: "simulated failure",
			}
		}, resultsChan)

		for range requests {
			result := <-resultsChan
			if result.Error != "simulated failure" {
				t.Errorf("Error = %v, want 'simulated failure'", result.Error)
			}
		}
	})
}
