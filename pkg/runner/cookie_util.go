package runner

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ParseCookieHeader 解析 Cookie Header 格式字符串
// 支持格式：
//   - "name=value"
//   - "name=value; name2=value2"
//   - "name=value; path=/; domain=.example.com"
//
// 返回 CustomCookie 列表
func ParseCookieHeader(headerStr string, defaultDomain string) []CustomCookie {
	if headerStr == "" {
		return nil
	}

	var cookies []CustomCookie
	pairs := strings.Split(headerStr, ";")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}

		// 跳过 Cookie 属性名（不是 Cookie 值）
		lowerName := strings.ToLower(name)
		if lowerName == "path" || lowerName == "domain" || lowerName == "expires" ||
			lowerName == "max-age" || lowerName == "secure" || lowerName == "httponly" ||
			lowerName == "samesite" {
			continue
		}

		value := ""
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}

		cookie := CustomCookie{
			Name:   name,
			Value:  value,
			Domain: defaultDomain,
			Path:   "/",
		}
		cookies = append(cookies, cookie)
	}

	return cookies
}

// ParseSetCookieHeaders 解析 HTTP Set-Cookie 头
// 格式：name=value; Path=/; Domain=.example.com; Secure; HttpOnly; SameSite=Lax; Max-Age=3600
func ParseSetCookieHeaders(headers []string, defaultDomain string) []CustomCookie {
	var cookies []CustomCookie

	for _, header := range headers {
		cookie := CustomCookie{
			Domain: defaultDomain,
			Path:   "/",
		}

		parts := strings.Split(header, ";")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			kv := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := ""
			if len(kv) > 1 {
				val = strings.TrimSpace(kv[1])
			}

			if i == 0 {
				// 第一个部分是 name=value
				cookie.Name = key
				cookie.Value = val
				continue
			}

			switch strings.ToLower(key) {
			case "path":
				cookie.Path = val
			case "domain":
				cookie.Domain = val
			case "secure":
				cookie.Secure = true
			case "httponly":
				cookie.HttpOnly = true
			case "samesite":
				cookie.SameSite = val
			case "max-age":
				if seconds, err := strconv.ParseInt(val, 10, 64); err == nil && seconds > 0 {
					cookie.Expires = seconds // 相对秒数，调用者需转为绝对时间
				}
			case "expires":
				// 尝试解析 HTTP 日期格式
				if t, err := http.ParseTime(val); err == nil {
					cookie.Expires = t.Unix()
				}
			}
		}

		if cookie.Name != "" {
			cookies = append(cookies, cookie)
		}
	}

	return cookies
}

// CustomCookie.ToHeaderString 将 Cookie 转换为 HTTP Cookie Header 格式
// 返回 "name=value" 格式（不含属性）
func (c CustomCookie) ToHeaderString() string {
	return fmt.Sprintf("%s=%s", c.Name, c.Value)
}

// CustomCookiesToHeaderString 将 Cookie 列表转为 Cookie Header 值
// 返回 "name1=value1; name2=value2" 格式
func CustomCookiesToHeaderString(cookies []CustomCookie) string {
	if len(cookies) == 0 {
		return ""
	}

	var parts []string
	for _, c := range cookies {
		parts = append(parts, c.ToHeaderString())
	}
	return strings.Join(parts, "; ")
}

// CustomCookie.ToSetCookieString 将 Cookie 转换为 Set-Cookie 头格式
// 返回完整的 Set-Cookie 值（含属性）
func (c CustomCookie) ToSetCookieString() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))

	if c.Domain != "" {
		parts = append(parts, fmt.Sprintf("Domain=%s", c.Domain))
	}
	if c.Path != "" {
		parts = append(parts, fmt.Sprintf("Path=%s", c.Path))
	}
	if c.Expires > 0 {
		parts = append(parts, fmt.Sprintf("Max-Age=%d", c.Expires))
	}
	if c.Secure {
		parts = append(parts, "Secure")
	}
	if c.HttpOnly {
		parts = append(parts, "HttpOnly")
	}
	if c.SameSite != "" {
		parts = append(parts, fmt.Sprintf("SameSite=%s", c.SameSite))
	}

	return strings.Join(parts, "; ")
}

// PersistentCookie.ToCustomCookieWithExpires 转换为 CustomCookie（保留 Expires）
func (pc *PersistentCookie) ToCustomCookieWithExpires() CustomCookie {
	return CustomCookie{
		Name:     pc.Name,
		Value:    pc.Value,
		Domain:   pc.Domain,
		Path:     pc.Path,
		Secure:   pc.Secure,
		HttpOnly: pc.HttpOnly,
		Expires:  pc.ExpiresAt,
	}
}
