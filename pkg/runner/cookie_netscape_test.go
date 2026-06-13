package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNetscapeCookieFile(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	content := `# Netscape HTTP Cookie File
# https://curl.se/docs/cookie-file.html

.example.com	TRUE	/	FALSE	0	session	abc123
.example.com	TRUE	/	TRUE	1735689600	auth_token	xyz789
.test.org	TRUE	/api	FALSE	0	api_key	key123
`
	if err := os.WriteFile(cookieFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	cookies, err := LoadNetscapeCookieFile(cookieFile)
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFile() error = %v", err)
	}

	if len(cookies) != 3 {
		t.Fatalf("cookies count = %d, want 3", len(cookies))
	}

	// 检查第一个 Cookie
	if cookies[0].Name != "session" || cookies[0].Value != "abc123" {
		t.Errorf("cookie[0] = %s=%s, want session=abc123", cookies[0].Name, cookies[0].Value)
	}
	if cookies[0].Domain != ".example.com" {
		t.Errorf("cookie[0].Domain = %s", cookies[0].Domain)
	}
	if cookies[0].Secure {
		t.Error("cookie[0] should not be secure")
	}

	// 检查第二个 Cookie (Secure)
	if !cookies[1].Secure {
		t.Error("cookie[1] should be secure")
	}

	// 检查第三个 Cookie
	if cookies[2].Path != "/api" {
		t.Errorf("cookie[2].Path = %s, want /api", cookies[2].Path)
	}
}

func TestLoadNetscapeCookieFile_SpaceSeparated(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	// 空格分隔格式（某些工具）
	content := `.example.com TRUE / FALSE 0 session abc123`
	if err := os.WriteFile(cookieFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	cookies, err := LoadNetscapeCookieFile(cookieFile)
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFile() error = %v", err)
	}

	if len(cookies) != 1 {
		t.Fatalf("cookies count = %d, want 1", len(cookies))
	}
	if cookies[0].Name != "session" {
		t.Errorf("cookie.Name = %s, want session", cookies[0].Name)
	}
}

func TestLoadNetscapeCookieFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(cookieFile, []byte("# only comments\n"), 0644)

	_, err := LoadNetscapeCookieFile(cookieFile)
	if err == nil {
		t.Error("空文件应该返回错误")
	}
}

func TestLoadNetscapeCookieFileToJar(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	content := `# Netscape HTTP Cookie File
.example.com	TRUE	/	FALSE	0	session	abc123
.example.com	TRUE	/	TRUE	9999999999	auth_token	xyz789
`
	os.WriteFile(cookieFile, []byte(content), 0644)

	jar, cookies, err := LoadNetscapeCookieFileToJar(cookieFile, true, "import")
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFileToJar() error = %v", err)
	}

	if len(cookies) != 2 {
		t.Errorf("cookies count = %d, want 2", len(cookies))
	}

	if jar == nil {
		t.Fatal("jar is nil")
	}

	if jar.Count() != 2 {
		t.Errorf("jar.Count() = %d, want 2", jar.Count())
	}

	// 检查持久化标记
	if !cookies[0].Persistent {
		t.Error("cookie[0] should be persistent")
	}
	if cookies[0].Source != "import" {
		t.Errorf("cookie[0].Source = %s, want import", cookies[0].Source)
	}
}

func TestSaveNetscapeCookieFile(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "output.txt")

	cookies := []PersistentCookie{
		{Name: "session", Value: "abc", Domain: ".example.com", Path: "/", Persistent: true},
		{Name: "token", Value: "xyz", Domain: ".test.org", Path: "/api", Secure: true, Persistent: true, ExpiresAt: 1893456000},
	}

	err := SaveNetscapeCookieFile(cookieFile, cookies)
	if err != nil {
		t.Fatalf("SaveNetscapeCookieFile() error = %v", err)
	}

	// 验证文件可以重新加载
	loaded, err := LoadNetscapeCookieFile(cookieFile)
	if err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("重新加载后 count = %d, want 2", len(loaded))
	}

	if loaded[0].Name != "session" {
		t.Errorf("loaded[0].Name = %s, want session", loaded[0].Name)
	}
}

func TestSaveAndLoadNetscapeCookie_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "roundtrip.txt")

	original := []PersistentCookie{
		{Name: "a", Value: "1", Domain: ".a.com", Path: "/", Secure: false, HttpOnly: true, Persistent: true},
		{Name: "b", Value: "2", Domain: ".b.com", Path: "/api", Secure: true, Persistent: true},
	}

	SaveNetscapeCookieFile(cookieFile, original)

	loaded, _ := LoadNetscapeCookieFile(cookieFile)
	if len(loaded) != len(original) {
		t.Errorf("roundtrip: %d loaded vs %d original", len(loaded), len(original))
	}

	// 验证字段一致性
	if loaded[0].Name != "a" || loaded[0].Value != "1" {
		t.Errorf("roundtrip cookie[0]: name=%s value=%s", loaded[0].Name, loaded[0].Value)
	}
	if loaded[0].HttpOnly != true {
		t.Error("roundtrip: HttpOnly not preserved")
	}
}
