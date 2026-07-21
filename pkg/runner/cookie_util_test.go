package runner

import (
	"testing"
)

func TestParseCookieHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		domain   string
		expected int
		first    CustomCookie
	}{
		{
			name:     "single cookie",
			input:    "session=abc123",
			domain:   ".example.com",
			expected: 1,
			first:    CustomCookie{Name: "session", Value: "abc123", Domain: ".example.com", Path: "/"},
		},
		{
			name:     "multiple cookies",
			input:    "session=abc123; token=xyz789",
			domain:   ".example.com",
			expected: 2,
			first:    CustomCookie{Name: "session", Value: "abc123", Domain: ".example.com", Path: "/"},
		},
		{
			name:     "with attributes (skipped)",
			input:    "session=abc123; path=/; domain=.example.com; secure",
			domain:   "",
			expected: 1,
			first:    CustomCookie{Name: "session", Value: "abc123", Path: "/"},
		},
		{
			name:     "empty string",
			input:    "",
			domain:   "",
			expected: 0,
		},
		{
			name:     "empty value",
			input:    "flag=",
			domain:   ".test.com",
			expected: 1,
			first:    CustomCookie{Name: "flag", Value: "", Domain: ".test.com", Path: "/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookies := ParseCookieHeader(tt.input, tt.domain)
			if len(cookies) != tt.expected {
				t.Fatalf("got %d cookies, want %d", len(cookies), tt.expected)
			}
			if tt.expected > 0 {
				if cookies[0].Name != tt.first.Name {
					t.Errorf("name = %s, want %s", cookies[0].Name, tt.first.Name)
				}
				if cookies[0].Value != tt.first.Value {
					t.Errorf("value = %s, want %s", cookies[0].Value, tt.first.Value)
				}
				if tt.first.Domain != "" && cookies[0].Domain != tt.first.Domain {
					t.Errorf("domain = %s, want %s", cookies[0].Domain, tt.first.Domain)
				}
			}
		})
	}
}

func TestCustomCookiesToHeaderString(t *testing.T) {
	cookies := []CustomCookie{
		{Name: "session", Value: "abc123"},
		{Name: "token", Value: "xyz789"},
	}

	result := CustomCookiesToHeaderString(cookies)
	expected := "session=abc123; token=xyz789"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}

	// Empty
	if result := CustomCookiesToHeaderString(nil); result != "" {
		t.Errorf("empty should return empty string, got %q", result)
	}
}

func TestCustomCookie_ToSetCookieString(t *testing.T) {
	tests := []struct {
		name     string
		cookie   CustomCookie
		contains []string
	}{
		{
			name:     "basic",
			cookie:   CustomCookie{Name: "session", Value: "abc"},
			contains: []string{"session=abc"},
		},
		{
			name:     "with domain and path",
			cookie:   CustomCookie{Name: "id", Value: "1", Domain: ".example.com", Path: "/api"},
			contains: []string{"id=1", "Domain=.example.com", "Path=/api"},
		},
		{
			name:     "secure and httponly",
			cookie:   CustomCookie{Name: "auth", Value: "x", Secure: true, HttpOnly: true},
			contains: []string{"auth=x", "Secure", "HttpOnly"},
		},
		{
			name:     "samesite",
			cookie:   CustomCookie{Name: "c", Value: "v", SameSite: "Strict"},
			contains: []string{"c=v", "SameSite=Strict"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cookie.ToSetCookieString()
			for _, s := range tt.contains {
				if !contains(result, s) {
					t.Errorf("result %q does not contain %q", result, s)
				}
			}
		})
	}
}

func TestParseSetCookieHeaders(t *testing.T) {
	headers := []string{
		"session=abc123; Path=/; Domain=.example.com; Secure; HttpOnly; SameSite=Lax",
	}

	cookies := ParseSetCookieHeaders(headers, ".default.com")
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}

	c := cookies[0]
	if c.Name != "session" || c.Value != "abc123" {
		t.Errorf("name/value = %s=%s", c.Name, c.Value)
	}
	if c.Path != "/" {
		t.Errorf("path = %s", c.Path)
	}
	if c.Domain != ".example.com" {
		t.Errorf("domain = %s", c.Domain)
	}
	if !c.Secure {
		t.Error("should be secure")
	}
	if !c.HttpOnly {
		t.Error("should be httponly")
	}
	if c.SameSite != "Lax" {
		t.Errorf("samesite = %s", c.SameSite)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestParseSetCookieHeaders_EdgeBranches 覆盖 ParseSetCookieHeaders 的边缘分支：
// max-age（有效/无效/<=0）、expires（有效/无效）、空 part、无 name（跳过）、
// 多 cookie、仅有 name 无属性。
func TestParseSetCookieHeaders_EdgeBranches(t *testing.T) {
	headers := []string{
		"token=xyz; Max-Age=3600",
		"bad=abc; Max-Age=notanumber",
		"neg=1; Max-Age=0",
		"exp=1; Expires=Wed, 09 Jun 2021 10:18:14 GMT",
		"badexp=1; Expires=not-a-date",
		"empty=1;; ;Path=/x",
		"; Path=/; Domain=.x.com",
		"plain=value",
	}

	cookies := ParseSetCookieHeaders(headers, ".default.com")
	if len(cookies) != 7 {
		t.Fatalf("got %d cookies, want 7", len(cookies))
	}

	if cookies[0].Expires != 3600 {
		t.Errorf("Max-Age=3600 → Expires = %d, want 3600", cookies[0].Expires)
	}
	if cookies[1].Expires != 0 {
		t.Errorf("无效 Max-Age → Expires = %d, want 0", cookies[1].Expires)
	}
	if cookies[2].Expires != 0 {
		t.Errorf("Max-Age=0 → Expires = %d, want 0", cookies[2].Expires)
	}
	if cookies[3].Expires == 0 {
		t.Error("有效 Expires 应解析为非零 Unix 时间戳")
	}
	if cookies[4].Expires != 0 {
		t.Errorf("无效 Expires → Expires = %d, want 0", cookies[4].Expires)
	}
	if cookies[5].Path != "/x" {
		t.Errorf("空 part cookie Path = %s, want /x", cookies[5].Path)
	}
	if cookies[6].Name != "plain" || cookies[6].Value != "value" {
		t.Errorf("plain cookie = %s=%s", cookies[6].Name, cookies[6].Value)
	}
}
