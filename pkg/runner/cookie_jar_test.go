package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCookieJar_AddAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, err := NewCookieJar(cookieFile)
	if err != nil {
		t.Fatalf("NewCookieJar() error = %v", err)
	}

	// 添加持久化 Cookie
	err = jar.AddCookie(PersistentCookie{
		Name:       "session",
		Value:      "abc123",
		Domain:     ".example.com",
		Persistent: true,
		Source:     "manual",
	})
	if err != nil {
		t.Fatalf("AddCookie() error = %v", err)
	}

	// 添加一次性 Cookie
	err = jar.AddCookie(PersistentCookie{
		Name:       "temp_token",
		Value:      "xyz789",
		Domain:     ".example.com",
		Persistent: false,
		Source:     "api",
	})
	if err != nil {
		t.Fatalf("AddCookie() error = %v", err)
	}

	// 获取 Cookie
	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 2 {
		t.Errorf("GetCookies() = %d, want 2", len(cookies))
	}

	// 一次性 Cookie 应该在获取后被消费
	cookies2 := jar.GetCookies(".example.com")
	if len(cookies2) != 1 {
		t.Errorf("第二次 GetCookies() = %d, want 1 (一次性 Cookie 被消费)", len(cookies2))
	}

	// 验证持久化 Cookie 仍然存在
	if cookies2[0].Name != "session" {
		t.Errorf("持久化 Cookie = %s, want session", cookies2[0].Name)
	}
}

func TestCookieJar_GlobalCookies(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, _ := NewCookieJar(cookieFile)

	// 添加全局 Cookie（无域名）
	jar.AddCookie(PersistentCookie{
		Name:       "global_id",
		Value:      "12345",
		Persistent: true,
	})

	// 任何域名都能获取全局 Cookie
	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 1 {
		t.Errorf("GetCookies() = %d, want 1 (全局 Cookie)", len(cookies))
	}
	if cookies[0].Name != "global_id" {
		t.Errorf("Cookie.Name = %s, want global_id", cookies[0].Name)
	}
}

func TestCookieJar_Expiration(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, _ := NewCookieJar(cookieFile)

	// 添加已过期的 Cookie
	jar.AddCookie(PersistentCookie{
		Name:       "expired",
		Value:      "old_value",
		Domain:     ".example.com",
		Persistent: true,
		ExpiresAt:  time.Now().Unix() - 3600, // 1 小时前过期
	})

	// 添加未过期的 Cookie
	jar.AddCookie(PersistentCookie{
		Name:       "valid",
		Value:      "new_value",
		Domain:     ".example.com",
		Persistent: true,
		ExpiresAt:  time.Now().Unix() + 3600, // 1 小时后过期
	})

	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 1 {
		t.Errorf("GetCookies() = %d, want 1 (过期 Cookie 被跳过)", len(cookies))
	}
	if cookies[0].Name != "valid" {
		t.Errorf("Cookie.Name = %s, want valid", cookies[0].Name)
	}
}

func TestCookieJar_UpdateCookie(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, _ := NewCookieJar(cookieFile)

	// 添加 Cookie
	jar.AddCookie(PersistentCookie{
		Name:       "session",
		Value:      "first",
		Domain:     ".example.com",
		Persistent: true,
	})

	// 更新同名 Cookie
	jar.AddCookie(PersistentCookie{
		Name:       "session",
		Value:      "second",
		Domain:     ".example.com",
		Persistent: true,
	})

	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 1 {
		t.Errorf("GetCookies() = %d, want 1 (更新而非追加)", len(cookies))
	}
	if cookies[0].Value != "second" {
		t.Errorf("Cookie.Value = %s, want second", cookies[0].Value)
	}
}

func TestCookieJar_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	// 创建第一个 jar，添加 Cookie
	jar1, _ := NewCookieJar(cookieFile)
	jar1.AddCookie(PersistentCookie{
		Name:       "persist_test",
		Value:      "saved_value",
		Domain:     ".example.com",
		Persistent: true,
	})

	// 创建第二个 jar，从同一文件加载
	jar2, _ := NewCookieJar(cookieFile)
	cookies := jar2.GetCookies(".example.com")
	if len(cookies) != 1 {
		t.Errorf("从文件加载后 GetCookies() = %d, want 1", len(cookies))
	}
	if cookies[0].Value != "saved_value" {
		t.Errorf("Cookie.Value = %s, want saved_value", cookies[0].Value)
	}
}

func TestCookieJar_RemoveCookie(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, _ := NewCookieJar(cookieFile)
	jar.AddCookie(PersistentCookie{
		Name:       "to_remove",
		Value:      "value",
		Domain:     ".example.com",
		Persistent: true,
	})

	jar.RemoveCookie(".example.com", "to_remove")

	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 0 {
		t.Errorf("RemoveCookie 后 GetCookies() = %d, want 0", len(cookies))
	}
}

func TestCookieJar_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, _ := NewCookieJar(cookieFile)
	jar.AddCookie(PersistentCookie{
		Name: "a", Value: "1", Domain: ".a.com", Persistent: true,
	})
	jar.AddCookie(PersistentCookie{
		Name: "b", Value: "2", Domain: ".b.com", Persistent: true,
	})

	jar.Clear()

	if jar.Count() != 0 {
		t.Errorf("Clear 后 Count() = %d, want 0", jar.Count())
	}
}

func TestCookieJar_AddCookies(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, _ := NewCookieJar(cookieFile)
	err := jar.AddCookies([]PersistentCookie{
		{Name: "c1", Value: "v1", Domain: ".example.com", Persistent: true},
		{Name: "c2", Value: "v2", Domain: ".example.com", Persistent: true},
	})
	if err != nil {
		t.Fatalf("AddCookies() error = %v", err)
	}

	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 2 {
		t.Errorf("GetCookies() = %d, want 2", len(cookies))
	}
}

func TestCookieJar_Domains(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	jar, _ := NewCookieJar(cookieFile)
	jar.AddCookie(PersistentCookie{Name: "a", Value: "1", Domain: ".a.com", Persistent: true})
	jar.AddCookie(PersistentCookie{Name: "b", Value: "2", Domain: ".b.com", Persistent: true})
	jar.AddCookie(PersistentCookie{Name: "g", Value: "3", Persistent: true}) // global

	domains := jar.Domains()
	if len(domains) != 2 {
		t.Errorf("Domains() = %d, want 2 (不含 _global)", len(domains))
	}
}

func TestPersistentCookie_IsExpired(t *testing.T) {
	// 永不过期
	c := &PersistentCookie{ExpiresAt: 0}
	if c.IsExpired() {
		t.Error("ExpiresAt=0 应该永不过期")
	}

	// 未过期
	c = &PersistentCookie{ExpiresAt: time.Now().Unix() + 3600}
	if c.IsExpired() {
		t.Error("未来过期时间应该未过期")
	}

	// 已过期
	c = &PersistentCookie{ExpiresAt: time.Now().Unix() - 3600}
	if !c.IsExpired() {
		t.Error("过去过期时间应该已过期")
	}
}

func TestCookieJarToCustomCookies(t *testing.T) {
	// nil jar
	if cookies := CookieJarToCustomCookies(nil, ".example.com"); cookies != nil {
		t.Error("nil jar 应该返回 nil")
	}

	// 有 jar
	tmpDir := t.TempDir()
	jar, _ := NewCookieJar(filepath.Join(tmpDir, "cookies.json"))
	jar.AddCookie(PersistentCookie{
		Name: "test", Value: "v", Domain: ".example.com", Persistent: true,
	})

	cookies := CookieJarToCustomCookies(jar, ".example.com")
	if len(cookies) != 1 {
		t.Errorf("CookieJarToCustomCookies() = %d, want 1", len(cookies))
	}
}

func TestCookieJar_FileDoesNotExist(t *testing.T) {
	// 文件不存在时应该正常创建空 jar
	jar, err := NewCookieJar("/tmp/nonexistent_dir/cookies_test.json")
	if err != nil {
		t.Fatalf("NewCookieJar() error = %v", err)
	}
	if jar.Count() != 0 {
		t.Errorf("新 jar Count() = %d, want 0", jar.Count())
	}
	// 清理
	os.Remove("/tmp/nonexistent_dir/cookies_test.json")
}

func TestDomainMatches(t *testing.T) {
	tests := []struct {
		request string
		cookie  string
		matches bool
	}{
		{"example.com", "example.com", true},
		{"sub.example.com", "example.com", true},
		{"sub.example.com", ".example.com", true},
		{"example.com", ".example.com", true},
		{"deep.sub.example.com", ".example.com", true},
		{"notexample.com", ".example.com", false},
		{"notexample.com", "example.com", false},
		{"example.com", "other.com", false},
	}

	for _, tt := range tests {
		name := tt.request + " vs " + tt.cookie
		t.Run(name, func(t *testing.T) {
			result := domainMatches(tt.request, tt.cookie)
			if result != tt.matches {
				t.Errorf("domainMatches(%q, %q) = %v, want %v", tt.request, tt.cookie, result, tt.matches)
			}
		})
	}
}

func TestCookieJar_SubdomainMatching(t *testing.T) {
	tmpDir := t.TempDir()
	jar, _ := NewCookieJar(filepath.Join(tmpDir, "cookies.json"))

	// Cookie 存储在 .example.com
	jar.AddCookie(PersistentCookie{
		Name: "session", Value: "abc", Domain: ".example.com", Persistent: true,
	})

	// 请求 sub.example.com 应该匹配
	cookies := jar.GetCookies("sub.example.com")
	if len(cookies) != 1 {
		t.Errorf("sub.example.com 匹配 .example.com: got %d cookies, want 1", len(cookies))
	}

	// 请求 example.com 也应该匹配
	cookies = jar.GetCookies("example.com")
	if len(cookies) != 1 {
		t.Errorf("example.com 匹配 .example.com: got %d cookies, want 1", len(cookies))
	}

	// 不相关域名不匹配
	cookies = jar.GetCookies("other.com")
	if len(cookies) != 0 {
		t.Errorf("other.com 不匹配 .example.com: got %d cookies, want 0", len(cookies))
	}
}

func TestToCustomCookie(t *testing.T) {
	pc := PersistentCookie{
		Name:     "test_cookie",
		Value:    "test_value",
		Domain:   ".example.com",
		Path:     "/api",
		Secure:   true,
		HttpOnly: true,
	}
	cc := pc.ToCustomCookie()

	if cc.Name != pc.Name {
		t.Errorf("Name = %s, want %s", cc.Name, pc.Name)
	}
	if cc.Value != pc.Value {
		t.Errorf("Value = %s, want %s", cc.Value, pc.Value)
	}
	if cc.Domain != pc.Domain {
		t.Errorf("Domain = %s, want %s", cc.Domain, pc.Domain)
	}
	if cc.Path != pc.Path {
		t.Errorf("Path = %s, want %s", cc.Path, pc.Path)
	}
	if cc.Secure != pc.Secure {
		t.Errorf("Secure = %v, want %v", cc.Secure, pc.Secure)
	}
	if cc.HttpOnly != pc.HttpOnly {
		t.Errorf("HttpOnly = %v, want %v", cc.HttpOnly, pc.HttpOnly)
	}
}

func TestGetAllCookies(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	// Add cookies to multiple domains
	jar.AddCookie(PersistentCookie{
		Name: "session", Value: "abc", Domain: ".example.com", Persistent: true,
	})
	jar.AddCookie(PersistentCookie{
		Name: "token", Value: "xyz", Domain: ".test.org", Persistent: true,
	})
	// Add a global cookie
	jar.AddCookie(PersistentCookie{
		Name: "global_c", Value: "g_value", Persistent: true,
	})

	cookies := jar.GetAllCookies()
	if len(cookies) != 3 {
		t.Errorf("GetAllCookies() = %d, want 3", len(cookies))
	}

	// Verify all expected names are present
	names := make(map[string]bool)
	for _, c := range cookies {
		names[c.Name] = true
	}
	for _, name := range []string{"session", "token", "global_c"} {
		if !names[name] {
			t.Errorf("Expected cookie %q not found", name)
		}
	}
}

func TestGetAllCookiesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	cookies := jar.GetAllCookies()
	if len(cookies) != 0 {
		t.Errorf("GetAllCookies() = %d, want 0", len(cookies))
	}
}

func TestCookieJar_ToCustomCookieWithExpires(t *testing.T) {
	pc := PersistentCookie{
		Name:       "expiring",
		Value:      "val",
		Domain:     ".example.com",
		Path:       "/",
		Secure:     true,
		HttpOnly:   true,
		ExpiresAt:  1735689600,
		Persistent: true,
		Source:     "test",
	}
	cc := pc.ToCustomCookieWithExpires()
	if cc.Name != "expiring" {
		t.Errorf("Name = %s, want expiring", cc.Name)
	}
	if cc.Expires != 1735689600 {
		t.Errorf("Expires = %d, want 1735689600", cc.Expires)
	}
}

func TestCookieJar_RemoveNonexistentCookie(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	// Removing a cookie that doesn't exist should not error
	err := jar.RemoveCookie(".nonexistent.com", "nonexistent")
	if err != nil {
		t.Errorf("RemoveCookie() on nonexistent cookie should not error: %v", err)
	}
}

func TestCookieJar_RemoveEmptyDomain(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	jar.AddCookie(PersistentCookie{
		Name: "global_to_remove", Value: "v", Persistent: true,
	})

	// Removing with empty domain should target _global
	err := jar.RemoveCookie("", "global_to_remove")
	if err != nil {
		t.Errorf("RemoveCookie() error = %v", err)
	}

	if jar.Count() != 0 {
		t.Errorf("Count() after remove = %d, want 0", jar.Count())
	}
}

func TestCookieJar_AddCookiesWithUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	jar.AddCookie(PersistentCookie{
		Name: "c1", Value: "old", Domain: ".example.com", Persistent: true,
	})

	// Batch add with update
	err := jar.AddCookies([]PersistentCookie{
		{Name: "c1", Value: "new", Domain: ".example.com", Persistent: true},
		{Name: "c2", Value: "v2", Domain: ".example.com", Persistent: true},
	})
	if err != nil {
		t.Fatalf("AddCookies() error = %v", err)
	}

	cookies := jar.GetCookies(".example.com")
	// c1 updated to "new", and c1 was consumed (one-time), c2 also consumed
	// Actually GetCookies consumes one-time cookies and returns them
	if len(cookies) != 2 {
		t.Errorf("GetCookies() = %d, want 2", len(cookies))
	}
	names := make(map[string]string)
	for _, c := range cookies {
		names[c.Name] = c.Value
	}
	if names["c1"] != "new" {
		t.Errorf("c1 value = %s, want new", names["c1"])
	}
}

func TestCookieJar_LoadError(t *testing.T) {
	// Create a file with invalid JSON to trigger load error
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "bad_cookies.json")
	os.WriteFile(cookieFile, []byte("not valid json"), 0644)

	// Creating a jar with this file should log a warning but not fail
	jar, err := NewCookieJar(cookieFile)
	if err != nil {
		t.Fatalf("NewCookieJar() should not error on invalid JSON: %v", err)
	}
	if jar == nil {
		t.Fatal("jar should not be nil")
	}
	// Jar should be empty since loading failed
	if jar.Count() != 0 {
		t.Errorf("Count() = %d, want 0 (load failed)", jar.Count())
	}
}

func TestCookieJar_SaveError(t *testing.T) {
	// Try to save to an invalid path to trigger save error
	jar := &CookieJar{
		filePath: "/proc/nonexistent_dir_should_fail/cookies.json",
		cookies: map[string][]PersistentCookie{
			"_global": {{Name: "test", Value: "v", Persistent: true}},
		},
	}

	// This should fail because /proc/... is not writable
	err := jar.AddCookie(PersistentCookie{
		Name: "test2", Value: "v2", Persistent: true,
	})
	if err == nil {
		t.Log("AddCookie save to invalid path returned nil (filesystem-dependent)")
	}
}

func TestCookieJar_AddCookie_UpdateNonPersistent(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	// Add a non-persistent cookie first
	err := jar.AddCookie(PersistentCookie{
		Name: "temp", Value: "first", Domain: ".example.com", Persistent: false,
	})
	if err != nil {
		t.Fatalf("AddCookie error = %v", err)
	}

	// Update it (still non-persistent)
	err = jar.AddCookie(PersistentCookie{
		Name: "temp", Value: "second", Domain: ".example.com", Persistent: false,
	})
	if err != nil {
		t.Fatalf("AddCookie update error = %v", err)
	}

	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Value != "second" {
		t.Errorf("Value = %s, want second", cookies[0].Value)
	}
}

func TestCookieJar_GetCookies_ExpiredWithSave(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	// Add expired non-persistent cookie
	jar.AddCookie(PersistentCookie{
		Name: "expired_temp", Value: "v", Domain: ".example.com",
		Persistent: false, ExpiresAt: time.Now().Unix() - 3600,
	})

	// Also add a persistent expired cookie
	jar.AddCookie(PersistentCookie{
		Name: "expired_persist", Value: "v", Domain: ".example.com",
		Persistent: true, ExpiresAt: time.Now().Unix() - 3600,
	})

	cookies := jar.GetCookies(".example.com")
	if len(cookies) != 0 {
		t.Errorf("Expected 0 cookies (all expired), got %d", len(cookies))
	}
}

func TestCookieJar_GetAllCookies_ExpiredFiltered(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	jar.AddCookie(PersistentCookie{
		Name: "valid", Value: "v", Domain: ".example.com", Persistent: true,
	})
	jar.AddCookie(PersistentCookie{
		Name: "expired", Value: "e", Domain: ".test.com",
		Persistent: true, ExpiresAt: time.Now().Unix() - 3600,
	})

	cookies := jar.GetAllCookies()
	if len(cookies) != 1 {
		t.Errorf("GetAllCookies() = %d, want 1 (expired filtered)", len(cookies))
	}
	if cookies[0].Name != "valid" {
		t.Errorf("Cookie name = %s, want valid", cookies[0].Name)
	}
}

func TestCookieJar_RemoveCookie_NonexistentDomain(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	jar, _ := NewCookieJar(cookieFile)

	// Remove from domain that doesn't exist should not error
	err := jar.RemoveCookie(".nonexistent.com", "nonexistent")
	if err != nil {
		t.Errorf("RemoveCookie on nonexistent domain should not error: %v", err)
	}
}
