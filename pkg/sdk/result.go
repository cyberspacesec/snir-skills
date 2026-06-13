package sdk

import "github.com/cyberspacesec/snir-skills/pkg/models"

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
	return r.HeaderMap()
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
