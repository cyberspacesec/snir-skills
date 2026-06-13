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
		Name:       "a", Value: "1", Domain: ".a.com", Persistent: true,
	})
	jar.AddCookie(PersistentCookie{
		Name:       "b", Value: "2", Domain: ".b.com", Persistent: true,
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
