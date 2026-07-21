package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
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

func TestExportResultCookiesToNetscape(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "exported.txt")

	resultCookies := []models.Cookie{
		{Name: "session", Value: "abc123", Domain: ".example.com", Path: "/"},
		{Name: "token", Value: "xyz789", Domain: "", Path: "/api"},
	}

	err := ExportResultCookiesToNetscape(cookieFile, resultCookies, "https://example.com/page")
	if err != nil {
		t.Fatalf("ExportResultCookiesToNetscape() error = %v", err)
	}

	// Verify file was created and is loadable
	loaded, err := LoadNetscapeCookieFile(cookieFile)
	if err != nil {
		t.Fatalf("Failed to reload exported cookies: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("Reloaded cookies count = %d, want 2", len(loaded))
	}

	if loaded[0].Name != "session" {
		t.Errorf("First cookie name = %s, want session", loaded[0].Name)
	}

	// The second cookie had empty domain, should get domain from URL
	if loaded[1].Name != "token" {
		t.Errorf("Second cookie name = %s, want token", loaded[1].Name)
	}
}

func TestLoadNetscapeCookieFile_InvalidFile(t *testing.T) {
	_, err := LoadNetscapeCookieFile("/nonexistent/cookies.txt")
	if err == nil {
		t.Error("Should return error for nonexistent file")
	}
}

func TestLoadNetscapeCookieFileToJar_InvalidFile(t *testing.T) {
	_, _, err := LoadNetscapeCookieFileToJar("/nonexistent/cookies.txt", true, "test")
	if err == nil {
		t.Error("Should return error for nonexistent file")
	}
}

func TestSaveNetscapeCookieFile_ExpiredSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "output.txt")

	cookies := []PersistentCookie{
		{Name: "valid", Value: "v", Domain: ".example.com", Persistent: true},
		{Name: "expired", Value: "e", Domain: ".example.com", Persistent: true, ExpiresAt: time.Now().Unix() - 3600},
	}

	err := SaveNetscapeCookieFile(cookieFile, cookies)
	if err != nil {
		t.Fatalf("SaveNetscapeCookieFile() error = %v", err)
	}

	loaded, _ := LoadNetscapeCookieFile(cookieFile)
	if len(loaded) != 1 {
		t.Errorf("Expired cookie should be skipped: got %d, want 1", len(loaded))
	}
	if loaded[0].Name != "valid" {
		t.Errorf("Cookie name = %s, want valid", loaded[0].Name)
	}
}

func TestSaveNetscapeCookieFile_EmptyDomain(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "output.txt")

	cookies := []PersistentCookie{
		{Name: "c", Value: "v", Domain: "", Path: "", Persistent: true},
	}

	err := SaveNetscapeCookieFile(cookieFile, cookies)
	if err != nil {
		t.Fatalf("SaveNetscapeCookieFile() error = %v", err)
	}

	loaded, _ := LoadNetscapeCookieFile(cookieFile)
	if len(loaded) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(loaded))
	}
	// Empty domain should become _global
	if loaded[0].Domain != "_global" {
		t.Errorf("Domain = %s, want _global", loaded[0].Domain)
	}
}

func TestLoadNetscapeCookieFile_HttpOnly(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	// Netscape format with HttpOnly in 8th column
	content := ".example.com\tTRUE\t/\tFALSE\t0\tsession\tabc123\tTRUE\n"
	os.WriteFile(cookieFile, []byte(content), 0644)

	cookies, err := LoadNetscapeCookieFile(cookieFile)
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFile() error = %v", err)
	}

	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}
	if !cookies[0].HttpOnly {
		t.Error("HttpOnly should be true")
	}
}

func TestLoadNetscapeCookieFile_HttpOnlyLowercase(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	content := ".example.com\tTRUE\t/\tFALSE\t0\tsession\tabc123\thttponly\n"
	os.WriteFile(cookieFile, []byte(content), 0644)

	cookies, err := LoadNetscapeCookieFile(cookieFile)
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFile() error = %v", err)
	}

	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}
	if !cookies[0].HttpOnly {
		t.Error("HttpOnly should be true (lowercase)")
	}
}

func TestLoadNetscapeCookieFileToJar_ExpiredSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	content := ".example.com\tTRUE\t/\tFALSE\t9999999999\tauth\txyz789\n"
	os.WriteFile(cookieFile, []byte(content), 0644)

	jar, cookies, err := LoadNetscapeCookieFileToJar(cookieFile, false, "file")
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFileToJar() error = %v", err)
	}

	if len(cookies) != 1 {
		t.Errorf("Expected 1 cookie, got %d", len(cookies))
	}
	if jar.Count() != 1 {
		t.Errorf("jar.Count() = %d, want 1", jar.Count())
	}
	if cookies[0].Persistent {
		t.Error("persistent should be false")
	}
	if cookies[0].Source != "file" {
		t.Errorf("Source = %s, want file", cookies[0].Source)
	}
}

// TestLoadNetscapeCookieFile_HttpOnlyAndSession 覆盖 LoadNetscapeCookieFileToJar
// 的 HttpOnly 字段（>7列）+ 会话 Cookie（expires=0）分支（line 156-164）。
func TestLoadNetscapeCookieFile_HttpOnlyAndSession(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	// Netscape 格式 8 列（含 HttpOnly）：domain flag path secure expires name value httponly
	content := ".example.com\tTRUE\t/\tTRUE\t0\tsess\tval\tTRUE\n"
	content += ".example.com\tTRUE\t/\tFALSE\t9999999999\tpersist\tval2\tHTTPONLY\n"
	if err := os.WriteFile(cookieFile, []byte(content), 0644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}
	jar, cookies, err := LoadNetscapeCookieFileToJar(cookieFile, true, "import")
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFileToJar 失败: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("应返回 2 cookie, got %d", len(cookies))
	}
	// 找到 session cookie（expiresAt=0）
	var sessFound bool
	var sess PersistentCookie
	for i := range cookies {
		if cookies[i].Name == "sess" {
			sess = cookies[i]
			sessFound = true
		}
	}
	if !sessFound {
		t.Fatal("未找到 session cookie")
	}
	if sess.ExpiresAt != 0 {
		t.Errorf("session cookie ExpiresAt 应为 0, got %d", sess.ExpiresAt)
	}
	_ = jar
}

// TestLoadNetscapeCookieFile_ExpiredSkipped 覆盖 LoadNetscapeCookieFileToJar
// 的过期 Cookie 跳过分支（line 167-169）。
func TestLoadNetscapeCookieFile_ExpiredSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.txt")
	// expires=1（1970 年，已过期）
	content := ".example.com\tTRUE\t/\tFALSE\t1\texpired\tval\n"
	if err := os.WriteFile(cookieFile, []byte(content), 0644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}
	_, cookies, err := LoadNetscapeCookieFileToJar(cookieFile, true, "import")
	if err != nil {
		t.Fatalf("LoadNetscapeCookieFileToJar 失败: %v", err)
	}
	if len(cookies) != 0 {
		t.Errorf("已过期 Cookie 应被跳过, got %d", len(cookies))
	}
}
