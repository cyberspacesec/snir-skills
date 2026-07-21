package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

func TestUrlWithProtocol(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		https    bool
		http     bool
		expected string
	}{
		{
			name:     "空URL",
			url:      "",
			https:    true,
			http:     false,
			expected: "",
		},
		{
			name:     "已有HTTPS前缀",
			url:      "https://example.com",
			https:    true,
			http:     false,
			expected: "https://example.com",
		},
		{
			name:     "已有HTTP前缀",
			url:      "http://example.com",
			https:    true,
			http:     false,
			expected: "http://example.com",
		},
		{
			name:     "无前缀，HTTPS优先",
			url:      "example.com",
			https:    true,
			http:     false,
			expected: "https://example.com",
		},
		{
			name:     "无前缀，HTTP优先",
			url:      "example.com",
			https:    false,
			http:     true,
			expected: "http://example.com",
		},
		{
			name:     "无前缀，HTTPS和HTTP都启用，优先HTTPS",
			url:      "example.com",
			https:    true,
			http:     true,
			expected: "https://example.com",
		},
		{
			name:     "无前缀，HTTPS和HTTP都不启用，默认HTTPS",
			url:      "example.com",
			https:    false,
			http:     false,
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UrlWithProtocol(tt.url, tt.https, tt.http)
			if result != tt.expected {
				t.Errorf("UrlWithProtocol() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUrlHasProtocol(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "空URL",
			url:      "",
			expected: false,
		},
		{
			name:     "有HTTPS前缀",
			url:      "https://example.com",
			expected: true,
		},
		{
			name:     "有HTTP前缀",
			url:      "http://example.com",
			expected: true,
		},
		{
			name:     "无前缀",
			url:      "example.com",
			expected: false,
		},
		{
			name:     "只有部分前缀",
			url:      "htt://example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UrlHasProtocol(tt.url)
			if result != tt.expected {
				t.Errorf("UrlHasProtocol() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetImageContentType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "PNG图片",
			filename: "image.png",
			expected: "image/png",
		},
		{
			name:     "JPG图片",
			filename: "image.jpg",
			expected: "image/jpeg",
		},
		{
			name:     "JPEG图片",
			filename: "image.jpeg",
			expected: "image/jpeg",
		},
		{
			name:     "GIF图片",
			filename: "image.gif",
			expected: "image/gif",
		},
		{
			name:     "未知类型",
			filename: "file.txt",
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetImageContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("GetImageContentType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "PNG图片",
			filename: "image.png",
			expected: true,
		},
		{
			name:     "JPG图片",
			filename: "image.jpg",
			expected: true,
		},
		{
			name:     "JPEG图片",
			filename: "image.jpeg",
			expected: true,
		},
		{
			name:     "GIF图片",
			filename: "image.gif",
			expected: true,
		},
		{
			name:     "文本文件",
			filename: "file.txt",
			expected: false,
		},
		{
			name:     "无扩展名",
			filename: "noextension",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsImageFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsImageFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSendJSONResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   APIResponse
		checkFunc  func(*httptest.ResponseRecorder) bool
	}{
		{
			name:       "成功响应",
			statusCode: http.StatusOK,
			response: APIResponse{
				Success: true,
				Message: "操作成功",
				Data:    map[string]string{"key": "value"},
			},
			checkFunc: func(rr *httptest.ResponseRecorder) bool {
				return rr.Code == http.StatusOK &&
					rr.Header().Get("Content-Type") == "application/json" &&
					rr.Header().Get("Access-Control-Allow-Origin") == "*"
			},
		},
		{
			name:       "错误响应",
			statusCode: http.StatusBadRequest,
			response: APIResponse{
				Success: false,
				Error:   "参数错误",
			},
			checkFunc: func(rr *httptest.ResponseRecorder) bool {
				return rr.Code == http.StatusBadRequest &&
					rr.Header().Get("Content-Type") == "application/json" &&
					rr.Header().Get("Access-Control-Allow-Origin") == "*"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			SendJSONResponse(rr, tt.statusCode, tt.response)

			if !tt.checkFunc(rr) {
				t.Errorf("SendJSONResponse() failed validation checks")
			}
		})
	}
}

// TestSendJSONResponseErrorPath tests SendJSONResponse with a broken writer
func TestSendJSONResponseErrorPath(t *testing.T) {
	// Create a response writer that fails on write
	bw := &brokenResponseWriter{
		header: make(http.Header),
	}
	// This should not panic; the function handles write errors internally
	SendJSONResponse(bw, http.StatusOK, APIResponse{
		Success: true,
		Message: "test",
	})
}

// brokenResponseWriter is a ResponseWriter that fails on Write
type brokenResponseWriter struct {
	header http.Header
}

func (bw *brokenResponseWriter) Header() http.Header {
	return bw.header
}

func (bw *brokenResponseWriter) Write(data []byte) (int, error) {
	return 0, fmt.Errorf("simulated write error")
}

func (bw *brokenResponseWriter) WriteHeader(statusCode int) {
	// no-op
}

// TestMemoryWriter tests the MemoryWriter Write and Close methods
func TestMemoryWriter(t *testing.T) {
	t.Run("Write adds result", func(t *testing.T) {
		mw := &MemoryWriter{}
		result := &models.Result{
			URL:        "https://example.com",
			Title:      "Test Title",
			Screenshot: "test.png",
		}

		err := mw.Write(result)
		if err != nil {
			t.Errorf("Write() error = %v, want nil", err)
		}

		if len(mw.Results) != 1 {
			t.Errorf("Results length = %v, want 1", len(mw.Results))
		}

		if mw.Results[0].URL != "https://example.com" {
			t.Errorf("Result URL = %v, want https://example.com", mw.Results[0].URL)
		}
	})

	t.Run("Write multiple results", func(t *testing.T) {
		mw := &MemoryWriter{}
		for i := 0; i < 5; i++ {
			err := mw.Write(&models.Result{URL: "https://example.com"})
			if err != nil {
				t.Errorf("Write() error = %v", err)
			}
		}

		if len(mw.Results) != 5 {
			t.Errorf("Results length = %v, want 5", len(mw.Results))
		}
	})

	t.Run("Write concurrent", func(t *testing.T) {
		mw := &MemoryWriter{}
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mw.Write(&models.Result{URL: "https://example.com"})
			}()
		}
		wg.Wait()

		if len(mw.Results) != 100 {
			t.Errorf("Results length = %v, want 100", len(mw.Results))
		}
	})

	t.Run("Close returns nil", func(t *testing.T) {
		mw := &MemoryWriter{}
		err := mw.Close()
		if err != nil {
			t.Errorf("Close() error = %v, want nil", err)
		}
	})
}

// TestSendJSONResponse_MarshalError 覆盖 SendJSONResponse 中 json.Marshal 失败分支。
func TestSendJSONResponse_MarshalError(t *testing.T) {
	rr := httptest.NewRecorder()
	// channel 无法被 json.Marshal，触发序列化失败分支
	SendJSONResponse(rr, http.StatusOK, APIResponse{
		Success: true,
		Data:    make(chan int),
	})
	body := rr.Body.String()
	if !strings.Contains(body, "内部服务器错误") {
		t.Fatalf("响应体应包含内部服务器错误, got %q", body)
	}
}

// TestSendJSONResponse_StatusCode 覆盖不同状态码写入。
func TestSendJSONResponse_StatusCode(t *testing.T) {
	rr := httptest.NewRecorder()
	SendJSONResponse(rr, http.StatusTeapot, APIResponse{Success: false, Error: "teapot"})
	if rr.Code != http.StatusTeapot {
		t.Fatalf("状态码 = %d, want %d", rr.Code, http.StatusTeapot)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if ao := rr.Header().Get("Access-Control-Allow-Origin"); ao != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q", ao)
	}
}
