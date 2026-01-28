package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
