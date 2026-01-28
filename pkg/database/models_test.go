package database

import (
	"testing"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/models"
)

func TestScreenshotFromResult(t *testing.T) {
	// 创建一个Result对象
	now := time.Now()
	result := &models.Result{
		URL:            "https://example.com",
		Title:          "Example Domain",
		Filename:       "test_screenshot.png",
		FinalURL:       "https://example.com/",
		ResponseCode:   200,
		ResponseReason: "OK",
		Protocol:       "HTTP/2",
		ContentLength:  1256,
		HTML:           "<html><body>Example</body></html>",
		ProbedAt:       now,
		Failed:         false,
		FailedReason:   "",
	}

	// 转换为Screenshot对象
	screenshot := Screenshot{}
	screenshot.FromResult(result)

	// 验证转换是否正确
	if screenshot.URL != result.URL {
		t.Errorf("URL不匹配, 期望: %s, 实际: %s", result.URL, screenshot.URL)
	}
	if screenshot.Title != result.Title {
		t.Errorf("Title不匹配, 期望: %s, 实际: %s", result.Title, screenshot.Title)
	}
	if screenshot.Filename != result.Filename {
		t.Errorf("Filename不匹配, 期望: %s, 实际: %s", result.Filename, screenshot.Filename)
	}
	if screenshot.ResponseCode != result.ResponseCode {
		t.Errorf("ResponseCode不匹配, 期望: %d, 实际: %d", result.ResponseCode, screenshot.ResponseCode)
	}
	if screenshot.ResponseReason != result.ResponseReason {
		t.Errorf("ResponseReason不匹配, 期望: %s, 实际: %s", result.ResponseReason, screenshot.ResponseReason)
	}
	if screenshot.Protocol != result.Protocol {
		t.Errorf("Protocol不匹配, 期望: %s, 实际: %s", result.Protocol, screenshot.Protocol)
	}
	if screenshot.ContentLength != result.ContentLength {
		t.Errorf("ContentLength不匹配, 期望: %d, 实际: %d", result.ContentLength, screenshot.ContentLength)
	}
	if screenshot.HTML != result.HTML {
		t.Errorf("HTML不匹配, 期望: %s, 实际: %s", result.HTML, screenshot.HTML)
	}
	if !screenshot.ProbedAt.Equal(result.ProbedAt) {
		t.Errorf("ProbedAt不匹配, 期望: %v, 实际: %v", result.ProbedAt, screenshot.ProbedAt)
	}
	if screenshot.Failed != result.Failed {
		t.Errorf("Failed不匹配, 期望: %t, 实际: %t", result.Failed, screenshot.Failed)
	}
	if screenshot.FailedReason != result.FailedReason {
		t.Errorf("FailedReason不匹配, 期望: %s, 实际: %s", result.FailedReason, screenshot.FailedReason)
	}
}

func TestScreenshotToResult(t *testing.T) {
	// 创建一个Screenshot对象
	now := time.Now()
	screenshot := Screenshot{
		ID:             1,
		URL:            "https://example.com",
		Title:          "Example Domain",
		Filename:       "test_screenshot.png",
		FinalURL:       "https://example.com/",
		ResponseCode:   200,
		ResponseReason: "OK",
		Protocol:       "HTTP/2",
		ContentLength:  1256,
		HTML:           "<html><body>Example</body></html>",
		ProbedAt:       now,
		Failed:         false,
		FailedReason:   "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// 转换为Result对象
	result := screenshot.ToResult()

	// 验证转换是否正确
	if result.URL != screenshot.URL {
		t.Errorf("URL不匹配, 期望: %s, 实际: %s", screenshot.URL, result.URL)
	}
	if result.Title != screenshot.Title {
		t.Errorf("Title不匹配, 期望: %s, 实际: %s", screenshot.Title, result.Title)
	}
	if result.Filename != screenshot.Filename {
		t.Errorf("Filename不匹配, 期望: %s, 实际: %s", screenshot.Filename, result.Filename)
	}
	if result.ResponseCode != screenshot.ResponseCode {
		t.Errorf("ResponseCode不匹配, 期望: %d, 实际: %d", screenshot.ResponseCode, result.ResponseCode)
	}
	if result.ResponseReason != screenshot.ResponseReason {
		t.Errorf("ResponseReason不匹配, 期望: %s, 实际: %s", screenshot.ResponseReason, result.ResponseReason)
	}
	if result.Protocol != screenshot.Protocol {
		t.Errorf("Protocol不匹配, 期望: %s, 实际: %s", screenshot.Protocol, result.Protocol)
	}
	if result.ContentLength != screenshot.ContentLength {
		t.Errorf("ContentLength不匹配, 期望: %d, 实际: %d", screenshot.ContentLength, result.ContentLength)
	}
	if result.HTML != screenshot.HTML {
		t.Errorf("HTML不匹配, 期望: %s, 实际: %s", screenshot.HTML, result.HTML)
	}
	if !result.ProbedAt.Equal(screenshot.ProbedAt) {
		t.Errorf("ProbedAt不匹配, 期望: %v, 实际: %v", screenshot.ProbedAt, result.ProbedAt)
	}
	if result.Failed != screenshot.Failed {
		t.Errorf("Failed不匹配, 期望: %t, 实际: %t", screenshot.Failed, result.Failed)
	}
	if result.FailedReason != screenshot.FailedReason {
		t.Errorf("FailedReason不匹配, 期望: %s, 实际: %s", screenshot.FailedReason, result.FailedReason)
	}
}

func TestRoundTripConversion(t *testing.T) {
	// 创建一个原始的Result对象
	originalResult := &models.Result{
		URL:            "https://example.com",
		Title:          "Example Domain",
		Filename:       "test_screenshot.png",
		FinalURL:       "https://example.com/",
		ResponseCode:   200,
		ResponseReason: "OK",
		Protocol:       "HTTP/2",
		ContentLength:  1256,
		HTML:           "<html><body>Example</body></html>",
		ProbedAt:       time.Now(),
		Failed:         false,
		FailedReason:   "",
	}

	// 先转换为Screenshot
	screenshot := Screenshot{}
	screenshot.FromResult(originalResult)

	// 再转换回Result
	resultAfterRoundTrip := screenshot.ToResult()

	// 验证转换后的Result与原始Result是否相同
	if resultAfterRoundTrip.URL != originalResult.URL {
		t.Errorf("URL不匹配, 期望: %s, 实际: %s", originalResult.URL, resultAfterRoundTrip.URL)
	}
	if resultAfterRoundTrip.Title != originalResult.Title {
		t.Errorf("Title不匹配, 期望: %s, 实际: %s", originalResult.Title, resultAfterRoundTrip.Title)
	}
	if resultAfterRoundTrip.Filename != originalResult.Filename {
		t.Errorf("Filename不匹配, 期望: %s, 实际: %s", originalResult.Filename, resultAfterRoundTrip.Filename)
	}
	if resultAfterRoundTrip.ResponseCode != originalResult.ResponseCode {
		t.Errorf("ResponseCode不匹配, 期望: %d, 实际: %d", originalResult.ResponseCode, resultAfterRoundTrip.ResponseCode)
	}
	if resultAfterRoundTrip.Protocol != originalResult.Protocol {
		t.Errorf("Protocol不匹配, 期望: %s, 实际: %s", originalResult.Protocol, resultAfterRoundTrip.Protocol)
	}
}
