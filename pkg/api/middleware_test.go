package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		requestPath    string
		requestMethod  string
		requestAPIKey  string
		expectedStatus int
	}{
		{
			name:           "无需认证的路径-根路径",
			apiKey:         "test-key",
			requestPath:    "/",
			requestMethod:  "GET",
			requestAPIKey:  "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无需认证的路径-健康检查",
			apiKey:         "test-key",
			requestPath:    "/health",
			requestMethod:  "GET",
			requestAPIKey:  "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无需认证的路径-状态信息",
			apiKey:         "test-key",
			requestPath:    "/stats",
			requestMethod:  "GET",
			requestAPIKey:  "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无需认证的路径-截图文件",
			apiKey:         "test-key",
			requestPath:    "/screenshots/test.png",
			requestMethod:  "GET",
			requestAPIKey:  "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "无需认证的OPTIONS请求",
			apiKey:         "test-key",
			requestPath:    "/screenshot",
			requestMethod:  "OPTIONS",
			requestAPIKey:  "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "未设置API密钥时跳过认证",
			apiKey:         "",
			requestPath:    "/screenshot",
			requestMethod:  "POST",
			requestAPIKey:  "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "需要认证的路径-有效的Header认证",
			apiKey:         "test-key",
			requestPath:    "/screenshot",
			requestMethod:  "POST",
			requestAPIKey:  "test-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "需要认证的路径-无效的Header认证",
			apiKey:         "test-key",
			requestPath:    "/screenshot",
			requestMethod:  "POST",
			requestAPIKey:  "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试服务器
			server := &Server{
				Options: ServerOptions{
					APIKey: tt.apiKey,
				},
			}

			// 创建一个简单的处理函数，它总是返回OK
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// 创建认证中间件
			authMiddleware := server.CreateAuthMiddleware()(handler)

			// 创建测试请求
			req, err := http.NewRequest(tt.requestMethod, tt.requestPath, nil)
			if err != nil {
				t.Fatalf("无法创建请求: %v", err)
			}

			// 如果提供了API密钥，则添加到请求头
			if tt.requestAPIKey != "" {
				req.Header.Set("X-API-Key", tt.requestAPIKey)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()

			// 处理请求
			authMiddleware.ServeHTTP(rr, req)

			// 检查响应状态码
			if rr.Code != tt.expectedStatus {
				t.Errorf("状态码 = %v, 期望 %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestCreateAuthMiddlewareWithQueryParam(t *testing.T) {
	// 测试查询参数中的API密钥
	server := &Server{
		Options: ServerOptions{
			APIKey: "test-key",
		},
	}

	// 创建一个简单的处理函数，它总是返回OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 创建认证中间件
	authMiddleware := server.CreateAuthMiddleware()(handler)

	// 创建测试请求，使用查询参数中的API密钥
	req, err := http.NewRequest("GET", "/screenshot?api_key=test-key", nil)
	if err != nil {
		t.Fatalf("无法创建请求: %v", err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 处理请求
	authMiddleware.ServeHTTP(rr, req)

	// 检查响应状态码
	if rr.Code != http.StatusOK {
		t.Errorf("状态码 = %v, 期望 %v", rr.Code, http.StatusOK)
	}
}

// TestCreateAuthMiddlewareNoAPIKey 测试未设置API密钥的情况
func TestCreateAuthMiddlewareNoAPIKey(t *testing.T) {
	// 创建一个没有API密钥的服务器
	server := &Server{
		Options: ServerOptions{
			APIKey: "", // 未设置API密钥
		},
	}

	// 创建中间件
	middleware := server.CreateAuthMiddleware()

	// 创建测试请求 - 使用需要认证的路径
	req, err := http.NewRequest("GET", "/screenshot", nil)
	if err != nil {
		t.Fatalf("无法创建请求: %v", err)
	}

	// 创建响应记录器
	rr := httptest.NewRecorder()

	// 创建处理器链
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// 应用中间件
	handlerChain := middleware(testHandler)

	// 处理请求
	handlerChain.ServeHTTP(rr, req)

	// 检查状态码 - 由于未设置API密钥，应该允许通过
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("处理程序返回了错误的状态码：获取=%v，期望=%v", status, http.StatusOK)
	}

	// 检查响应体
	expected := "success"
	if rr.Body.String() != expected {
		t.Errorf("处理程序返回了意外的响应体：获取=%v，期望=%v", rr.Body.String(), expected)
	}
}

// TestCreateAuthMiddlewareSpecialPaths 测试特殊路径不需要认证
func TestCreateAuthMiddlewareSpecialPaths(t *testing.T) {
	// 创建需要API密钥的服务器
	server := &Server{
		Options: ServerOptions{
			APIKey: "test-api-key",
		},
	}

	// 创建中间件
	middleware := server.CreateAuthMiddleware()

	// 测试不同的特殊路径
	specialPaths := []string{
		"/",              // 根路径
		"/health",        // 健康检查
		"/stats",         // 统计信息
		"/screenshots/1", // 静态资源
	}

	for _, path := range specialPaths {
		t.Run(path, func(t *testing.T) {
			// 创建测试请求 - 无API密钥但使用特殊路径
			req, err := http.NewRequest("GET", path, nil)
			if err != nil {
				t.Fatalf("无法创建请求: %v", err)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()

			// 创建处理器链
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			// 应用中间件
			handlerChain := middleware(testHandler)

			// 处理请求
			handlerChain.ServeHTTP(rr, req)

			// 检查状态码 - 应该允许通过
			if status := rr.Code; status != http.StatusOK {
				t.Errorf("处理程序返回了错误的状态码：获取=%v，期望=%v", status, http.StatusOK)
			}

			// 检查响应体
			expected := "success"
			if rr.Body.String() != expected {
				t.Errorf("处理程序返回了意外的响应体：获取=%v，期望=%v", rr.Body.String(), expected)
			}
		})
	}
}

// TestCreateAuthMiddlewareMalformedKey 测试格式错误的API密钥
func TestCreateAuthMiddlewareMalformedKey(t *testing.T) {
	// 创建需要API密钥的服务器
	server := &Server{
		Options: ServerOptions{
			APIKey: "test-api-key",
		},
	}

	// 创建中间件
	middleware := server.CreateAuthMiddleware()

	// 创建测试用例
	tests := []struct {
		name     string
		apiKey   string
		expected int
	}{
		{
			name:     "空API密钥",
			apiKey:   "",
			expected: http.StatusUnauthorized,
		},
		{
			name:     "错误的API密钥",
			apiKey:   "wrong-key",
			expected: http.StatusUnauthorized,
		},
		{
			name:     "格式错误的API密钥",
			apiKey:   "test-api-key   ", // 尾部有空格
			expected: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试请求
			req, err := http.NewRequest("GET", "/screenshot", nil)
			if err != nil {
				t.Fatalf("无法创建请求: %v", err)
			}

			// 添加API密钥到请求头
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()

			// 创建处理器链
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			// 应用中间件
			handlerChain := middleware(testHandler)

			// 处理请求
			handlerChain.ServeHTTP(rr, req)

			// 检查状态码
			if status := rr.Code; status != tt.expected {
				t.Errorf("处理程序返回了错误的状态码：获取=%v，期望=%v", status, tt.expected)
			}
		})
	}
}
