package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		options ServerOptions
	}{
		{
			name: "默认选项",
			options: ServerOptions{
				Port:                  8080,
				Host:                  "localhost",
				APIKey:                "test-key",
				ScreenshotPath:        "/tmp/screenshots",
				MaxBatchSize:          100,
				MaxConcurrency:        5,
				MaxConcurrentRequests: 10,
				RequestQueueSize:      50,
			},
		},
		{
			name: "自定义选项",
			options: ServerOptions{
				Port:                  9090,
				Host:                  "0.0.0.0",
				APIKey:                "custom-key",
				ScreenshotPath:        "/var/screenshots",
				MaxBatchSize:          200,
				MaxConcurrency:        10,
				MaxConcurrentRequests: 20,
				RequestQueueSize:      100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.options)

			// 验证服务器是否正确初始化
			if server == nil {
				t.Error("NewServer() 返回 nil")
			}

			// 验证关键字段是否匹配，而不是整个结构体比较
			if server.Options.Port != tt.options.Port ||
				server.Options.Host != tt.options.Host ||
				server.Options.APIKey != tt.options.APIKey ||
				server.Options.ScreenshotPath != tt.options.ScreenshotPath ||
				server.Options.MaxBatchSize != tt.options.MaxBatchSize ||
				server.Options.MaxConcurrency != tt.options.MaxConcurrency ||
				server.Options.MaxConcurrentRequests != tt.options.MaxConcurrentRequests ||
				server.Options.RequestQueueSize != tt.options.RequestQueueSize {
				t.Errorf("NewServer() 配置不匹配，得到 %+v, 期望 %+v", server.Options, tt.options)
			}

			if server.Router == nil {
				t.Error("Server Router 未初始化")
			}
		})
	}
}

func TestSetupRoutes(t *testing.T) {
	// 创建服务器
	server := &Server{
		Options: ServerOptions{
			APIKey:         "test-key",
			ScreenshotPath: "/tmp/screenshots",
		},
		Router: mux.NewRouter(),
	}

	// 设置路由
	server.SetupRoutes()

	// 测试路由是否正确设置
	expectedRoutes := []struct {
		path   string
		method string
	}{
		{"/screenshot", "POST"},
		{"/batch", "POST"},
		{"/screenshots_list", "GET"},
		{"/get_screenshot/{filename}", "GET"},
		{"/", "GET"},
		{"/stats", "GET"},
		{"/health", "GET"},
	}

	for _, route := range expectedRoutes {
		req, _ := http.NewRequest(route.method, route.path, nil)
		match := &mux.RouteMatch{}

		if !server.Router.Match(req, match) {
			t.Errorf("路由 %s %s 未找到", route.method, route.path)
		}
	}
}

func TestHandleRoot(t *testing.T) {
	// 创建服务器
	server := &Server{
		Options: ServerOptions{
			ScreenshotPath: "/tmp/screenshots",
		},
		Router: mux.NewRouter(),
	}

	// 创建测试请求
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("无法创建请求: %v", err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用处理函数
	handler := http.HandlerFunc(server.HandleRoot)
	handler.ServeHTTP(rr, req)

	// 检查状态码
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("状态码错误: 得到 %v 期望 %v", status, http.StatusOK)
	}

	// 检查响应体是JSON格式
	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("无法解析JSON响应: %v", err)
	}

	// 检查API响应是否包含必要字段
	if !response.Success {
		t.Errorf("响应中的Success字段应为true")
	}

	if response.Message == "" {
		t.Errorf("响应中的Message字段不应为空")
	}

	// 检查Data字段是否包含预期内容
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Errorf("响应中的Data字段类型错误")
		return
	}

	// 检查关键字段是否存在
	requiredFields := []string{"version", "documentation", "endpoints", "screenshot_dir"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			t.Errorf("响应中的Data字段缺少 %s", field)
		}
	}
}

func TestHandleHealth(t *testing.T) {
	// 创建测试请求
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("无法创建请求: %v", err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用处理函数
	handler := http.HandlerFunc(HandleHealth)
	handler.ServeHTTP(rr, req)

	// 检查状态码
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("状态码错误: 得到 %v 期望 %v", status, http.StatusOK)
	}

	// 检查响应体是JSON格式
	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("无法解析JSON响应: %v", err)
	}

	// 检查响应是否包含必要字段
	if !response.Success {
		t.Errorf("响应中的Success字段应为true")
	}

	if response.Message == "" {
		t.Errorf("响应中的Message字段不应为空")
	}
}

func TestHandleStats(t *testing.T) {
	// 初始化并发限制器，确保有统计数据
	InitConcurrencyLimiter(10, 100)

	// 创建测试请求
	req, err := http.NewRequest("GET", "/stats", nil)
	if err != nil {
		t.Fatalf("无法创建请求: %v", err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 调用处理函数
	handler := http.HandlerFunc(HandleStats)
	handler.ServeHTTP(rr, req)

	// 检查状态码
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("状态码错误: 得到 %v 期望 %v", status, http.StatusOK)
	}

	// 检查响应体是JSON格式
	var response APIResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("无法解析JSON响应: %v", err)
	}

	// 检查API响应是否包含必要字段
	if !response.Success {
		t.Errorf("响应中的Success字段应为true")
	}

	// 检查Data字段是否包含预期内容
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Errorf("响应中的Data字段类型错误")
		return
	}

	// 检查关键字段是否存在
	requiredFields := []string{"active_requests", "waiting_requests", "max_concurrent", "queue_size", "uptime", "started_at"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			t.Errorf("响应中的Data字段缺少 %s", field)
		}
	}
}

func TestInitConcurrencyLimiter(t *testing.T) {
	// 测试初始化并发限制器
	tests := []struct {
		name          string
		maxConcurrent int
		queueSize     int
	}{
		{
			name:          "正常参数",
			maxConcurrent: 20,
			queueSize:     200,
		},
		{
			name:          "负的最大并发数",
			maxConcurrent: -1,
			queueSize:     100,
		},
		{
			name:          "零最大并发数",
			maxConcurrent: 0,
			queueSize:     100,
		},
		{
			name:          "负的等待队列",
			maxConcurrent: 10,
			queueSize:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置状态以便重新初始化
			limiterInitialized = false

			// 初始化并发限制器
			InitConcurrencyLimiter(tt.maxConcurrent, tt.queueSize)

			// 检查初始化状态
			if !limiterInitialized {
				t.Error("并发限制器未被初始化")
			}

			// 检查参数设置
			if tt.maxConcurrent <= 0 && maxConcurrent <= 0 {
				t.Errorf("应该使用默认并发数，但得到 %v", maxConcurrent)
			}

			if tt.queueSize <= 0 && maxQueueSize <= 0 {
				t.Errorf("应该使用默认队列大小，但得到 %v", maxQueueSize)
			}

			// 检查通道创建
			if concurrencySemaCh == nil {
				t.Error("信号量通道未被创建")
			}

			// 重新初始化应该无效
			oldMaxConcurrent := maxConcurrent
			InitConcurrencyLimiter(99, 999)
			if maxConcurrent != oldMaxConcurrent {
				t.Error("重复初始化不应改变参数")
			}
		})
	}
}

func TestAcquireAndReleaseConcurrencyPermit(t *testing.T) {
	// 重置状态
	limiterInitialized = false

	// 初始化并发限制器，最大并发数为2
	InitConcurrencyLimiter(2, 5)

	// 创建上下文
	ctx := context.Background()

	// 获取第一个许可
	err1 := AcquireConcurrencyPermit(ctx)
	if err1 != nil {
		t.Errorf("获取第一个许可失败: %v", err1)
	}

	// 检查活跃请求数
	active, _, _, _, _ := GetConcurrencyStats()
	if active != 1 {
		t.Errorf("活跃请求数应为1，但得到: %v", active)
	}

	// 获取第二个许可
	err2 := AcquireConcurrencyPermit(ctx)
	if err2 != nil {
		t.Errorf("获取第二个许可失败: %v", err2)
	}

	// 检查活跃请求数
	active, _, _, _, _ = GetConcurrencyStats()
	if active != 2 {
		t.Errorf("活跃请求数应为2，但得到: %v", active)
	}

	// 尝试获取第三个许可，应该会被阻塞，使用带超时的上下文测试
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// 启动一个goroutine尝试获取许可
	done := make(chan struct{})
	var err3 error
	go func() {
		err3 = AcquireConcurrencyPermit(ctxWithTimeout)
		close(done)
	}()

	// 等待超时或许可获取
	select {
	case <-done:
		// 检查错误类型
		if err3 != context.DeadlineExceeded {
			t.Errorf("预期超时错误，但得到: %v", err3)
		}
	case <-time.After(200 * time.Millisecond):
		t.Errorf("测试超时")
	}

	// 释放一个许可
	ReleaseConcurrencyPermit()

	// 检查活跃请求数
	active, _, _, _, _ = GetConcurrencyStats()
	if active != 1 {
		t.Errorf("活跃请求数应为1，但得到: %v", active)
	}

	// 最后清理
	ReleaseConcurrencyPermit()

	// 检查活跃请求数
	active, _, _, _, _ = GetConcurrencyStats()
	if active != 0 {
		t.Errorf("活跃请求数应为0，但得到: %v", active)
	}

	// 未初始化时的行为
	limiterInitialized = false
	err := AcquireConcurrencyPermit(ctx)
	if err != nil {
		t.Errorf("未初始化时获取许可应返回nil，但得到: %v", err)
	}

	// 未初始化时释放许可应该不会崩溃
	ReleaseConcurrencyPermit()
}

func TestCreateConcurrencyLimitMiddleware(t *testing.T) {
	// 重置状态
	limiterInitialized = false

	// 初始化并发限制器
	InitConcurrencyLimiter(2, 5)

	// 创建中间件
	middleware := CreateConcurrencyLimitMiddleware()

	// 创建一个测试处理函数
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 应用中间件
	handler := middleware(testHandler)

	// 测试不同路径的请求
	tests := []struct {
		name         string
		path         string
		method       string
		expectedCode int
		shouldLimit  bool
	}{
		{
			name:         "健康检查路径",
			path:         "/health",
			method:       "GET",
			expectedCode: http.StatusOK,
			shouldLimit:  false,
		},
		{
			name:         "状态路径",
			path:         "/stats",
			method:       "GET",
			expectedCode: http.StatusOK,
			shouldLimit:  false,
		},
		{
			name:         "根路径",
			path:         "/",
			method:       "GET",
			expectedCode: http.StatusOK,
			shouldLimit:  false,
		},
		{
			name:         "静态资源",
			path:         "/screenshots/image.png",
			method:       "GET",
			expectedCode: http.StatusOK,
			shouldLimit:  false,
		},
		{
			name:         "OPTIONS请求",
			path:         "/screenshot",
			method:       "OPTIONS",
			expectedCode: http.StatusOK,
			shouldLimit:  false,
		},
		{
			name:         "API路径",
			path:         "/screenshot",
			method:       "POST",
			expectedCode: http.StatusOK,
			shouldLimit:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建请求
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatalf("无法创建请求: %v", err)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()

			// 处理请求
			handler.ServeHTTP(rr, req)

			// 检查状态码
			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("状态码错误: 得到 %v 期望 %v", status, tt.expectedCode)
			}
		})
	}

	// 测试并发限制
	if true {
		// 已经有2个许可被获取
		AcquireConcurrencyPermit(context.Background())
		AcquireConcurrencyPermit(context.Background())

		// 创建一个API请求，应该因并发限制而超时
		req, _ := http.NewRequest("POST", "/screenshot", nil)

		// 创建响应记录器
		rr := httptest.NewRecorder()

		// 处理请求，应该被拒绝
		handler.ServeHTTP(rr, req)

		// 检查状态码，应该是服务不可用或请求过多
		if status := rr.Code; status != http.StatusServiceUnavailable && status != http.StatusTooManyRequests {
			t.Errorf("状态码错误: 得到 %v, 期望 %v 或 %v",
				status, http.StatusServiceUnavailable, http.StatusTooManyRequests)
		}

		// 释放许可
		ReleaseConcurrencyPermit()
		ReleaseConcurrencyPermit()
	}
}
