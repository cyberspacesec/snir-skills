package sdk

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

type fakeDriverPool struct {
	mu              sync.Mutex
	result          *models.Result
	err             error
	lastURL         string
	urls            []string
	lastOptions     runner.Options
	calls           int
	stats           runner.PoolStats
	idleTimeout     time.Duration
	activeCount     int
	closed          bool
	registeredEvent bool
}

func (p *fakeDriverPool) ScreenshotWithContext(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls++
	p.lastURL = target
	p.urls = append(p.urls, target)
	if opts != nil {
		p.lastOptions = *opts
	}
	if p.result != nil || p.err != nil {
		return p.result, p.err
	}
	return &models.Result{URL: target, Title: "ok"}, nil
}

func (p *fakeDriverPool) Stats() runner.PoolStats {
	return p.stats
}

func (p *fakeDriverPool) SetIdleTimeout(timeout time.Duration) {
	p.idleTimeout = timeout
}

func (p *fakeDriverPool) On(handler runner.PoolEventHandler) {
	p.registeredEvent = handler != nil
}

func (p *fakeDriverPool) ActiveCount() int {
	return p.activeCount
}

func (p *fakeDriverPool) Close() {
	p.closed = true
}

func restoreSDKHooks(t *testing.T) {
	t.Helper()

	originalNewDriverPool := newDriverPool
	originalNewCookieJar := newCookieJar
	originalAutoConnect := autoConnect
	originalSharedScreenshotWithContext := sharedScreenshotWithContext
	originalSharedSetIdleTimeout := sharedSetIdleTimeout
	originalSharedPoolStats := sharedPoolStats
	originalCloseSharedPool := closeSharedPool

	t.Cleanup(func() {
		newDriverPool = originalNewDriverPool
		newCookieJar = originalNewCookieJar
		autoConnect = originalAutoConnect
		sharedScreenshotWithContext = originalSharedScreenshotWithContext
		sharedSetIdleTimeout = originalSharedSetIdleTimeout
		sharedPoolStats = originalSharedPoolStats
		closeSharedPool = originalCloseSharedPool
	})
}

func TestNewClient_UnitBranches(t *testing.T) {
	restoreSDKHooks(t)

	pool := &fakeDriverPool{}
	newDriverPool = func(opts *runner.Options, maxConcurrent int) (driverPool, error) {
		if opts.Chrome.WSS != "ws://chrome" {
			t.Fatalf("WSS = %q, want ws://chrome", opts.Chrome.WSS)
		}
		if maxConcurrent != 3 {
			t.Fatalf("maxConcurrent = %d, want 3", maxConcurrent)
		}
		return pool, nil
	}

	jarPath := filepath.Join(t.TempDir(), "cookies.json")
	jar, err := runner.NewCookieJar(jarPath)
	if err != nil {
		t.Fatalf("NewCookieJar() error = %v", err)
	}
	newCookieJar = func(path string) (*runner.CookieJar, error) {
		if path != jarPath {
			t.Fatalf("cookie path = %q, want %q", path, jarPath)
		}
		return jar, nil
	}

	opts := DefaultClientOptions()
	opts.WSSURL = "ws://chrome"
	opts.MaxConcurrent = 3
	opts.CookieFile = jarPath

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.pool != pool {
		t.Fatal("NewClient() did not install factory pool")
	}
	if client.CookieJar() != jar {
		t.Fatal("NewClient() did not install cookie jar")
	}
}

func TestNewClient_CookieLoadWarningKeepsClient(t *testing.T) {
	restoreSDKHooks(t)

	pool := &fakeDriverPool{}
	newDriverPool = func(*runner.Options, int) (driverPool, error) {
		return pool, nil
	}
	newCookieJar = func(string) (*runner.CookieJar, error) {
		return nil, errors.New("bad cookie file")
	}

	opts := DefaultClientOptions()
	opts.CookieFile = filepath.Join(t.TempDir(), "bad.json")

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client.pool != pool {
		t.Fatal("NewClient() did not return the created client")
	}
	if client.CookieJar() != nil {
		t.Fatal("NewClient() should leave CookieJar nil after load failure")
	}
}

func TestNewClient_DefaultFactoryError(t *testing.T) {
	opts := DefaultClientOptions()
	opts.WSSURL = "ws://chrome"
	opts.Proxy = "http://127.0.0.1:8080"

	_, err := NewClient(opts)
	if err == nil {
		t.Fatal("NewClient() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "初始化截图客户端失败") {
		t.Fatalf("NewClient() error = %v", err)
	}
}

func TestNewRemoteClient_UnitBranches(t *testing.T) {
	restoreSDKHooks(t)

	newDriverPool = func(opts *runner.Options, maxConcurrent int) (driverPool, error) {
		if opts.Chrome.WSS != "ws://remote" {
			t.Fatalf("WSS = %q, want ws://remote", opts.Chrome.WSS)
		}
		if maxConcurrent != 7 {
			t.Fatalf("maxConcurrent = %d, want 7", maxConcurrent)
		}
		return &fakeDriverPool{}, nil
	}

	client, err := NewRemoteClient("ws://remote", 7)
	if err != nil {
		t.Fatalf("NewRemoteClient() error = %v", err)
	}
	if client.opts.MaxConcurrent != 7 {
		t.Fatalf("MaxConcurrent = %d, want 7", client.opts.MaxConcurrent)
	}

	newDriverPool = func(_ *runner.Options, maxConcurrent int) (driverPool, error) {
		if maxConcurrent != DefaultClientOptions().MaxConcurrent {
			t.Fatalf("maxConcurrent = %d, want default", maxConcurrent)
		}
		return &fakeDriverPool{}, nil
	}

	client, err = NewRemoteClient("ws://remote", 0)
	if err != nil {
		t.Fatalf("NewRemoteClient() default concurrency error = %v", err)
	}
	if client.opts.MaxConcurrent != DefaultClientOptions().MaxConcurrent {
		t.Fatalf("MaxConcurrent = %d, want default", client.opts.MaxConcurrent)
	}

	newDriverPool = func(*runner.Options, int) (driverPool, error) {
		return nil, errors.New("dial failed")
	}
	_, err = NewRemoteClient("ws://remote", 1)
	if err == nil {
		t.Fatal("NewRemoteClient() error = nil, want error")
	}
}

func TestScreenshotWithContext_UnitBranches(t *testing.T) {
	t.Run("blacklisted target skips pool", func(t *testing.T) {
		pool := &fakeDriverPool{}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		result, err := client.ScreenshotWithContext(context.Background(), "https://example.com", NewScreenshotOptions(
			WithBlacklist("example.com"),
		))
		if err == nil || !strings.Contains(err.Error(), "URL在黑名单中") {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}
		if result == nil || !result.Failed || !strings.Contains(result.FailedReason, "example.com") {
			t.Fatalf("blacklist result = %+v", result)
		}
		if pool.calls != 0 {
			t.Fatalf("pool calls = %d, want 0", pool.calls)
		}
	})

	t.Run("blacklist init error", func(t *testing.T) {
		pool := &fakeDriverPool{}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		result, err := client.ScreenshotWithContext(context.Background(), "https://safe-domain-example.org", NewScreenshotOptions(
			WithBlacklistFile(filepath.Join(t.TempDir(), "missing.txt")),
		))
		if result != nil || err == nil || !strings.Contains(err.Error(), "初始化URL黑名单失败") {
			t.Fatalf("ScreenshotWithContext() result/error = %+v/%v", result, err)
		}
		if pool.calls != 0 {
			t.Fatalf("pool calls = %d, want 0", pool.calls)
		}
	})

	t.Run("pool error", func(t *testing.T) {
		client := &Client{pool: &fakeDriverPool{err: errors.New("boom")}, opts: DefaultClientOptions()}
		_, err := client.ScreenshotWithContext(context.Background(), "https://example.com", nil)
		if err == nil || !strings.Contains(err.Error(), "截图失败") {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}
	})

	t.Run("failed result", func(t *testing.T) {
		result := &models.Result{Failed: true, FailedReason: "blocked"}
		client := &Client{pool: &fakeDriverPool{result: result}, opts: DefaultClientOptions()}
		got, err := client.ScreenshotWithContext(context.Background(), "https://example.com", nil)
		if got != result {
			t.Fatal("ScreenshotWithContext() did not return failed result")
		}
		if err == nil || !strings.Contains(err.Error(), "blocked") {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}
	})

	t.Run("cookie jar merge", func(t *testing.T) {
		jar, err := runner.NewCookieJar("")
		if err != nil {
			t.Fatalf("NewCookieJar() error = %v", err)
		}
		if err := jar.AddCookie(runner.PersistentCookie{Name: "jar", Value: "1", Domain: "example.com"}); err != nil {
			t.Fatalf("AddCookie() error = %v", err)
		}

		pool := &fakeDriverPool{}
		client := &Client{pool: pool, opts: DefaultClientOptions(), cookieJar: jar}
		_, err = client.ScreenshotWithContext(context.Background(), "https://example.com/path", &ScreenshotOptions{
			Cookies: []runner.CustomCookie{{Name: "opt", Value: "2", Domain: "example.com"}},
		})
		if err != nil {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}
		if len(pool.lastOptions.Scan.Cookies) != 2 {
			t.Fatalf("cookies = %v, want 2 cookies", pool.lastOptions.Scan.Cookies)
		}
		if pool.lastOptions.Scan.Cookies[0].Name != "jar" || pool.lastOptions.Scan.Cookies[1].Name != "opt" {
			t.Fatalf("cookie merge order = %v", pool.lastOptions.Scan.Cookies)
		}
	})

	t.Run("per request cookie file merge", func(t *testing.T) {
		jarPath := filepath.Join(t.TempDir(), "cookies.json")
		jar, err := runner.NewCookieJar(jarPath)
		if err != nil {
			t.Fatalf("NewCookieJar() error = %v", err)
		}
		if err := jar.AddCookie(runner.PersistentCookie{
			Name:       "persisted",
			Value:      "yes",
			Domain:     "example.com",
			Path:       "/",
			Persistent: true,
		}); err != nil {
			t.Fatalf("AddCookie() error = %v", err)
		}

		pool := &fakeDriverPool{}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		_, err = client.ScreenshotWithContext(context.Background(), "https://example.com/path", NewScreenshotOptions(
			WithCookieFile(jarPath),
			WithInjectedCookies(runner.CustomCookie{Name: "opt", Value: "2", Domain: "example.com"}),
		))
		if err != nil {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}
		if pool.lastOptions.Scan.CookiesFile != jarPath {
			t.Fatalf("CookiesFile = %q, want %q", pool.lastOptions.Scan.CookiesFile, jarPath)
		}
		if len(pool.lastOptions.Scan.Cookies) != 2 {
			t.Fatalf("cookies = %+v, want 2 cookies", pool.lastOptions.Scan.Cookies)
		}
		if pool.lastOptions.Scan.Cookies[0].Name != "persisted" || pool.lastOptions.Scan.Cookies[1].Name != "opt" {
			t.Fatalf("cookie merge order = %+v", pool.lastOptions.Scan.Cookies)
		}
		if client.CookieJar() != nil {
			t.Fatal("per-request CookieFile should not replace the client CookieJar")
		}
	})

	t.Run("per request cookie file writeback", func(t *testing.T) {
		jarPath := filepath.Join(t.TempDir(), "cookies.json")
		pool := &fakeDriverPool{result: &models.Result{
			Cookies: []models.Cookie{{Name: "session", Value: "new", Domain: "example.com", Path: "/"}},
		}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		_, err := client.ScreenshotWithContext(context.Background(), "https://example.com/path", NewScreenshotOptions(
			WithCookieFile(jarPath),
			WithCookieWriteBack(),
			WithCookies(),
		))
		if err != nil {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}

		loaded, err := runner.NewCookieJar(jarPath)
		if err != nil {
			t.Fatalf("NewCookieJar() reload error = %v", err)
		}
		written := loaded.GetCookies("example.com")
		if len(written) != 1 || written[0].Name != "session" || written[0].Value != "new" {
			t.Fatalf("written cookies = %+v", written)
		}
		if client.CookieJar() != nil {
			t.Fatal("per-request CookieFile writeback should not replace the client CookieJar")
		}
	})

	t.Run("cookie sources and writeback", func(t *testing.T) {
		cookieFile := filepath.Join(t.TempDir(), "cookies.txt")
		content := "# Netscape HTTP Cookie File\n.example.com\tTRUE\t/\tFALSE\t0\timported\tyes\n"
		if err := os.WriteFile(cookieFile, []byte(content), 0644); err != nil {
			t.Fatalf("write cookie file: %v", err)
		}

		pool := &fakeDriverPool{result: &models.Result{
			Cookies: []models.Cookie{{Name: "session", Value: "new", Domain: "example.com", Path: "/"}},
		}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		_, err := client.ScreenshotWithContext(context.Background(), "https://example.com/path", NewScreenshotOptions(
			WithCookieHeader("sid=abc; theme=dark"),
			WithCookieImport(cookieFile),
			WithCookieWriteBack(),
			WithCookies(),
		))
		if err != nil {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}

		names := make(map[string]bool)
		for _, cookie := range pool.lastOptions.Scan.Cookies {
			names[cookie.Name] = true
			if cookie.Name == "sid" && cookie.Domain != "example.com" {
				t.Fatalf("CookieHeader default domain = %q, want example.com", cookie.Domain)
			}
		}
		for _, name := range []string{"sid", "theme", "imported"} {
			if !names[name] {
				t.Fatalf("cookie %q not injected: %+v", name, pool.lastOptions.Scan.Cookies)
			}
		}
		if client.CookieJar() == nil {
			t.Fatal("CookieWriteBack did not create CookieJar")
		}
		written := client.CookieJar().GetCookies("example.com")
		if len(written) != 1 || written[0].Name != "session" || written[0].Value != "new" {
			t.Fatalf("written cookies = %+v", written)
		}
	})

	t.Run("cookie export", func(t *testing.T) {
		exportFile := filepath.Join(t.TempDir(), "export.txt")
		pool := &fakeDriverPool{result: &models.Result{
			Cookies: []models.Cookie{{Name: "exported", Value: "1", Domain: "example.com", Path: "/"}},
		}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		if _, err := client.ScreenshotWithContext(context.Background(), "https://example.com", NewScreenshotOptions(
			WithCookieExport(exportFile),
		)); err != nil {
			t.Fatalf("ScreenshotWithContext() error = %v", err)
		}
		if !pool.lastOptions.Scan.SaveCookies {
			t.Fatal("CookieExport should enable SaveCookies")
		}
		loaded, err := runner.LoadNetscapeCookieFile(exportFile)
		if err != nil {
			t.Fatalf("LoadNetscapeCookieFile() error = %v", err)
		}
		if len(loaded) != 1 || loaded[0].Name != "exported" {
			t.Fatalf("exported cookies = %+v", loaded)
		}
	})
}

func TestScreenshotBytesWithContext_UnitBranches(t *testing.T) {
	t.Run("blacklisted target skips pool", func(t *testing.T) {
		pool := &fakeDriverPool{}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		data, result, err := client.ScreenshotBytesWithContext(context.Background(), "https://example.com", NewScreenshotOptions(
			WithBlacklist("example.com"),
		))
		if data != nil || err == nil || !strings.Contains(err.Error(), "URL在黑名单中") {
			t.Fatalf("ScreenshotBytesWithContext() data/error = %v/%v", data, err)
		}
		if result == nil || !result.Failed || !strings.Contains(result.FailedReason, "example.com") {
			t.Fatalf("blacklist result = %+v", result)
		}
		if pool.calls != 0 {
			t.Fatalf("pool calls = %d, want 0", pool.calls)
		}
	})

	t.Run("pool error", func(t *testing.T) {
		client := &Client{pool: &fakeDriverPool{err: errors.New("boom")}, opts: DefaultClientOptions()}
		data, result, err := client.ScreenshotBytesWithContext(context.Background(), "https://example.com", nil)
		if data != nil || result != nil {
			t.Fatalf("ScreenshotBytesWithContext() data/result = %v/%v, want nil/nil", data, result)
		}
		if err == nil || !strings.Contains(err.Error(), "截图失败") {
			t.Fatalf("ScreenshotBytesWithContext() error = %v", err)
		}
	})

	t.Run("failed result", func(t *testing.T) {
		failed := &models.Result{Failed: true, FailedReason: "blocked"}
		client := &Client{pool: &fakeDriverPool{result: failed}, opts: DefaultClientOptions()}
		data, result, err := client.ScreenshotBytesWithContext(context.Background(), "https://example.com", nil)
		if data != nil || result != failed {
			t.Fatalf("ScreenshotBytesWithContext() data/result = %v/%v", data, result)
		}
		if err == nil || !strings.Contains(err.Error(), "blocked") {
			t.Fatalf("ScreenshotBytesWithContext() error = %v", err)
		}
	})

	t.Run("success sets byte options", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}
		data, result, err := client.ScreenshotBytesWithContext(context.Background(), "https://example.com", nil)
		if err != nil {
			t.Fatalf("ScreenshotBytesWithContext() error = %v", err)
		}
		if string(data) != "png" || result == nil {
			t.Fatalf("ScreenshotBytesWithContext() data/result = %q/%v", data, result)
		}
		if !pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("byte options not set: %+v", pool.lastOptions.Scan)
		}
	})

	t.Run("cookie jar merge and writeback", func(t *testing.T) {
		jar, err := runner.NewCookieJar("")
		if err != nil {
			t.Fatalf("NewCookieJar() error = %v", err)
		}
		if err := jar.AddCookie(runner.PersistentCookie{Name: "jar", Value: "1", Domain: "example.com"}); err != nil {
			t.Fatalf("AddCookie() error = %v", err)
		}

		pool := &fakeDriverPool{result: &models.Result{
			ScreenshotBytes: []byte("png"),
			Cookies:         []models.Cookie{{Name: "bytes", Value: "2", Domain: "example.com", Path: "/"}},
		}}
		client := &Client{pool: pool, opts: DefaultClientOptions(), cookieJar: jar}
		if _, _, err := client.ScreenshotBytesWithContext(context.Background(), "https://example.com", NewScreenshotOptions(
			WithCookieWriteBack(),
			WithCookies(),
		)); err != nil {
			t.Fatalf("ScreenshotBytesWithContext() error = %v", err)
		}
		if len(pool.lastOptions.Scan.Cookies) != 1 || pool.lastOptions.Scan.Cookies[0].Name != "jar" {
			t.Fatalf("cookie jar was not merged into bytes request: %+v", pool.lastOptions.Scan.Cookies)
		}
		written := client.CookieJar().GetCookies("example.com")
		found := false
		for _, cookie := range written {
			if cookie.Name == "bytes" && cookie.Value == "2" {
				found = true
			}
		}
		if !found {
			t.Fatalf("bytes result cookie was not written back: %+v", written)
		}
	})

	t.Run("extract error", func(t *testing.T) {
		client := &Client{pool: &fakeDriverPool{result: &models.Result{}}, opts: DefaultClientOptions()}
		data, result, err := client.ScreenshotBytesWithContext(context.Background(), "https://example.com", nil)
		if data != nil || result == nil {
			t.Fatalf("ScreenshotBytesWithContext() data/result = %v/%v", data, result)
		}
		if err == nil || !strings.Contains(err.Error(), "截图文件路径为空") {
			t.Fatalf("ScreenshotBytesWithContext() error = %v", err)
		}
	})
}

func TestScreenshotBytesFromResult_ErrorBranches(t *testing.T) {
	if _, err := screenshotBytesFromResult(&models.Result{}); err == nil {
		t.Fatal("screenshotBytesFromResult() error = nil, want empty path error")
	}
	if _, err := screenshotBytesFromResult(&models.Result{Screenshot: filepath.Join(t.TempDir(), "missing.png")}); err == nil {
		t.Fatal("screenshotBytesFromResult() error = nil, want read error")
	}
}

func TestScreenshotHTML_ErrorBranch(t *testing.T) {
	client := &Client{pool: &fakeDriverPool{err: errors.New("boom")}, opts: DefaultClientOptions()}
	html, result, err := client.ScreenshotHTML("https://example.com", nil)
	if html != "" || result != nil {
		t.Fatalf("ScreenshotHTML() html/result = %q/%v, want empty/nil", html, result)
	}
	if err == nil {
		t.Fatal("ScreenshotHTML() error = nil, want error")
	}
}

func TestCaptureFunctionalOptions_Unit(t *testing.T) {
	pool := &fakeDriverPool{}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	result, err := client.Capture(
		"https://example.com",
		WithFullPage(),
		WithEvidence(),
		WithDevice("iphone-15"),
		WithViewport(390, 844),
		WithCustomHeaders(map[string]string{"X-Agent": "snir"}),
		WithIgnoreCertErrors(),
		WithDisableWebRTC(),
		WithSpoofedScreen(390, 844),
	)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if result == nil || result.Title != "ok" {
		t.Fatalf("Capture() result = %+v", result)
	}

	if !pool.lastOptions.Scan.CaptureFullPage {
		t.Fatal("Capture() did not set full-page capture")
	}
	if !pool.lastOptions.Scan.SaveHTML || !pool.lastOptions.Scan.SaveHeaders ||
		!pool.lastOptions.Scan.SaveConsole || !pool.lastOptions.Scan.SaveCookies ||
		!pool.lastOptions.Scan.SaveNetwork {
		t.Fatalf("Capture() evidence flags = %+v", pool.lastOptions.Scan)
	}
	if pool.lastOptions.Chrome.DeviceName != "iPhone 15" {
		t.Fatalf("DeviceName = %q, want iPhone 15", pool.lastOptions.Chrome.DeviceName)
	}
	if pool.lastOptions.Chrome.WindowX != 390 || pool.lastOptions.Chrome.WindowY != 844 {
		t.Fatalf("viewport = %dx%d", pool.lastOptions.Chrome.WindowX, pool.lastOptions.Chrome.WindowY)
	}
	if pool.lastOptions.Chrome.CustomHeaders["X-Agent"] != "snir" {
		t.Fatalf("CustomHeaders = %+v", pool.lastOptions.Chrome.CustomHeaders)
	}
	if !pool.lastOptions.Chrome.IgnoreCertErrors || !pool.lastOptions.Chrome.DisableWebRTC ||
		!pool.lastOptions.Chrome.SpoofScreenSize {
		t.Fatalf("browser bools = %+v", pool.lastOptions.Chrome)
	}
}

func TestCaptureBytesFunctionalOptions_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	data, result, err := client.CaptureBytes("https://example.com", WithElement("#main"), WithEvidence())
	if err != nil {
		t.Fatalf("CaptureBytes() error = %v", err)
	}
	if string(data) != "png" || result == nil {
		t.Fatalf("CaptureBytes() data/result = %q/%+v", data, result)
	}
	if pool.lastOptions.Scan.Selector != "#main" {
		t.Fatalf("Selector = %q, want #main", pool.lastOptions.Scan.Selector)
	}
	if !pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
		t.Fatalf("byte options = %+v", pool.lastOptions.Scan)
	}
	if !pool.lastOptions.Scan.SaveNetwork {
		t.Fatal("CaptureBytes() did not keep evidence flags")
	}
}

func TestScenarioConvenienceMethods_Unit(t *testing.T) {
	t.Run("evidence", func(t *testing.T) {
		pool := &fakeDriverPool{}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		if _, err := client.ScreenshotEvidence("https://example.com", nil); err != nil {
			t.Fatalf("ScreenshotEvidence() error = %v", err)
		}
		if !pool.lastOptions.Scan.SaveHTML || !pool.lastOptions.Scan.SaveHeaders ||
			!pool.lastOptions.Scan.SaveConsole || !pool.lastOptions.Scan.SaveCookies ||
			!pool.lastOptions.Scan.SaveNetwork {
			t.Fatalf("evidence flags = %+v", pool.lastOptions.Scan)
		}
	})

	t.Run("element bytes", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		data, _, err := client.ScreenshotElementBytes("https://example.com", "#hero", nil)
		if err != nil {
			t.Fatalf("ScreenshotElementBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("data = %q", data)
		}
		if pool.lastOptions.Scan.Selector != "#hero" {
			t.Fatalf("Selector = %q, want #hero", pool.lastOptions.Scan.Selector)
		}
	})

	t.Run("xpath and full page bytes", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		if _, err := client.ScreenshotXPath("https://example.com", "//main", nil); err != nil {
			t.Fatalf("ScreenshotXPath() error = %v", err)
		}
		if pool.lastOptions.Scan.XPath != "//main" {
			t.Fatalf("XPath = %q, want //main", pool.lastOptions.Scan.XPath)
		}

		if _, _, err := client.ScreenshotFullPageBytes("https://example.com", nil); err != nil {
			t.Fatalf("ScreenshotFullPageBytes() error = %v", err)
		}
		if !pool.lastOptions.Scan.CaptureFullPage {
			t.Fatal("ScreenshotFullPageBytes() did not set full page")
		}
	})

	t.Run("actions form cookies bytes", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		data, _, err := client.ScreenshotWithActionsBytes("https://example.com", []runner.InteractionAction{
			ActionClick("#accept"),
		}, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithActionsBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("actions data = %q", data)
		}
		if len(pool.lastOptions.Scan.Actions) != 1 ||
			pool.lastOptions.Scan.Actions[0].Type != "click" ||
			pool.lastOptions.Scan.Actions[0].Selector != "#accept" {
			t.Fatalf("actions = %+v", pool.lastOptions.Scan.Actions)
		}

		form := FormWithSubmit("#login", 2*time.Second, FormInput("#user", "admin"))
		data, _, err = client.ScreenshotWithFormBytes("https://example.com/login", form, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithFormBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("form data = %q", data)
		}
		if pool.lastOptions.Scan.Form.SubmitSelector != "#login" ||
			pool.lastOptions.Scan.Form.WaitAfterSubmit != 2000 ||
			len(pool.lastOptions.Scan.Form.Fields) != 1 {
			t.Fatalf("form = %+v", pool.lastOptions.Scan.Form)
		}

		cookies := []runner.CustomCookie{{Name: "session", Value: "abc123", Domain: "example.com"}}
		data, _, err = client.ScreenshotWithCookiesBytes("https://example.com/dashboard", cookies, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithCookiesBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("cookies data = %q", data)
		}
		if len(pool.lastOptions.Scan.Cookies) != 1 ||
			pool.lastOptions.Scan.Cookies[0].Name != "session" {
			t.Fatalf("cookies = %+v", pool.lastOptions.Scan.Cookies)
		}
		if !pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("byte options = %+v", pool.lastOptions.Scan)
		}
	})

	t.Run("device viewport js file", func(t *testing.T) {
		pool := &fakeDriverPool{}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		if _, err := client.ScreenshotDevice("https://example.com", "pixel-8-pro", nil); err != nil {
			t.Fatalf("ScreenshotDevice() error = %v", err)
		}
		if pool.lastOptions.Chrome.DeviceName != "Pixel 8 Pro" {
			t.Fatalf("DeviceName = %q, want Pixel 8 Pro", pool.lastOptions.Chrome.DeviceName)
		}

		if _, err := client.ScreenshotViewport("https://example.com", 1440, 900, nil); err != nil {
			t.Fatalf("ScreenshotViewport() error = %v", err)
		}
		if pool.lastOptions.Chrome.WindowX != 1440 || pool.lastOptions.Chrome.WindowY != 900 {
			t.Fatalf("viewport = %dx%d", pool.lastOptions.Chrome.WindowX, pool.lastOptions.Chrome.WindowY)
		}

		if _, err := client.ScreenshotWithJSBefore("https://example.com", "window.preload=true", nil); err != nil {
			t.Fatalf("ScreenshotWithJSBefore() error = %v", err)
		}
		if pool.lastOptions.Scan.JavaScript != "window.preload=true" ||
			!pool.lastOptions.Scan.RunJSBefore || pool.lastOptions.Scan.RunJSAfter {
			t.Fatalf("js before = %+v", pool.lastOptions.Scan)
		}

		if _, err := client.ScreenshotWithJSFile("https://example.com", "script.js", true, nil); err != nil {
			t.Fatalf("ScreenshotWithJSFile() error = %v", err)
		}
		if pool.lastOptions.Scan.JavaScriptFile != "script.js" ||
			!pool.lastOptions.Scan.RunJSBefore || pool.lastOptions.Scan.RunJSAfter {
			t.Fatalf("js file = %+v", pool.lastOptions.Scan)
		}

		if _, err := client.ScreenshotWithJSFile("https://example.com", "after.js", false, nil); err != nil {
			t.Fatalf("ScreenshotWithJSFile() after-load error = %v", err)
		}
		if pool.lastOptions.Scan.JavaScriptFile != "after.js" ||
			pool.lastOptions.Scan.RunJSBefore || !pool.lastOptions.Scan.RunJSAfter {
			t.Fatalf("js file after = %+v", pool.lastOptions.Scan)
		}
	})

	t.Run("device viewport js bytes", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		data, _, err := client.ScreenshotDeviceBytes("https://example.com", "pixel-8-pro", nil)
		if err != nil {
			t.Fatalf("ScreenshotDeviceBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("device data = %q", data)
		}
		if pool.lastOptions.Chrome.DeviceName != "Pixel 8 Pro" {
			t.Fatalf("DeviceName = %q, want Pixel 8 Pro", pool.lastOptions.Chrome.DeviceName)
		}

		data, _, err = client.ScreenshotViewportBytes("https://example.com", 1440, 900, nil)
		if err != nil {
			t.Fatalf("ScreenshotViewportBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("viewport data = %q", data)
		}
		if pool.lastOptions.Chrome.WindowX != 1440 || pool.lastOptions.Chrome.WindowY != 900 {
			t.Fatalf("viewport = %dx%d", pool.lastOptions.Chrome.WindowX, pool.lastOptions.Chrome.WindowY)
		}

		data, _, err = client.ScreenshotWithJSBytes("https://example.com", "document.body.dataset.ready='1'", nil)
		if err != nil {
			t.Fatalf("ScreenshotWithJSBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("js data = %q", data)
		}
		if pool.lastOptions.Scan.JavaScript != "document.body.dataset.ready='1'" ||
			pool.lastOptions.Scan.RunJSBefore || !pool.lastOptions.Scan.RunJSAfter {
			t.Fatalf("js after = %+v", pool.lastOptions.Scan)
		}

		data, _, err = client.ScreenshotWithJSBeforeBytes("https://example.com", "window.preload=true", nil)
		if err != nil {
			t.Fatalf("ScreenshotWithJSBeforeBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("js before data = %q", data)
		}
		if pool.lastOptions.Scan.JavaScript != "window.preload=true" ||
			!pool.lastOptions.Scan.RunJSBefore || pool.lastOptions.Scan.RunJSAfter {
			t.Fatalf("js before = %+v", pool.lastOptions.Scan)
		}

		data, _, err = client.ScreenshotWithJSFileBytes("https://example.com", "preload.js", true, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithJSFileBytes() error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("js file data = %q", data)
		}
		if pool.lastOptions.Scan.JavaScriptFile != "preload.js" ||
			!pool.lastOptions.Scan.RunJSBefore || pool.lastOptions.Scan.RunJSAfter {
			t.Fatalf("js file before = %+v", pool.lastOptions.Scan)
		}

		data, _, err = client.ScreenshotWithJSFileBytes("https://example.com", "after.js", false, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithJSFileBytes() after-load error = %v", err)
		}
		if string(data) != "png" {
			t.Fatalf("js file after data = %q", data)
		}
		if pool.lastOptions.Scan.JavaScriptFile != "after.js" ||
			pool.lastOptions.Scan.RunJSBefore || !pool.lastOptions.Scan.RunJSAfter {
			t.Fatalf("js file after = %+v", pool.lastOptions.Scan)
		}
	})
}

func TestMergeWithScreenshotOptions_PerRequestBrowserOverrides(t *testing.T) {
	base := toRunnerOptions(DefaultClientOptions())
	merged := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithViewport(1600, 900),
		WithDeviceEmulation(390, 844, 3, true, true),
		WithTouchEmulation(false),
		WithIgnoreCertErrors(),
		WithProxyList(runner.ProxyRoundRobin, "http://a:8080", "http://b:8080"),
		WithPorts(8080, 8443),
		WithAcceptLanguage("en-US"),
		WithCustomHeaders(map[string]string{"X-Test": "1"}),
		WithFingerprint("Linux x86_64", "Google Inc.", "Mesa", "llvmpipe"),
		WithPlugins("Chrome PDF Viewer"),
		WithDisableWebRTC(),
		WithSpoofedScreen(1600, 900),
	))

	if merged.Chrome.WindowX != 390 || merged.Chrome.WindowY != 844 {
		t.Fatalf("viewport = %dx%d", merged.Chrome.WindowX, merged.Chrome.WindowY)
	}
	if merged.Chrome.DeviceScaleFactor != 3 || !merged.Chrome.IsMobile || merged.Chrome.HasTouch {
		t.Fatalf("device emulation = dpr:%v mobile:%t touch:%t", merged.Chrome.DeviceScaleFactor, merged.Chrome.IsMobile, merged.Chrome.HasTouch)
	}
	if !merged.Chrome.IgnoreCertErrors {
		t.Fatal("IgnoreCertErrors was not merged")
	}
	if len(merged.Chrome.ProxyList) != 2 || merged.Chrome.ProxyStrategy != runner.ProxyRoundRobin {
		t.Fatalf("proxy rotation = %+v", merged.Chrome)
	}
	if len(merged.Scan.Ports) != 2 || merged.Scan.Ports[0] != 8080 {
		t.Fatalf("ports = %+v", merged.Scan.Ports)
	}
	if merged.Chrome.AcceptLanguage != "en-US" {
		t.Fatalf("AcceptLanguage = %q", merged.Chrome.AcceptLanguage)
	}
	if merged.Chrome.CustomHeaders["X-Test"] != "1" {
		t.Fatalf("CustomHeaders = %+v", merged.Chrome.CustomHeaders)
	}
	if merged.Chrome.Platform != "Linux x86_64" || merged.Chrome.Vendor != "Google Inc." ||
		merged.Chrome.WebGLVendor != "Mesa" || merged.Chrome.WebGLRenderer != "llvmpipe" {
		t.Fatalf("fingerprint = %+v", merged.Chrome)
	}
	if len(merged.Chrome.Plugins) != 1 || merged.Chrome.Plugins[0] != "Chrome PDF Viewer" {
		t.Fatalf("Plugins = %v", merged.Chrome.Plugins)
	}
	if !merged.Chrome.DisableWebRTC || !merged.Chrome.SpoofScreenSize ||
		merged.Chrome.ScreenWidth != 1600 || merged.Chrome.ScreenHeight != 900 {
		t.Fatalf("privacy/screen = %+v", merged.Chrome)
	}
}

func TestMergeWithScreenshotOptions_BlacklistOverrides(t *testing.T) {
	base := toRunnerOptions(DefaultClientOptions())
	base.Scan.BlacklistPatterns = []string{"base.example"}
	base.Scan.BlacklistFile = "base.txt"

	custom := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithBlacklist("*.internal.*"),
		WithBlacklistFile("request.txt"),
	))
	if !custom.Scan.EnableBlacklist || custom.Scan.DefaultBlacklist {
		t.Fatalf("blacklist flags = %+v", custom.Scan)
	}
	if len(custom.Scan.BlacklistPatterns) != 1 || custom.Scan.BlacklistPatterns[0] != "*.internal.*" {
		t.Fatalf("blacklist patterns = %+v", custom.Scan.BlacklistPatterns)
	}
	if custom.Scan.BlacklistFile != "request.txt" {
		t.Fatalf("blacklist file = %q", custom.Scan.BlacklistFile)
	}

	disabled := mergeWithScreenshotOptions(base, NewScreenshotOptions(WithNoBlacklist()))
	if disabled.Scan.EnableBlacklist || disabled.Scan.DefaultBlacklist ||
		len(disabled.Scan.BlacklistPatterns) != 0 || disabled.Scan.BlacklistFile != "" {
		t.Fatalf("disabled blacklist = %+v", disabled.Scan)
	}
}

func TestToRunnerOptions_ClientCookieProxyPorts(t *testing.T) {
	opts := DefaultClientOptions()
	opts.ProxyList = []string{"http://a:8080"}
	opts.ProxyFile = "proxies.txt"
	opts.ProxyURL = "https://proxy.example/api"
	opts.ProxyStrategy = runner.ProxyRandom
	opts.Ports = []int{80, 443}
	opts.CookieHeader = "sid=abc"
	opts.CookieStrings = []string{"theme=dark"}
	opts.CookieImport = "cookies.txt"
	opts.CookieExport = "out.txt"
	opts.CookieFile = "cookies.json"
	opts.CookieWriteBack = true
	opts.DeviceScaleFactor = 2
	opts.IsMobile = true
	opts.HasTouch = true

	got := toRunnerOptions(opts)
	if len(got.Chrome.ProxyList) != 1 || got.Chrome.ProxyFile != "proxies.txt" ||
		got.Chrome.ProxyURL != "https://proxy.example/api" || got.Chrome.ProxyStrategy != runner.ProxyRandom {
		t.Fatalf("proxy fields = %+v", got.Chrome)
	}
	if len(got.Scan.Ports) != 2 || got.Scan.Ports[1] != 443 {
		t.Fatalf("ports = %v", got.Scan.Ports)
	}
	if len(got.Scan.CookieStrings) != 2 || got.Scan.CookieStrings[0] != "sid=abc" ||
		got.Scan.CookieImport != "cookies.txt" || got.Scan.CookieExport != "out.txt" ||
		got.Scan.CookiesFile != "cookies.json" || !got.Scan.CookieWriteBack || !got.Scan.SaveCookies {
		t.Fatalf("cookie fields = %+v", got.Scan)
	}
	if got.Chrome.DeviceScaleFactor != 2 || !got.Chrome.IsMobile || !got.Chrome.HasTouch {
		t.Fatalf("device fields = %+v", got.Chrome)
	}
}

func TestMergeWithScreenshotOptions_CookieFile(t *testing.T) {
	base := toRunnerOptions(DefaultClientOptions())
	base.Scan.CookiesFile = "client.json"

	got := mergeWithScreenshotOptions(base, NewScreenshotOptions(WithCookieFile("request.json")))
	if got.Scan.CookiesFile != "request.json" {
		t.Fatalf("CookiesFile = %q, want request.json", got.Scan.CookiesFile)
	}
}

func TestMergeWithScreenshotOptions_ProxySourceOverrides(t *testing.T) {
	base := toRunnerOptions(DefaultClientOptions())
	base.Chrome.Proxy = "http://static:8080"
	base.Chrome.ProxyList = []string{"http://list:8080"}
	base.Chrome.ProxyFile = "proxies.txt"
	base.Chrome.ProxyURL = "https://proxy.example/api"

	static := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithProxy("http://override:8080"),
	))
	if static.Chrome.Proxy != "http://override:8080" ||
		len(static.Chrome.ProxyList) != 0 || static.Chrome.ProxyFile != "" || static.Chrome.ProxyURL != "" {
		t.Fatalf("static proxy override = %+v", static.Chrome)
	}

	list := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithProxyList(runner.ProxyRoundRobin, "http://a:8080", "http://b:8080"),
	))
	if list.Chrome.Proxy != "" || len(list.Chrome.ProxyList) != 2 ||
		list.Chrome.ProxyFile != "" || list.Chrome.ProxyURL != "" {
		t.Fatalf("proxy list override = %+v", list.Chrome)
	}

	file := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithProxyFile("request-proxies.txt", runner.ProxySequential),
	))
	if file.Chrome.Proxy != "" || file.Chrome.ProxyFile != "request-proxies.txt" ||
		len(file.Chrome.ProxyList) != 0 || file.Chrome.ProxyURL != "" {
		t.Fatalf("proxy file override = %+v", file.Chrome)
	}

	url := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithProxyURL("https://request-proxy.example/api", runner.ProxyRandom),
	))
	if url.Chrome.Proxy != "" || url.Chrome.ProxyURL != "https://request-proxy.example/api" ||
		len(url.Chrome.ProxyList) != 0 || url.Chrome.ProxyFile != "" {
		t.Fatalf("proxy url override = %+v", url.Chrome)
	}
}

func TestMergeWithScreenshotOptions_JSExecutionTiming(t *testing.T) {
	base := toRunnerOptions(DefaultClientOptions())

	before := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithJSBefore("window.preload = true"),
	))
	if before.Scan.JavaScript != "window.preload = true" ||
		!before.Scan.RunJSBefore || before.Scan.RunJSAfter {
		t.Fatalf("before timing = %+v", before.Scan)
	}

	afterFile := mergeWithScreenshotOptions(base, NewScreenshotOptions(
		WithJSFile("after.js", false),
	))
	if afterFile.Scan.JavaScriptFile != "after.js" ||
		afterFile.Scan.RunJSBefore || !afterFile.Scan.RunJSAfter {
		t.Fatalf("after file timing = %+v", afterFile.Scan)
	}

	baseBefore := base
	baseBefore.Scan.RunJSBefore = true
	baseBefore.Scan.RunJSAfter = false
	overrideAfter := mergeWithScreenshotOptions(baseBefore, NewScreenshotOptions(
		WithJSFile("override-after.js", false),
	))
	if overrideAfter.Scan.RunJSBefore || !overrideAfter.Scan.RunJSAfter {
		t.Fatalf("override after timing = %+v", overrideAfter.Scan)
	}

	defaultAfter := mergeWithScreenshotOptions(base, &ScreenshotOptions{JavaScript: "window.after = true"})
	if !defaultAfter.Scan.RunJSAfter {
		t.Fatalf("default JS timing = %+v", defaultAfter.Scan)
	}
}

func TestBatchScreenshotStreaming_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &Client{pool: &fakeDriverPool{}, opts: DefaultClientOptions()}
	ch := client.BatchScreenshotStreaming(ctx, []string{"https://a.test", "https://b.test"}, nil)

	count := 0
	for result := range ch {
		count++
		if !errors.Is(result.Error, context.Canceled) {
			t.Fatalf("BatchScreenshotStreaming() error = %v, want context.Canceled", result.Error)
		}
		if result.Result != nil {
			t.Fatalf("BatchScreenshotStreaming() result = %v, want nil", result.Result)
		}
	}
	if count != 2 {
		t.Fatalf("BatchScreenshotStreaming() count = %d, want 2", count)
	}
}

func TestExpandTargets_Unit(t *testing.T) {
	got := ExpandTarget("example.com/admin?x=1", NewScreenshotOptions(
		WithHTTPOnly(),
		WithPorts(80, 8080),
	))
	want := []string{
		"http://example.com:80/admin?x=1",
		"http://example.com:8080/admin?x=1",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ExpandTarget() = %#v, want %#v", got, want)
	}

	explicit := ExpandTargets([]string{"https://example.com:9443/path"}, NewScreenshotOptions(
		WithHTTPAndHTTPS(),
		WithPorts(80, 443),
	))
	if !slices.Equal(explicit, []string{"https://example.com:9443/path"}) {
		t.Fatalf("ExpandTargets() explicit = %#v", explicit)
	}
}

func TestClientExpandTargetsUsesClientAndRequestOptions(t *testing.T) {
	opts := DefaultClientOptions()
	opts.Ports = []int{80, 443}
	client := &Client{opts: opts}

	got := client.ExpandTarget("example.com", NewScreenshotOptions(WithHTTPSOnly(), WithPorts(8443)))
	want := []string{"https://example.com:8443"}
	if !slices.Equal(got, want) {
		t.Fatalf("Client.ExpandTarget() = %#v, want %#v", got, want)
	}
}

func TestBatchScreenshotTargets_Unit(t *testing.T) {
	pool := &fakeDriverPool{}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	results := client.BatchScreenshotTargets([]string{"example.com/path", "https://already.test/login"}, NewScreenshotOptions(
		WithHTTPOnly(),
		WithPorts(80, 8080),
	))
	wantURLs := []string{
		"http://example.com:80/path",
		"http://example.com:8080/path",
		"https://already.test/login",
	}
	if len(results) != len(wantURLs) {
		t.Fatalf("BatchScreenshotTargets() len = %d, want %d", len(results), len(wantURLs))
	}
	for i, want := range wantURLs {
		if results[i].URL != want || results[i].Result == nil || results[i].Error != nil {
			t.Fatalf("result[%d] = %+v, want URL %q without error", i, results[i], want)
		}
	}
	if pool.calls != len(wantURLs) {
		t.Fatalf("pool calls = %d, want %d", pool.calls, len(wantURLs))
	}
}

func TestBatchScreenshotTargets_BlacklistBeforePool(t *testing.T) {
	pool := &fakeDriverPool{}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	results := client.BatchScreenshotTargets([]string{"blocked.example.com", "safe.example.org"}, NewScreenshotOptions(
		WithHTTPOnly(),
		WithPorts(80),
		WithBlacklist("blocked.example.com"),
	))
	if len(results) != 2 {
		t.Fatalf("BatchScreenshotTargets() len = %d, want 2", len(results))
	}
	if results[0].Error == nil || results[0].Result == nil || !results[0].Result.Failed {
		t.Fatalf("blacklisted result = %+v", results[0])
	}
	if results[1].Error != nil || results[1].Result == nil {
		t.Fatalf("safe result = %+v", results[1])
	}
	if pool.calls != 1 || !slices.Equal(pool.urls, []string{"http://safe.example.org:80"}) {
		t.Fatalf("pool calls/urls = %d/%#v", pool.calls, pool.urls)
	}
}

func TestBatchScreenshotTargetsCallback_Unit(t *testing.T) {
	pool := &fakeDriverPool{}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	seen := map[string]bool{}
	client.BatchScreenshotTargetsCallback(context.Background(), []string{"example.com"}, NewScreenshotOptions(
		WithHTTPSOnly(),
		WithPorts(443),
	), func(result BatchResult) {
		seen[result.URL] = true
	})

	if !seen["https://example.com:443"] || pool.calls != 1 {
		t.Fatalf("callback seen/pool calls = %#v/%d", seen, pool.calls)
	}
}

func TestClientPoolMethods_Unit(t *testing.T) {
	pool := &fakeDriverPool{
		stats:       runner.PoolStats{MaxConcurrent: 9, ActiveCount: 4},
		activeCount: 4,
	}
	client := &Client{pool: pool}

	if stats := client.Stats(); stats.MaxConcurrent != 9 {
		t.Fatalf("Stats().MaxConcurrent = %d, want 9", stats.MaxConcurrent)
	}
	client.SetIdleTimeout(5 * time.Second)
	if pool.idleTimeout != 5*time.Second {
		t.Fatalf("idleTimeout = %v, want 5s", pool.idleTimeout)
	}
	client.OnEvent(func(runner.PoolEvent) {})
	if !pool.registeredEvent {
		t.Fatal("OnEvent() did not register handler")
	}
	if active := client.ActiveCount(); active != 4 {
		t.Fatalf("ActiveCount() = %d, want 4", active)
	}
	client.Close()
	if !pool.closed {
		t.Fatal("Close() did not close pool")
	}
}

func TestClientCookieMethods_Unit(t *testing.T) {
	t.Run("add cookie jar creation error", func(t *testing.T) {
		restoreSDKHooks(t)
		newCookieJar = func(string) (*runner.CookieJar, error) {
			return nil, errors.New("jar failed")
		}
		client := &Client{}
		if err := client.AddCookie(runner.PersistentCookie{Name: "a"}); err == nil {
			t.Fatal("AddCookie() error = nil, want error")
		}
	})

	t.Run("add persistent cookie", func(t *testing.T) {
		jarPath := filepath.Join(t.TempDir(), "cookies.json")
		jar, err := runner.NewCookieJar(jarPath)
		if err != nil {
			t.Fatalf("NewCookieJar() error = %v", err)
		}
		client := &Client{cookieJar: jar}
		if err := client.AddPersistentCookie("session", "abc", "example.com"); err != nil {
			t.Fatalf("AddPersistentCookie() error = %v", err)
		}
		if _, err := os.Stat(jarPath); err != nil {
			t.Fatalf("persistent cookie file was not written: %v", err)
		}
	})
}

func TestApplyDevicePreset_Invalid(t *testing.T) {
	opts := runner.Options{}
	applyDevicePreset("missing-device", &opts)
	if opts.Chrome.DeviceName != "" {
		t.Fatalf("DeviceName = %q, want empty", opts.Chrome.DeviceName)
	}
}

func TestSharedWrappers_Unit(t *testing.T) {
	t.Run("blacklisted target skips shared pool", func(t *testing.T) {
		restoreSDKHooks(t)
		called := false
		sharedScreenshotWithContext = func(context.Context, string, *runner.Options) (*models.Result, error) {
			called = true
			return &models.Result{Title: "unexpected"}, nil
		}
		result, err := SharedScreenshotWithContext(context.Background(), "https://example.com", NewScreenshotOptions(
			WithBlacklist("example.com"),
		))
		if err != nil {
			t.Fatalf("SharedScreenshotWithContext() error = %v", err)
		}
		if result == nil || !result.Failed || !strings.Contains(result.FailedReason, "example.com") {
			t.Fatalf("blacklist result = %+v", result)
		}
		if called {
			t.Fatal("sharedScreenshotWithContext was called for a blacklisted target")
		}
	})

	t.Run("screenshot success and option merge", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if target != "https://example.com" {
				t.Fatalf("target = %q", target)
			}
			if !opts.Scan.SaveHTML {
				t.Fatal("SaveHTML was not merged")
			}
			return &models.Result{URL: target, Title: "ok"}, nil
		}
		result, err := SharedScreenshot("https://example.com", &ScreenshotOptions{SaveHTML: true})
		if err != nil {
			t.Fatalf("SharedScreenshot() error = %v", err)
		}
		if result.Title != "ok" {
			t.Fatalf("SharedScreenshot() title = %q, want ok", result.Title)
		}
	})

	t.Run("screenshot error", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(context.Context, string, *runner.Options) (*models.Result, error) {
			return nil, errors.New("shared failed")
		}
		result, err := SharedScreenshotWithContext(context.Background(), "https://example.com", nil)
		if result != nil || err == nil {
			t.Fatalf("SharedScreenshotWithContext() result/error = %v/%v", result, err)
		}
	})

	t.Run("failed result is returned without error", func(t *testing.T) {
		restoreSDKHooks(t)
		failed := &models.Result{Failed: true, FailedReason: "blocked"}
		sharedScreenshotWithContext = func(context.Context, string, *runner.Options) (*models.Result, error) {
			return failed, nil
		}
		result, err := SharedScreenshotWithContext(context.Background(), "https://example.com", nil)
		if result != failed || err != nil {
			t.Fatalf("SharedScreenshotWithContext() result/error = %v/%v", result, err)
		}
	})

	t.Run("idle timeout", func(t *testing.T) {
		restoreSDKHooks(t)
		var got time.Duration
		sharedSetIdleTimeout = func(timeout time.Duration) error {
			got = timeout
			return nil
		}
		SharedSetIdleTimeout(3 * time.Second)
		if got != 3*time.Second {
			t.Fatalf("timeout = %v, want 3s", got)
		}

		sharedSetIdleTimeout = func(time.Duration) error {
			return errors.New("idle failed")
		}
		SharedSetIdleTimeout(time.Second)
	})

	t.Run("stats and close", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedPoolStats = func() (runner.PoolStats, error) {
			return runner.PoolStats{MaxConcurrent: 5}, nil
		}
		stats, err := SharedStats()
		if err != nil {
			t.Fatalf("SharedStats() error = %v", err)
		}
		if stats.MaxConcurrent != 5 {
			t.Fatalf("MaxConcurrent = %d, want 5", stats.MaxConcurrent)
		}

		sharedPoolStats = func() (runner.PoolStats, error) {
			return runner.PoolStats{}, errors.New("stats failed")
		}
		if _, err := SharedStats(); err == nil {
			t.Fatal("SharedStats() error = nil, want error")
		}

		closed := false
		closeSharedPool = func() {
			closed = true
		}
		CloseSharedPool()
		if !closed {
			t.Fatal("CloseSharedPool() did not call runner hook")
		}
	})
}

func TestAutoConnectClient_UnitBranches(t *testing.T) {
	modes := []AutoConnectMode{AutoConnectRemote, AutoConnectDiscovered, AutoConnectLocal}
	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			restoreSDKHooks(t)
			pool := &fakeDriverPool{}
			autoConnect = func(opts *runner.Options, maxConcurrent int) (driverPool, string, error) {
				if opts.Scan.ScreenshotPath != "screenshots" {
					t.Fatalf("ScreenshotPath = %q", opts.Scan.ScreenshotPath)
				}
				if maxConcurrent != DefaultClientOptions().MaxConcurrent {
					t.Fatalf("maxConcurrent = %d, want default", maxConcurrent)
				}
				return pool, string(mode), nil
			}

			client, gotMode, err := AutoConnectClient(DefaultClientOptions())
			if err != nil {
				t.Fatalf("AutoConnectClient() error = %v", err)
			}
			if gotMode != mode {
				t.Fatalf("mode = %s, want %s", gotMode, mode)
			}
			if client.pool != pool {
				t.Fatal("AutoConnectClient() did not install pool")
			}
		})
	}

	t.Run("stub error", func(t *testing.T) {
		restoreSDKHooks(t)
		autoConnect = func(*runner.Options, int) (driverPool, string, error) {
			return nil, "", errors.New("connect failed")
		}
		client, mode, err := AutoConnectClient(DefaultClientOptions())
		if client != nil || mode != "" || err == nil {
			t.Fatalf("AutoConnectClient() client/mode/error = %v/%q/%v", client, mode, err)
		}
	})
}

func TestAutoConnectClient_DefaultFactoryError(t *testing.T) {
	opts := DefaultClientOptions()
	opts.WSSURL = "ws://chrome"
	opts.Proxy = "http://127.0.0.1:8080"

	client, mode, err := AutoConnectClient(opts)
	if client != nil || mode != "" || err == nil {
		t.Fatalf("AutoConnectClient() client/mode/error = %v/%q/%v", client, mode, err)
	}
}
