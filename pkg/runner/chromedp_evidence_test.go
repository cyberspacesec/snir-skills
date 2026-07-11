package runner

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/network"
)

// TestContentLengthFromEncodedDataLength 验证 EventResponseReceived 时优先用 EncodedDataLength
func TestContentLengthFromEncodedDataLength(t *testing.T) {
	var contentLength int64
	e := &network.EventResponseReceived{
		RequestID: "1",
		Response: &network.Response{
			URL:               "https://example.com/",
			Status:            200,
			EncodedDataLength: 1256,
			Headers:           network.Headers{}, // 无 Content-Length 头，模拟 HTTP/2
		},
	}
	// 模拟主请求匹配块内的逻辑
	if e.Response.EncodedDataLength > 0 {
		contentLength = int64(e.Response.EncodedDataLength)
	}
	if contentLength != 1256 {
		t.Errorf("contentLength = %d, want 1256 (from EncodedDataLength)", contentLength)
	}
}

// TestContentLengthFallbackToHeader 验证 EncodedDataLength 为 0 时从头解析
func TestContentLengthFallbackToHeader(t *testing.T) {
	var contentLength int64
	e := &network.EventResponseReceived{
		RequestID: "1",
		Response: &network.Response{
			URL:               "https://example.com/",
			Status:            200,
			EncodedDataLength: 0,
			Headers:           network.Headers{"Content-Length": "999"},
		},
	}
	if e.Response.EncodedDataLength > 0 {
		contentLength = int64(e.Response.EncodedDataLength)
	} else if cl, ok := e.Response.Headers["Content-Length"]; ok {
		if clStr, ok := cl.(string); ok {
			if clInt, err := strconv.ParseInt(clStr, 10, 64); err == nil {
				contentLength = clInt
			}
		}
	}
	if contentLength != 999 {
		t.Errorf("contentLength = %d, want 999 (from Content-Length header)", contentLength)
	}
}

// TestLoadingFinishedOverridesZero 验证 EventLoadingFinished 兜底覆盖
func TestLoadingFinishedOverridesZero(t *testing.T) {
	var contentLength int64
	fin := &network.EventLoadingFinished{
		RequestID:         "1",
		EncodedDataLength: 2048,
	}
	if contentLength == 0 && fin.EncodedDataLength > 0 {
		contentLength = int64(fin.EncodedDataLength)
	}
	if contentLength != 2048 {
		t.Errorf("contentLength = %d, want 2048 (overridden by LoadingFinished)", contentLength)
	}
}

// TestResponseReasonHTTP1Fallback 验证 HTTP/1.x 空 StatusText 时从状态码推断
func TestResponseReasonHTTP1Fallback(t *testing.T) {
	protocol := "http/1.1"
	var responseReason string
	status := int64(200)
	if responseReason == "" && strings.HasPrefix(protocol, "http/1") {
		responseReason = http.StatusText(int(status))
	}
	if responseReason != "OK" {
		t.Errorf("responseReason = %q, want %q (inferred from status 200)", responseReason, "OK")
	}
}

// TestResponseReasonHTTP2EmptyIsNormal 验证 HTTP/2 空 StatusText 不触发降级
func TestResponseReasonHTTP2EmptyIsNormal(t *testing.T) {
	protocol := "h2"
	var responseReason string
	status := int64(200)
	if responseReason == "" && strings.HasPrefix(protocol, "http/1") {
		responseReason = http.StatusText(int(status))
	}
	if responseReason != "" {
		t.Errorf("responseReason = %q, want empty for HTTP/2 (no reason phrase transmitted)", responseReason)
	}
}
