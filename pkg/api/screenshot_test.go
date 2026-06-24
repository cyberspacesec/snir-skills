package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/gorilla/mux"
)

// 模拟错误
var ErrMockScreenshotFailed = fmt.Errorf("模拟截图失败")

// 为测试创建一个MockScreenshotHandler
func MockScreenshotHandler(success bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ScreenshotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "无效的请求格式",
				Error:   err.Error(),
			})
			return
		}

		// 验证URL是否为空
		if req.URL == "" {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Message: "URL不能为空",
			})
			return
		}

		// 根据success参数决定是否返回成功结果
		if !success {
			SendJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Message: "截图处理失败",
				Error:   ErrMockScreenshotFailed.Error(),
			})
			return
		}

		// 返回模拟的成功结果
		result := &models.Result{
			URL:        req.URL,
			Title:      "Mock Page Title",
			Screenshot: "mock_screenshot.png",
		}

		SendJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "截图处理成功",
			Data:    result,
		})
	}
}

func TestHandleScreenshot(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectSuccess  bool
		shouldSucceed  bool
	}{
		{
			name: "有效请求",
			requestBody: ScreenshotRequest{
				URL: "https://example.com",
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			shouldSucceed:  true,
		},
		{
			name: "空URL",
			requestBody: ScreenshotRequest{
				URL: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			shouldSucceed:  true,
		},
		{
			name:           "无效的请求格式",
			requestBody:    "这不是一个有效的JSON",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			shouldSucceed:  true,
		},
		{
			name: "处理失败",
			requestBody: ScreenshotRequest{
				URL: "https://example.com",
			},
			expectedStatus: http.StatusInternalServerError,
			expectSuccess:  false,
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的处理函数
			handler := MockScreenshotHandler(tt.shouldSucceed)

			// 创建请求
			var reqBody []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("无法序列化请求: %v", err)
				}
			}

			req, err := http.NewRequest("POST", "/screenshot", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("无法创建请求: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// 创建响应记录器
			rr := httptest.NewRecorder()

			// 处理请求
			handler.ServeHTTP(rr, req)

			// 检查状态码
			if rr.Code != tt.expectedStatus {
				t.Errorf("状态码 = %v, 期望 %v", rr.Code, tt.expectedStatus)
			}

			// 解析响应
			var response APIResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("无法解析响应: %v", err)
			}

			// 检查成功状态
			if response.Success != tt.expectSuccess {
				t.Errorf("Success = %v, 期望 %v", response.Success, tt.expectSuccess)
			}
		})
	}
}

// 创建一个模拟HTTP中间件用于安全检查
func safeFilenameMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		filename := vars["filename"]

		// 安全检查
		if strings.Contains(filename, "..") {
			SendJSONResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "无效的文件名",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// 创建一个直接处理文件名的模拟处理函数
func createGetScreenshotHandler(screenshotPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取文件名参数
		vars := mux.Vars(r)
		filename := vars["filename"]

		// 构建文件路径
		filePath := filepath.Join(screenshotPath, filename)

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			SendJSONResponse(w, http.StatusNotFound, APIResponse{
				Success: false,
				Error:   "文件不存在",
			})
			return
		}

		// 读取文件
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
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
}

func TestHandleGetScreenshot(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "screenshot_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试图片
	testImagePath := filepath.Join(tempDir, "test.png")
	testImageContent := []byte("模拟PNG图片内容")
	if err := ioutil.WriteFile(testImagePath, testImageContent, 0644); err != nil {
		t.Fatalf("无法创建测试图片: %v", err)
	}

	// 创建测试服务器和路由器
	router := mux.NewRouter()

	// 添加安全中间件
	router.Use(safeFilenameMiddleware)

	// 注册模拟的处理函数
	router.HandleFunc("/get_screenshot/{filename}", createGetScreenshotHandler(tempDir))

	tests := []struct {
		name           string
		filename       string
		expectedStatus int
		expectedType   string
		expectedBody   []byte
	}{
		{
			name:           "存在的图片",
			filename:       "test.png",
			expectedStatus: http.StatusOK,
			expectedType:   "image/png",
			expectedBody:   testImageContent,
		},
		{
			name:           "不存在的图片",
			filename:       "nonexistent.png",
			expectedStatus: http.StatusNotFound,
			expectedType:   "application/json",
			expectedBody:   nil,
		},
		{
			// 路径遍历攻击的情况 - Gorilla Mux会返回301而不是400
			name:           "无效文件名",
			filename:       "../../../etc/passwd",
			expectedStatus: http.StatusMovedPermanently, // 301状态码表示重定向
			expectedType:   "",                          // 重定向响应没有内容类型
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建请求
			requestURL := "/get_screenshot/" + tt.filename
			req, err := http.NewRequest("GET", requestURL, nil)
			if err != nil {
				t.Fatalf("无法创建请求: %v", err)
			}

			// 创建响应记录器
			rr := httptest.NewRecorder()

			// 处理请求
			router.ServeHTTP(rr, req)

			// 检查状态码
			if rr.Code != tt.expectedStatus {
				t.Errorf("状态码 = %v, 期望 %v", rr.Code, tt.expectedStatus)
			}

			// 对于成功响应，检查Content-Type和响应体内容
			if tt.expectedStatus == http.StatusOK {
				// 检查Content-Type
				if contentType := rr.Header().Get("Content-Type"); contentType != tt.expectedType {
					t.Errorf("Content-Type = %v, 期望 %v", contentType, tt.expectedType)
				}

				// 检查响应体内容
				if !bytes.Equal(rr.Body.Bytes(), tt.expectedBody) {
					t.Errorf("响应内容不匹配")
				}
			}

			// 对于非成功且非重定向响应，检查Content-Type
			if tt.expectedStatus != http.StatusOK && tt.expectedStatus != http.StatusMovedPermanently &&
				rr.Header().Get("Content-Type") != tt.expectedType {
				t.Errorf("Content-Type = %v, 期望 %v", rr.Header().Get("Content-Type"), tt.expectedType)
			}
		})
	}
}

// TestHandleScreenshotServerMethod tests the actual server method with validation paths
// that don't require a Chrome instance (invalid JSON, empty URL, invalid URL format)
func TestHandleScreenshotServerMethod(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name:           "invalid JSON body",
			requestBody:    "not json",
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "empty URL",
			requestBody:    `{"url":""}`,
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name:           "invalid URL format with newlines",
			requestBody:    `{"url":"http://\ninvalid"}`,
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				Options: ServerOptions{
					ScreenshotPath: "/tmp/test-screenshots",
				},
			}

			req, err := http.NewRequest("POST", "/screenshot", bytes.NewBufferString(tt.requestBody))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(server.HandleScreenshot)
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

// TestHandleGetScreenshotServerMethod tests the actual server method HandleGetScreenshot
func TestHandleGetScreenshotServerMethod(t *testing.T) {
	// Create a temp directory with a test image
	tempDir, err := ioutil.TempDir("", "get_screenshot_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test PNG file
	testFilename := "test_screenshot.png"
	testFilePath := filepath.Join(tempDir, testFilename)
	testContent := []byte("fake png content")
	if err := ioutil.WriteFile(testFilePath, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a test JPG file
	testJpgFilename := "test_photo.jpg"
	testJpgPath := filepath.Join(tempDir, testJpgFilename)
	if err := ioutil.WriteFile(testJpgPath, []byte("fake jpg content"), 0644); err != nil {
		t.Fatalf("failed to create test jpg file: %v", err)
	}

	server := &Server{
		Options: ServerOptions{
			ScreenshotPath: tempDir,
		},
	}

	router := mux.NewRouter()
	router.HandleFunc("/get_screenshot/{filename}", server.HandleGetScreenshot)

	tests := []struct {
		name           string
		filename       string
		expectedStatus int
		expectedType   string
		checkBody      bool
		expectedBody   []byte
	}{
		{
			name:           "existing PNG file",
			filename:       testFilename,
			expectedStatus: http.StatusOK,
			expectedType:   "image/png",
			checkBody:      true,
			expectedBody:   testContent,
		},
		{
			name:           "existing JPG file",
			filename:       testJpgFilename,
			expectedStatus: http.StatusOK,
			expectedType:   "image/jpeg",
			checkBody:      true,
			expectedBody:   []byte("fake jpg content"),
		},
		{
			name:           "non-existent file",
			filename:       "does_not_exist.png",
			expectedStatus: http.StatusNotFound,
			expectedType:   "application/json",
		},
		{
			name:           "path traversal attempt",
			filename:       "..%2F..%2Fetc%2Fpasswd",
			expectedStatus: http.StatusMovedPermanently,
			expectedType:   "",
		},
		{
			name:           "double dot traversal",
			filename:       "../etc/passwd",
			expectedStatus: http.StatusMovedPermanently,
			expectedType:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestURL := "/get_screenshot/" + tt.filename
			req, err := http.NewRequest("GET", requestURL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.expectedStatus)
			}

			if ct := rr.Header().Get("Content-Type"); tt.expectedType != "" && ct != tt.expectedType {
				t.Errorf("Content-Type = %v, want %v", ct, tt.expectedType)
			}

			if tt.checkBody && !bytes.Equal(rr.Body.Bytes(), tt.expectedBody) {
				t.Errorf("response body mismatch")
			}
		})
	}
}

func TestHandleListScreenshots(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "screenshot_list_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试图片
	testFiles := []string{
		"example.com_20220101.png",
		"test.com_20220102.jpg",
		"notanimage.txt", // 这个不应该在结果中
		"subdirectory",   // 这个不应该在结果中
	}

	for _, filename := range testFiles {
		path := filepath.Join(tempDir, filename)
		if filename == "subdirectory" {
			if err := os.Mkdir(path, 0755); err != nil {
				t.Fatalf("无法创建测试目录: %v", err)
			}
		} else {
			if err := ioutil.WriteFile(path, []byte("test content"), 0644); err != nil {
				t.Fatalf("无法创建测试文件: %v", err)
			}
		}
	}

	// 创建测试服务器
	server := &Server{
		Options: ServerOptions{
			ScreenshotPath: tempDir,
		},
	}

	// 测试列表截图处理程序
	req, err := http.NewRequest("GET", "/screenshots_list", nil)
	if err != nil {
		t.Fatalf("无法创建请求: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.HandleListScreenshots)
	handler.ServeHTTP(rr, req)

	// 检查状态码
	if rr.Code != http.StatusOK {
		t.Errorf("状态码 = %v, 期望 %v", rr.Code, http.StatusOK)
	}

	// 解析响应
	var response APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("无法解析响应: %v", err)
	}

	// 检查成功状态
	if !response.Success {
		t.Errorf("期望成功响应")
	}

	// 转换数据为结构体切片
	responseData, _ := json.Marshal(response.Data)
	var screenshots []ScreenshotInfo
	if err := json.Unmarshal(responseData, &screenshots); err != nil {
		t.Errorf("无法解析截图列表: %v", err)
	}

	// 应该只有两个图片文件
	if len(screenshots) != 2 {
		t.Errorf("期望2个截图，但得到%d个", len(screenshots))
	}

	// 检查文件名是否正确
	expectedFilenames := map[string]bool{
		"example.com_20220101.png": false,
		"test.com_20220102.jpg":    false,
	}

	for _, info := range screenshots {
		if _, exists := expectedFilenames[info.Filename]; exists {
			expectedFilenames[info.Filename] = true
		} else {
			t.Errorf("意外的文件名: %s", info.Filename)
		}
	}

	// 验证所有预期的文件都被找到
	for filename, found := range expectedFilenames {
		if !found {
			t.Errorf("没有找到预期的文件: %s", filename)
		}
	}
}
