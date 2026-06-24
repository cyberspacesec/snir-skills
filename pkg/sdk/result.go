package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// EvidenceSummary summarizes evidence captured with a screenshot result.
type EvidenceSummary struct {
	HasScreenshot      bool `json:"has_screenshot"`
	HasScreenshotBytes bool `json:"has_screenshot_bytes"`
	HasHTML            bool `json:"has_html"`
	HeaderCount        int  `json:"header_count"`
	CookieCount        int  `json:"cookie_count"`
	ConsoleCount       int  `json:"console_count"`
	ConsoleErrorCount  int  `json:"console_error_count"`
	NetworkCount       int  `json:"network_count"`
	NetworkErrorCount  int  `json:"network_error_count"`
	TechnologyCount    int  `json:"technology_count"`
	HasTLS             bool `json:"has_tls"`
}

// ResultWrapper 包装 models.Result 提供便捷访问方法
type ResultWrapper struct {
	*models.Result
}

// WrapResult 包装 Result
func WrapResult(r *models.Result) *ResultWrapper {
	if r == nil {
		return nil
	}
	return &ResultWrapper{Result: r}
}

// IsSuccess 截图是否成功
func (r *ResultWrapper) IsSuccess() bool {
	return r != nil && !r.Failed
}

// IsFailed 截图是否失败
func (r *ResultWrapper) IsFailed() bool {
	return r == nil || r.Failed
}

// HasScreenshot 是否有截图文件
func (r *ResultWrapper) HasScreenshot() bool {
	return r != nil && r.Screenshot != ""
}

// HasHTML 是否包含 HTML 源码
func (r *ResultWrapper) HasHTML() bool {
	return r != nil && r.HTML != ""
}

// HasEvidence reports whether the result contains any captured evidence.
func (r *ResultWrapper) HasEvidence() bool {
	summary := r.EvidenceSummary()
	return summary.HasScreenshot ||
		summary.HasScreenshotBytes ||
		summary.HasHTML ||
		summary.HeaderCount > 0 ||
		summary.CookieCount > 0 ||
		summary.ConsoleCount > 0 ||
		summary.NetworkCount > 0 ||
		summary.TechnologyCount > 0 ||
		summary.HasTLS
}

// EvidenceSummary returns counts and availability flags for captured evidence.
func (r *ResultWrapper) EvidenceSummary() EvidenceSummary {
	if r == nil || r.Result == nil {
		return EvidenceSummary{}
	}

	summary := EvidenceSummary{
		HasScreenshot:      r.Screenshot != "",
		HasScreenshotBytes: len(r.ScreenshotBytes) > 0,
		HasHTML:            r.HTML != "",
		HeaderCount:        len(r.Headers),
		CookieCount:        len(r.Cookies),
		ConsoleCount:       len(r.Console),
		NetworkCount:       len(r.Network),
		TechnologyCount:    len(r.Technologies),
		HasTLS:             hasTLSInfo(r.TLS),
	}

	for _, c := range r.Console {
		if c.Level == "error" {
			summary.ConsoleErrorCount++
		}
	}
	for _, n := range r.Network {
		if n.StatusCode >= 400 {
			summary.NetworkErrorCount++
		}
	}

	return summary
}

// SaveJSON writes the result metadata and evidence as pretty JSON.
func (r *ResultWrapper) SaveJSON(path string) error {
	if r == nil || r.Result == nil {
		return fmt.Errorf("截图结果为空")
	}
	if path == "" {
		return fmt.Errorf("JSON 输出路径为空")
	}

	data, err := json.MarshalIndent(r.Result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化截图结果失败: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入 JSON 结果失败: %v", err)
	}
	return nil
}

// SaveHTML writes captured HTML to a file.
func (r *ResultWrapper) SaveHTML(path string) error {
	if r == nil || r.Result == nil {
		return fmt.Errorf("截图结果为空")
	}
	if path == "" {
		return fmt.Errorf("HTML 输出路径为空")
	}
	if r.HTML == "" {
		return fmt.Errorf("截图结果不包含 HTML")
	}
	if err := os.WriteFile(path, []byte(r.HTML), 0644); err != nil {
		return fmt.Errorf("写入 HTML 失败: %v", err)
	}
	return nil
}

// ReadScreenshot returns screenshot bytes from memory or the screenshot file.
func (r *ResultWrapper) ReadScreenshot() ([]byte, error) {
	if r == nil || r.Result == nil {
		return nil, fmt.Errorf("截图结果为空")
	}
	if len(r.ScreenshotBytes) > 0 {
		return r.ScreenshotBytes, nil
	}
	if r.Screenshot == "" {
		return nil, fmt.Errorf("截图结果不包含截图")
	}

	data, err := os.ReadFile(r.Screenshot)
	if err != nil {
		return nil, fmt.Errorf("读取截图文件失败: %v", err)
	}
	return data, nil
}

// WriteScreenshot writes screenshot bytes to the provided writer.
func (r *ResultWrapper) WriteScreenshot(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("截图输出 writer 为空")
	}

	data, err := r.ReadScreenshot()
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	if err != nil {
		return fmt.Errorf("写入截图失败: %v", err)
	}
	if n != len(data) {
		return fmt.Errorf("写入截图失败: %v", io.ErrShortWrite)
	}
	return nil
}

// SaveScreenshot writes screenshot bytes to a file.
func (r *ResultWrapper) SaveScreenshot(path string) error {
	if path == "" {
		return fmt.Errorf("截图输出路径为空")
	}

	data, err := r.ReadScreenshot()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入截图失败: %v", err)
	}
	return nil
}

// TitleOrDefault 返回页面标题，空则返回默认值
func (r *ResultWrapper) TitleOrDefault(defaultTitle string) string {
	if r == nil || r.Title == "" {
		return defaultTitle
	}
	return r.Title
}

// ResponseCodeOrDefault 返回状态码，0 则返回默认值
func (r *ResultWrapper) ResponseCodeOrDefault(defaultCode int) int {
	if r == nil || r.ResponseCode == 0 {
		return defaultCode
	}
	return r.ResponseCode
}

// HeaderMap 返回 HTTP 头的 map 形式
func (r *ResultWrapper) HeaderMap() map[string][]string {
	if r == nil {
		return nil
	}
	return r.Result.HeaderMap()
}

// CookieMap 返回 Cookie 的 map 形式 (name -> value)
func (r *ResultWrapper) CookieMap() map[string]string {
	if r == nil {
		return nil
	}
	cookies := make(map[string]string, len(r.Cookies))
	for _, c := range r.Cookies {
		cookies[c.Name] = c.Value
	}
	return cookies
}

// HeaderValue 获取指定 HTTP 头的值（第一个匹配）
func (r *ResultWrapper) HeaderValue(name string) string {
	if r == nil {
		return ""
	}
	for _, h := range r.Headers {
		if h.Name == name {
			return h.Value
		}
	}
	return ""
}

// ConsoleErrors 返回控制台中的错误日志
func (r *ResultWrapper) ConsoleErrors() []string {
	if r == nil {
		return nil
	}
	var errors []string
	for _, c := range r.Console {
		if c.Level == "error" {
			errors = append(errors, c.Message)
		}
	}
	return errors
}

// NetworkErrors 返回失败的网络请求
func (r *ResultWrapper) NetworkErrors() []models.NetworkLog {
	if r == nil {
		return nil
	}
	var errors []models.NetworkLog
	for _, n := range r.Network {
		if n.StatusCode >= 400 {
			errors = append(errors, n)
		}
	}
	return errors
}

// TechnologyNames 返回检测到的技术名称列表
func (r *ResultWrapper) TechnologyNames() []string {
	if r == nil {
		return nil
	}
	names := make([]string, len(r.Technologies))
	for i, t := range r.Technologies {
		names[i] = t.Name
	}
	return names
}

// TLSInfo 返回 TLS 信息（如果可用）
func (r *ResultWrapper) TLSInfo() *models.TLS {
	if r == nil {
		return nil
	}
	return &r.TLS
}

func hasTLSInfo(tls models.TLS) bool {
	return tls.Version != "" ||
		tls.CipherSuite != "" ||
		tls.Issuer != "" ||
		tls.Subject != "" ||
		!tls.NotBefore.IsZero() ||
		!tls.NotAfter.IsZero() ||
		tls.SANs != "" ||
		tls.FingerprintSHA1 != ""
}
