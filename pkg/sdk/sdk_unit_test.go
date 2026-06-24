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
	optionsByURL    map[string]runner.Options
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
		if p.optionsByURL == nil {
			p.optionsByURL = map[string]runner.Options{}
		}
		p.optionsByURL[target] = *opts
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

	t.Run("evidence output timing helpers", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{
			Headers:         []models.Header{{Name: "Content-Type", Value: "text/html"}},
			Console:         []models.ConsoleLog{{Level: "error", Message: "boom"}},
			Network:         []models.NetworkLog{{URL: "https://example.com/api", StatusCode: 200}},
			ScreenshotBytes: []byte("jpeg"),
		}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		headers, result, err := client.ScreenshotHeaders("https://example.com", nil)
		if err != nil {
			t.Fatalf("ScreenshotHeaders() error = %v", err)
		}
		if len(headers) != 1 || headers[0].Name != "Content-Type" || result == nil {
			t.Fatalf("headers/result = %+v/%+v", headers, result)
		}
		if !pool.lastOptions.Scan.SaveHeaders {
			t.Fatal("ScreenshotHeaders() did not enable headers")
		}

		data, _, err := client.ScreenshotConsoleBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("ScreenshotConsoleBytes() error = %v", err)
		}
		if string(data) != "jpeg" || !pool.lastOptions.Scan.SaveConsole ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("console bytes/options = %q/%+v", data, pool.lastOptions.Scan)
		}

		network, _, err := client.ScreenshotNetwork("https://example.com", nil)
		if err != nil {
			t.Fatalf("ScreenshotNetwork() error = %v", err)
		}
		if len(network) != 1 || network[0].StatusCode != 200 || !pool.lastOptions.Scan.SaveNetwork {
			t.Fatalf("network/options = %+v/%+v", network, pool.lastOptions.Scan)
		}

		data, _, err = client.ScreenshotWithFormatBytes("https://example.com", "jpeg", 82, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithFormatBytes() error = %v", err)
		}
		if string(data) != "jpeg" || pool.lastOptions.Scan.ScreenshotFormat != "jpeg" ||
			pool.lastOptions.Scan.ScreenshotQuality != 82 {
			t.Fatalf("format bytes/options = %q/%+v", data, pool.lastOptions.Scan)
		}

		if _, err := client.ScreenshotToPath("https://example.com", "captures/example", nil); err != nil {
			t.Fatalf("ScreenshotToPath() error = %v", err)
		}
		if pool.lastOptions.Scan.ScreenshotPath != "captures/example" {
			t.Fatalf("ScreenshotPath = %q", pool.lastOptions.Scan.ScreenshotPath)
		}

		if _, _, err := client.ScreenshotWithDelayBytes("https://example.com", 3*time.Second, nil); err != nil {
			t.Fatalf("ScreenshotWithDelayBytes() error = %v", err)
		}
		if pool.lastOptions.Chrome.Delay != 3 {
			t.Fatalf("Delay = %d", pool.lastOptions.Chrome.Delay)
		}

		if _, err := client.ScreenshotWithTimeout("https://example.com", 17*time.Second, nil); err != nil {
			t.Fatalf("ScreenshotWithTimeout() error = %v", err)
		}
		if pool.lastOptions.Chrome.Timeout != 17 {
			t.Fatalf("Timeout = %d", pool.lastOptions.Chrome.Timeout)
		}
	})

	t.Run("request profile helpers", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{
			ScreenshotBytes: []byte("png"),
			Cookies:         []models.Cookie{{Name: "exported", Value: "1", Domain: "example.com", Path: "/"}},
		}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		data, _, err := client.ScreenshotWithProxyListBytes("https://example.com", runner.ProxyRoundRobin, []string{
			"http://a:8080",
			"http://b:8080",
		}, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithProxyListBytes() error = %v", err)
		}
		if string(data) != "png" || len(pool.lastOptions.Chrome.ProxyList) != 2 ||
			pool.lastOptions.Chrome.ProxyStrategy != runner.ProxyRoundRobin ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("proxy list bytes data/options = %q/%+v/%+v", data, pool.lastOptions.Chrome, pool.lastOptions.Scan)
		}

		if _, err := client.ScreenshotWithProxy("https://example.com", "http://static:8080", NewScreenshotOptions(
			WithProxyList(runner.ProxyRandom, "http://old:8080"),
		)); err != nil {
			t.Fatalf("ScreenshotWithProxy() error = %v", err)
		}
		if pool.lastOptions.Chrome.Proxy != "http://static:8080" ||
			len(pool.lastOptions.Chrome.ProxyList) != 0 {
			t.Fatalf("proxy override options = %+v", pool.lastOptions.Chrome)
		}

		if _, err := client.ScreenshotWithProxyFile("https://example.com", "proxies.txt", runner.ProxyRandom, nil); err != nil {
			t.Fatalf("ScreenshotWithProxyFile() error = %v", err)
		}
		if pool.lastOptions.Chrome.ProxyFile != "proxies.txt" ||
			pool.lastOptions.Chrome.ProxyStrategy != runner.ProxyRandom {
			t.Fatalf("proxy file options = %+v", pool.lastOptions.Chrome)
		}

		if _, _, err := client.ScreenshotWithProxyURLBytes("https://example.com", "https://proxy-api.example/list", runner.ProxyRoundRobin, nil); err != nil {
			t.Fatalf("ScreenshotWithProxyURLBytes() error = %v", err)
		}
		if pool.lastOptions.Chrome.ProxyURL != "https://proxy-api.example/list" ||
			pool.lastOptions.Chrome.ProxyStrategy != runner.ProxyRoundRobin {
			t.Fatalf("proxy url options = %+v", pool.lastOptions.Chrome)
		}

		headers := map[string]string{"X-Test": "1"}
		if _, err := client.ScreenshotWithCustomHeaders("https://example.com", headers, nil); err != nil {
			t.Fatalf("ScreenshotWithCustomHeaders() error = %v", err)
		}
		if pool.lastOptions.Chrome.CustomHeaders["X-Test"] != "1" {
			t.Fatalf("CustomHeaders = %+v", pool.lastOptions.Chrome.CustomHeaders)
		}

		if _, err := client.ScreenshotWithUserAgent("https://example.com", "snir-test-agent", nil); err != nil {
			t.Fatalf("ScreenshotWithUserAgent() error = %v", err)
		}
		if pool.lastOptions.Chrome.UserAgent != "snir-test-agent" {
			t.Fatalf("UserAgent = %q", pool.lastOptions.Chrome.UserAgent)
		}

		if _, err := client.ScreenshotWithAcceptLanguage("https://example.com", "zh-CN,zh;q=0.9", nil); err != nil {
			t.Fatalf("ScreenshotWithAcceptLanguage() error = %v", err)
		}
		if pool.lastOptions.Chrome.AcceptLanguage != "zh-CN,zh;q=0.9" {
			t.Fatalf("AcceptLanguage = %q", pool.lastOptions.Chrome.AcceptLanguage)
		}

		if _, err := client.ScreenshotWithFingerprint("https://example.com", "Linux x86_64", "Google Inc.", "Mesa", "llvmpipe", nil); err != nil {
			t.Fatalf("ScreenshotWithFingerprint() error = %v", err)
		}
		if pool.lastOptions.Chrome.Platform != "Linux x86_64" ||
			pool.lastOptions.Chrome.Vendor != "Google Inc." ||
			pool.lastOptions.Chrome.WebGLVendor != "Mesa" ||
			pool.lastOptions.Chrome.WebGLRenderer != "llvmpipe" {
			t.Fatalf("fingerprint options = %+v", pool.lastOptions.Chrome)
		}

		if _, err := client.ScreenshotWithCookieHeader("https://example.com/path", "sid=abc; theme=dark", nil); err != nil {
			t.Fatalf("ScreenshotWithCookieHeader() error = %v", err)
		}
		if len(pool.lastOptions.Scan.Cookies) != 2 ||
			pool.lastOptions.Scan.Cookies[0].Name != "sid" ||
			pool.lastOptions.Scan.Cookies[0].Domain != "example.com" {
			t.Fatalf("cookie header cookies = %+v", pool.lastOptions.Scan.Cookies)
		}

		jarPath := filepath.Join(t.TempDir(), "cookies.json")
		if _, _, err := client.ScreenshotWithCookieFileBytes("https://example.com", jarPath, true, nil); err != nil {
			t.Fatalf("ScreenshotWithCookieFileBytes() error = %v", err)
		}
		if pool.lastOptions.Scan.CookiesFile != jarPath || !pool.lastOptions.Scan.CookieWriteBack ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("cookie file bytes options = %+v", pool.lastOptions.Scan)
		}

		importFile := filepath.Join(t.TempDir(), "cookies.txt")
		importContent := "# Netscape HTTP Cookie File\n.example.com\tTRUE\t/\tFALSE\t0\timported\tyes\n"
		if err := os.WriteFile(importFile, []byte(importContent), 0644); err != nil {
			t.Fatalf("write import cookie file: %v", err)
		}
		if _, err := client.ScreenshotWithCookieImport("https://example.com", importFile, nil); err != nil {
			t.Fatalf("ScreenshotWithCookieImport() error = %v", err)
		}
		if pool.lastOptions.Scan.CookieImport != importFile ||
			len(pool.lastOptions.Scan.Cookies) != 1 ||
			pool.lastOptions.Scan.Cookies[0].Name != "imported" {
			t.Fatalf("cookie import options = %+v", pool.lastOptions.Scan)
		}

		exportFile := filepath.Join(t.TempDir(), "export.txt")
		if _, _, err := client.ScreenshotWithCookieExportBytes("https://example.com", exportFile, nil); err != nil {
			t.Fatalf("ScreenshotWithCookieExportBytes() error = %v", err)
		}
		if pool.lastOptions.Scan.CookieExport != exportFile || !pool.lastOptions.Scan.SaveCookies ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("cookie export bytes options = %+v", pool.lastOptions.Scan)
		}
		exported, err := runner.LoadNetscapeCookieFile(exportFile)
		if err != nil {
			t.Fatalf("LoadNetscapeCookieFile() error = %v", err)
		}
		if len(exported) != 1 || exported[0].Name != "exported" {
			t.Fatalf("exported cookies = %+v", exported)
		}

		if _, err := client.ScreenshotWithBlacklist("https://example.com", []string{"blocked.example"}, nil); err != nil {
			t.Fatalf("ScreenshotWithBlacklist() error = %v", err)
		}
		if !pool.lastOptions.Scan.EnableBlacklist || pool.lastOptions.Scan.DefaultBlacklist ||
			len(pool.lastOptions.Scan.BlacklistPatterns) != 1 ||
			pool.lastOptions.Scan.BlacklistPatterns[0] != "blocked.example" {
			t.Fatalf("blacklist options = %+v", pool.lastOptions.Scan)
		}

		blacklistFile := filepath.Join(t.TempDir(), "blacklist.txt")
		if err := os.WriteFile(blacklistFile, []byte("blocked.example\n"), 0644); err != nil {
			t.Fatalf("write blacklist file: %v", err)
		}
		if _, _, err := client.ScreenshotWithBlacklistFileBytes("https://example.com", blacklistFile, nil); err != nil {
			t.Fatalf("ScreenshotWithBlacklistFileBytes() error = %v", err)
		}
		if !pool.lastOptions.Scan.EnableBlacklist || pool.lastOptions.Scan.BlacklistFile != blacklistFile ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("blacklist file bytes options = %+v", pool.lastOptions.Scan)
		}

		blockedOpts := NewScreenshotOptions(WithBlacklist("example.com"))
		if _, err := client.ScreenshotWithoutBlacklist("https://example.com", blockedOpts); err != nil {
			t.Fatalf("ScreenshotWithoutBlacklist() error = %v", err)
		}
		if pool.lastOptions.Scan.EnableBlacklist || pool.lastOptions.Scan.DefaultBlacklist ||
			len(pool.lastOptions.Scan.BlacklistPatterns) != 0 || pool.lastOptions.Scan.BlacklistFile != "" {
			t.Fatalf("without blacklist options = %+v", pool.lastOptions.Scan)
		}

		if _, _, err := client.ScreenshotWithRetriesBytes("https://example.com", 4, nil); err != nil {
			t.Fatalf("ScreenshotWithRetriesBytes() error = %v", err)
		}
		if pool.lastOptions.Scan.MaxRetries != 4 ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("retries bytes options = %+v", pool.lastOptions.Scan)
		}
	})

	t.Run("browser environment helpers", func(t *testing.T) {
		pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
		client := &Client{pool: pool, opts: DefaultClientOptions()}

		data, _, err := client.ScreenshotWithDeviceEmulationBytes("https://example.com", 412, 915, 2.75, true, true, nil)
		if err != nil {
			t.Fatalf("ScreenshotWithDeviceEmulationBytes() error = %v", err)
		}
		if string(data) != "png" ||
			pool.lastOptions.Chrome.WindowX != 412 ||
			pool.lastOptions.Chrome.WindowY != 915 ||
			pool.lastOptions.Chrome.DeviceScaleFactor != 2.75 ||
			!pool.lastOptions.Chrome.IsMobile ||
			!pool.lastOptions.Chrome.HasTouch ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("device emulation bytes data/options = %q/%+v/%+v", data, pool.lastOptions.Chrome, pool.lastOptions.Scan)
		}

		if _, err := client.ScreenshotWithMobileEmulation("https://example.com", 3, nil); err != nil {
			t.Fatalf("ScreenshotWithMobileEmulation() error = %v", err)
		}
		if pool.lastOptions.Chrome.DeviceScaleFactor != 3 ||
			!pool.lastOptions.Chrome.IsMobile ||
			!pool.lastOptions.Chrome.HasTouch {
			t.Fatalf("mobile emulation options = %+v", pool.lastOptions.Chrome)
		}

		if _, err := client.ScreenshotWithTouchEmulation("https://example.com", false, NewScreenshotOptions(
			WithMobileEmulation(2),
		)); err != nil {
			t.Fatalf("ScreenshotWithTouchEmulation() error = %v", err)
		}
		if pool.lastOptions.Chrome.HasTouch || !pool.lastOptions.Chrome.IsMobile {
			t.Fatalf("touch emulation options = %+v", pool.lastOptions.Chrome)
		}

		if _, _, err := client.ScreenshotWithIgnoreCertErrorsBytes("https://example.com", nil); err != nil {
			t.Fatalf("ScreenshotWithIgnoreCertErrorsBytes() error = %v", err)
		}
		if !pool.lastOptions.Chrome.IgnoreCertErrors ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("ignore cert bytes options = %+v/%+v", pool.lastOptions.Chrome, pool.lastOptions.Scan)
		}

		if _, err := client.ScreenshotWithPlugins("https://example.com", []string{"Chrome PDF Viewer", "Native Client"}, nil); err != nil {
			t.Fatalf("ScreenshotWithPlugins() error = %v", err)
		}
		if len(pool.lastOptions.Chrome.Plugins) != 2 ||
			pool.lastOptions.Chrome.Plugins[0] != "Chrome PDF Viewer" {
			t.Fatalf("plugins = %+v", pool.lastOptions.Chrome.Plugins)
		}

		if _, err := client.ScreenshotWithDisabledWebRTC("https://example.com", nil); err != nil {
			t.Fatalf("ScreenshotWithDisabledWebRTC() error = %v", err)
		}
		if !pool.lastOptions.Chrome.DisableWebRTC {
			t.Fatalf("DisableWebRTC = %t", pool.lastOptions.Chrome.DisableWebRTC)
		}

		if _, _, err := client.ScreenshotWithSpoofedScreenBytes("https://example.com", 1920, 1080, nil); err != nil {
			t.Fatalf("ScreenshotWithSpoofedScreenBytes() error = %v", err)
		}
		if !pool.lastOptions.Chrome.SpoofScreenSize ||
			pool.lastOptions.Chrome.ScreenWidth != 1920 ||
			pool.lastOptions.Chrome.ScreenHeight != 1080 ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("spoofed screen bytes options = %+v/%+v", pool.lastOptions.Chrome, pool.lastOptions.Scan)
		}

		if _, err := client.ScreenshotWithCookieStrings("https://example.com/path", []string{
			"sid=abc",
			"theme=dark; lang=zh",
		}, nil); err != nil {
			t.Fatalf("ScreenshotWithCookieStrings() error = %v", err)
		}
		if len(pool.lastOptions.Scan.Cookies) != 3 ||
			pool.lastOptions.Scan.Cookies[0].Name != "sid" ||
			pool.lastOptions.Scan.Cookies[0].Domain != "example.com" ||
			pool.lastOptions.Scan.Cookies[2].Name != "lang" {
			t.Fatalf("cookie strings cookies = %+v", pool.lastOptions.Scan.Cookies)
		}

		if _, _, err := client.ScreenshotWithDefaultBlacklistBytes("https://example.com", NewScreenshotOptions(
			WithNoBlacklist(),
		)); err != nil {
			t.Fatalf("ScreenshotWithDefaultBlacklistBytes() error = %v", err)
		}
		if !pool.lastOptions.Scan.EnableBlacklist || !pool.lastOptions.Scan.DefaultBlacklist ||
			!pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
			t.Fatalf("default blacklist bytes options = %+v", pool.lastOptions.Scan)
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

func TestEvidenceBundleCapture_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{
		URL:             "https://example.com",
		Title:           "Example",
		HTML:            "<html><body>ok</body></html>",
		ScreenshotBytes: []byte("png"),
		Headers:         []models.Header{{Name: "Content-Type", Value: "text/html"}},
		Cookies:         []models.Cookie{{Name: "session", Value: "abc123"}},
		Console:         []models.ConsoleLog{{Level: "error", Message: "boom"}},
		Network:         []models.NetworkLog{{URL: "https://example.com/api", StatusCode: 200}},
	}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	dir := filepath.Join(t.TempDir(), "bundle")
	bundle, result, err := client.CaptureEvidenceBundle("https://example.com", dir, WithFullPage())
	if err != nil {
		t.Fatalf("CaptureEvidenceBundle() error = %v", err)
	}
	if result == nil || result.Title != "Example" {
		t.Fatalf("result = %+v", result)
	}
	if bundle == nil || bundle.Dir != dir {
		t.Fatalf("bundle = %+v, want dir %q", bundle, dir)
	}
	for _, path := range []string{bundle.ManifestJSON, bundle.ResultJSON, bundle.SummaryJSON, bundle.HTML, bundle.Screenshot} {
		if path == "" {
			t.Fatalf("bundle returned empty path: %+v", bundle)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected bundle file %q: %v", path, err)
		}
	}
	if !pool.lastOptions.Scan.SaveHTML || !pool.lastOptions.Scan.SaveHeaders ||
		!pool.lastOptions.Scan.SaveConsole || !pool.lastOptions.Scan.SaveCookies ||
		!pool.lastOptions.Scan.SaveNetwork {
		t.Fatalf("evidence flags = %+v", pool.lastOptions.Scan)
	}
	if !pool.lastOptions.Scan.CaptureFullPage {
		t.Fatal("CaptureEvidenceBundle() did not keep functional options")
	}
	if !pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
		t.Fatalf("byte options = %+v", pool.lastOptions.Scan)
	}

	shot, err := os.ReadFile(bundle.Screenshot)
	if err != nil {
		t.Fatalf("ReadFile(screenshot) error = %v", err)
	}
	if string(shot) != "png" {
		t.Fatalf("bundle screenshot = %q", shot)
	}
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

func TestBatchScreenshotBytes_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	results := client.BatchScreenshotBytes([]string{"https://a.test", "https://b.test"}, NewScreenshotOptions(
		WithFullPage(),
	))
	if len(results) != 2 {
		t.Fatalf("BatchScreenshotBytes() len = %d, want 2", len(results))
	}
	for _, result := range results {
		if result.Error != nil || string(result.Data) != "png" || result.Result == nil {
			t.Fatalf("BatchScreenshotBytes() result = %+v", result)
		}
	}
	if pool.calls != 2 {
		t.Fatalf("pool calls = %d, want 2", pool.calls)
	}
	if !pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
		t.Fatalf("byte options = %+v", pool.lastOptions.Scan)
	}
	if !pool.lastOptions.Scan.CaptureFullPage {
		t.Fatal("BatchScreenshotBytes() did not keep screenshot options")
	}
}

func TestBatchScreenshotBytesStreaming_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &Client{pool: &fakeDriverPool{}, opts: DefaultClientOptions()}
	ch := client.BatchScreenshotBytesStreaming(ctx, []string{"https://a.test", "https://b.test"}, nil)

	count := 0
	for result := range ch {
		count++
		if !errors.Is(result.Error, context.Canceled) {
			t.Fatalf("BatchScreenshotBytesStreaming() error = %v, want context.Canceled", result.Error)
		}
		if result.Result != nil || result.Data != nil {
			t.Fatalf("BatchScreenshotBytesStreaming() result = %+v, want no data/result", result)
		}
	}
	if count != 2 {
		t.Fatalf("BatchScreenshotBytesStreaming() count = %d, want 2", count)
	}
}

func TestBatchScreenshotRequests_Unit(t *testing.T) {
	pool := &fakeDriverPool{}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	requests := []ScreenshotRequest{
		{
			Name:    "desktop-full",
			URL:     "https://a.test",
			Options: NewScreenshotOptions(WithViewport(1440, 900), WithFullPage()),
		},
		{
			Name:    "mobile-element",
			URL:     "https://b.test",
			Options: NewScreenshotOptions(WithDevice("iphone-15"), WithElement("#hero")),
		},
	}

	results := client.BatchScreenshotRequests(requests)
	if len(results) != 2 {
		t.Fatalf("BatchScreenshotRequests() len = %d, want 2", len(results))
	}
	if results[0].Name != "desktop-full" || results[0].URL != "https://a.test" || results[0].Error != nil {
		t.Fatalf("result[0] = %+v", results[0])
	}
	if results[1].Name != "mobile-element" || results[1].URL != "https://b.test" || results[1].Error != nil {
		t.Fatalf("result[1] = %+v", results[1])
	}
	if pool.calls != 2 {
		t.Fatalf("pool calls = %d, want 2", pool.calls)
	}

	desktop := pool.optionsByURL["https://a.test"]
	if desktop.Chrome.WindowX != 1440 || desktop.Chrome.WindowY != 900 || !desktop.Scan.CaptureFullPage {
		t.Fatalf("desktop options = %+v/%+v", desktop.Chrome, desktop.Scan)
	}
	mobile := pool.optionsByURL["https://b.test"]
	if mobile.Chrome.DeviceName != "iPhone 15" || mobile.Scan.Selector != "#hero" {
		t.Fatalf("mobile options = %+v/%+v", mobile.Chrome, mobile.Scan)
	}
}

func TestBatchScreenshotRequestsBytes_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	results := client.BatchScreenshotRequestsBytes([]ScreenshotRequest{
		{Name: "jpeg", URL: "https://a.test", Options: NewScreenshotOptions(WithFormat("jpeg", 80))},
		{Name: "html", URL: "https://b.test", Options: NewScreenshotOptions(WithEvidence())},
	})
	if len(results) != 2 {
		t.Fatalf("BatchScreenshotRequestsBytes() len = %d, want 2", len(results))
	}
	if results[0].Name != "jpeg" || string(results[0].Data) != "png" || results[0].Error != nil {
		t.Fatalf("result[0] = %+v", results[0])
	}
	if results[1].Name != "html" || string(results[1].Data) != "png" || results[1].Error != nil {
		t.Fatalf("result[1] = %+v", results[1])
	}

	jpeg := pool.optionsByURL["https://a.test"]
	if jpeg.Scan.ScreenshotFormat != "jpeg" || jpeg.Scan.ScreenshotQuality != 80 ||
		!jpeg.Scan.ReturnScreenshotBytes || !jpeg.Scan.ScreenshotSkipSave {
		t.Fatalf("jpeg options = %+v", jpeg.Scan)
	}
	evidence := pool.optionsByURL["https://b.test"]
	if !evidence.Scan.SaveHTML || !evidence.Scan.SaveHeaders || !evidence.Scan.SaveConsole ||
		!evidence.Scan.SaveCookies || !evidence.Scan.SaveNetwork {
		t.Fatalf("evidence options = %+v", evidence.Scan)
	}
}

func TestBatchScreenshotEvidenceBundles_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{
		HTML:            "<html></html>",
		ScreenshotBytes: []byte("png"),
	}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}
	dir := t.TempDir()

	results := client.BatchScreenshotEvidenceBundles([]string{"https://a.test/path", "https://b.test"}, dir, NewScreenshotOptions(
		WithFullPage(),
	))
	if len(results) != 2 {
		t.Fatalf("BatchScreenshotEvidenceBundles() len = %d, want 2", len(results))
	}
	for _, result := range results {
		if result.Error != nil || result.Bundle == nil || result.Result == nil {
			t.Fatalf("BatchScreenshotEvidenceBundles() result = %+v", result)
		}
		for _, path := range []string{result.Bundle.ManifestJSON, result.Bundle.ResultJSON, result.Bundle.SummaryJSON, result.Bundle.HTML, result.Bundle.Screenshot} {
			if path == "" {
				t.Fatalf("BatchScreenshotEvidenceBundles() returned empty bundle path: %+v", result.Bundle)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("bundle file %s: %v", path, err)
			}
		}
	}
	if filepath.Base(results[0].Dir) != "001_https___a.test_path" {
		t.Fatalf("result[0].Dir = %q", results[0].Dir)
	}
	if pool.calls != 2 {
		t.Fatalf("pool calls = %d, want 2", pool.calls)
	}
	got := pool.optionsByURL["https://a.test/path"]
	if !got.Scan.SaveHTML || !got.Scan.SaveHeaders || !got.Scan.SaveConsole ||
		!got.Scan.SaveCookies || !got.Scan.SaveNetwork ||
		!got.Scan.ReturnScreenshotBytes || !got.Scan.ScreenshotSkipSave ||
		!got.Scan.CaptureFullPage {
		t.Fatalf("bundle options = %+v", got.Scan)
	}
}

func TestBatchScreenshotRequestsEvidenceBundles_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{
		HTML:            "<html></html>",
		ScreenshotBytes: []byte("png"),
	}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}
	dir := t.TempDir()

	results := client.BatchScreenshotRequestsEvidenceBundles([]ScreenshotRequest{
		{Name: "desktop:full", URL: "https://a.test", Options: NewScreenshotOptions(WithViewport(1440, 900), WithFullPage())},
		{Name: "mobile/hero", URL: "https://b.test", Options: NewScreenshotOptions(WithDevice("iphone-15"), WithElement("#hero"))},
	}, dir)
	if len(results) != 2 {
		t.Fatalf("BatchScreenshotRequestsEvidenceBundles() len = %d, want 2", len(results))
	}
	if results[0].Name != "desktop:full" || filepath.Base(results[0].Dir) != "001_desktop_full" || results[0].Bundle == nil || results[0].Error != nil {
		t.Fatalf("result[0] = %+v", results[0])
	}
	if results[1].Name != "mobile/hero" || filepath.Base(results[1].Dir) != "002_mobile_hero" || results[1].Bundle == nil || results[1].Error != nil {
		t.Fatalf("result[1] = %+v", results[1])
	}

	desktop := pool.optionsByURL["https://a.test"]
	if desktop.Chrome.WindowX != 1440 || desktop.Chrome.WindowY != 900 || !desktop.Scan.CaptureFullPage {
		t.Fatalf("desktop options = %+v/%+v", desktop.Chrome, desktop.Scan)
	}
	mobile := pool.optionsByURL["https://b.test"]
	if mobile.Chrome.DeviceName != "iPhone 15" || mobile.Scan.Selector != "#hero" {
		t.Fatalf("mobile options = %+v/%+v", mobile.Chrome, mobile.Scan)
	}
}

func TestBatchScreenshotEvidenceBundlesStreaming_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pool := &fakeDriverPool{}
	client := &Client{pool: pool, opts: DefaultClientOptions()}
	ch := client.BatchScreenshotEvidenceBundlesStreaming(ctx, []string{"https://a.test", "https://b.test"}, t.TempDir(), nil)

	count := 0
	for result := range ch {
		count++
		if !errors.Is(result.Error, context.Canceled) {
			t.Fatalf("BatchScreenshotEvidenceBundlesStreaming() error = %v, want context.Canceled", result.Error)
		}
		if result.Bundle != nil || result.Result != nil || result.Dir == "" {
			t.Fatalf("BatchScreenshotEvidenceBundlesStreaming() result = %+v, want dir only", result)
		}
	}
	if count != 2 {
		t.Fatalf("BatchScreenshotEvidenceBundlesStreaming() count = %d, want 2", count)
	}
	if pool.calls != 0 {
		t.Fatalf("pool calls = %d, want 0", pool.calls)
	}
}

func TestBatchScreenshotTargetsEvidenceBundlesCallback_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{
		HTML:            "<html></html>",
		ScreenshotBytes: []byte("png"),
	}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	seen := map[string]string{}
	client.BatchScreenshotTargetsEvidenceBundlesCallback(context.Background(), []string{"example.com/admin"}, t.TempDir(), NewScreenshotOptions(
		WithHTTPOnly(),
		WithPorts(8080),
	), func(result BatchEvidenceBundleResult) {
		if result.Error != nil {
			t.Fatalf("callback result = %+v", result)
		}
		seen[result.URL] = result.Dir
	})

	if len(seen) != 1 || seen["http://example.com:8080/admin"] == "" {
		t.Fatalf("callback seen = %#v", seen)
	}
	if pool.calls != 1 {
		t.Fatalf("pool calls = %d, want 1", pool.calls)
	}
}

func TestBatchScreenshotRequestsStreaming_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &Client{pool: &fakeDriverPool{}, opts: DefaultClientOptions()}
	ch := client.BatchScreenshotRequestsStreaming(ctx, []ScreenshotRequest{
		{Name: "one", URL: "https://a.test"},
		{Name: "two", URL: "https://b.test"},
	})

	seen := map[string]error{}
	for result := range ch {
		seen[result.Name] = result.Error
		if result.Result != nil {
			t.Fatalf("BatchScreenshotRequestsStreaming() result = %+v, want no result", result)
		}
	}
	if !errors.Is(seen["one"], context.Canceled) || !errors.Is(seen["two"], context.Canceled) {
		t.Fatalf("seen = %#v, want canceled errors", seen)
	}
}

func TestBatchScreenshotRequestsBytesCallback_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	seen := map[string]string{}
	client.BatchScreenshotRequestsBytesCallback(context.Background(), []ScreenshotRequest{
		{Name: "mobile", URL: "https://a.test", Options: NewScreenshotOptions(WithDevice("pixel-8"))},
	}, func(result BatchBytesResult) {
		seen[result.Name] = string(result.Data)
	})

	if seen["mobile"] != "png" || pool.calls != 1 {
		t.Fatalf("callback seen/pool calls = %#v/%d", seen, pool.calls)
	}
	if pool.optionsByURL["https://a.test"].Chrome.DeviceName != "Pixel 8" {
		t.Fatalf("request options = %+v", pool.optionsByURL["https://a.test"].Chrome)
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

func TestBatchScreenshotTargetsBytes_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	results := client.BatchScreenshotTargetsBytes([]string{"example.com/path", "https://already.test/login"}, NewScreenshotOptions(
		WithHTTPOnly(),
		WithPorts(80, 8080),
	))
	wantURLs := []string{
		"http://example.com:80/path",
		"http://example.com:8080/path",
		"https://already.test/login",
	}
	if len(results) != len(wantURLs) {
		t.Fatalf("BatchScreenshotTargetsBytes() len = %d, want %d", len(results), len(wantURLs))
	}
	for i, want := range wantURLs {
		if results[i].URL != want || string(results[i].Data) != "png" ||
			results[i].Result == nil || results[i].Error != nil {
			t.Fatalf("result[%d] = %+v, want URL %q with bytes", i, results[i], want)
		}
	}
	if pool.calls != len(wantURLs) {
		t.Fatalf("pool calls = %d, want %d", pool.calls, len(wantURLs))
	}
	if !pool.lastOptions.Scan.ReturnScreenshotBytes || !pool.lastOptions.Scan.ScreenshotSkipSave {
		t.Fatalf("byte options = %+v", pool.lastOptions.Scan)
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

func TestBatchScreenshotTargetsBytesCallback_Unit(t *testing.T) {
	pool := &fakeDriverPool{result: &models.Result{ScreenshotBytes: []byte("png")}}
	client := &Client{pool: pool, opts: DefaultClientOptions()}

	seen := map[string]string{}
	client.BatchScreenshotTargetsBytesCallback(context.Background(), []string{"example.com"}, NewScreenshotOptions(
		WithHTTPSOnly(),
		WithPorts(443),
	), func(result BatchBytesResult) {
		seen[result.URL] = string(result.Data)
	})

	if seen["https://example.com:443"] != "png" || pool.calls != 1 {
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

	t.Run("capture applies functional options", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if target != "https://example.com" {
				t.Fatalf("target = %q", target)
			}
			if !opts.Scan.CaptureFullPage || !opts.Scan.SaveHTML {
				t.Fatalf("functional options were not merged: %+v", opts.Scan)
			}
			return &models.Result{URL: target, Title: "capture"}, nil
		}
		result, err := SharedCapture("https://example.com", WithFullPage(), WithHTML())
		if err != nil {
			t.Fatalf("SharedCapture() error = %v", err)
		}
		if result.Title != "capture" {
			t.Fatalf("SharedCapture() title = %q, want capture", result.Title)
		}
	})

	t.Run("bytes sets in-memory flags and returns data", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if !opts.Scan.ReturnScreenshotBytes {
				t.Fatal("ReturnScreenshotBytes was not enabled")
			}
			if !opts.Scan.ScreenshotSkipSave {
				t.Fatal("ScreenshotSkipSave was not enabled")
			}
			if !opts.Scan.CaptureFullPage {
				t.Fatal("functional option was not merged")
			}
			return &models.Result{URL: target, ScreenshotBytes: []byte("png")}, nil
		}
		data, result, err := SharedCaptureBytes("https://example.com", WithFullPage())
		if err != nil {
			t.Fatalf("SharedCaptureBytes() error = %v", err)
		}
		if string(data) != "png" || result == nil || result.URL != "https://example.com" {
			t.Fatalf("SharedCaptureBytes() data/result = %q/%+v", data, result)
		}
	})

	t.Run("bytes failed result returns error", func(t *testing.T) {
		restoreSDKHooks(t)
		failed := &models.Result{Failed: true, FailedReason: "blocked"}
		sharedScreenshotWithContext = func(context.Context, string, *runner.Options) (*models.Result, error) {
			return failed, nil
		}
		data, result, err := SharedScreenshotBytesWithContext(context.Background(), "https://example.com", nil)
		if data != nil || result != failed || err == nil || !strings.Contains(err.Error(), "blocked") {
			t.Fatalf("SharedScreenshotBytesWithContext() data/result/error = %v/%v/%v", data, result, err)
		}
	})

	t.Run("bytes blacklisted target skips shared pool", func(t *testing.T) {
		restoreSDKHooks(t)
		called := false
		sharedScreenshotWithContext = func(context.Context, string, *runner.Options) (*models.Result, error) {
			called = true
			return &models.Result{Title: "unexpected"}, nil
		}
		data, result, err := SharedScreenshotBytesWithContext(context.Background(), "https://example.com", NewScreenshotOptions(
			WithBlacklist("example.com"),
		))
		if data != nil || result == nil || !result.Failed || err == nil {
			t.Fatalf("SharedScreenshotBytesWithContext() data/result/error = %v/%+v/%v", data, result, err)
		}
		if called {
			t.Fatal("sharedScreenshotWithContext was called for a blacklisted target")
		}
	})

	t.Run("scenario helpers map options", func(t *testing.T) {
		restoreSDKHooks(t)
		var last runner.Options
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			last = *opts
			return &models.Result{URL: target, ScreenshotBytes: []byte("png")}, nil
		}

		if _, err := SharedScreenshotElement("https://example.com", "#hero", nil); err != nil {
			t.Fatalf("SharedScreenshotElement() error = %v", err)
		}
		if last.Scan.Selector != "#hero" {
			t.Fatalf("Selector = %q, want #hero", last.Scan.Selector)
		}

		if _, err := SharedScreenshotXPath("https://example.com", "//main", nil); err != nil {
			t.Fatalf("SharedScreenshotXPath() error = %v", err)
		}
		if last.Scan.XPath != "//main" {
			t.Fatalf("XPath = %q, want //main", last.Scan.XPath)
		}

		if _, err := SharedScreenshotFullPage("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotFullPage() error = %v", err)
		}
		if !last.Scan.CaptureFullPage {
			t.Fatal("CaptureFullPage was not enabled")
		}

		if _, err := SharedScreenshotDevice("https://example.com", "pixel-8-pro", nil); err != nil {
			t.Fatalf("SharedScreenshotDevice() error = %v", err)
		}
		if last.Chrome.DeviceName != "Pixel 8 Pro" {
			t.Fatalf("DeviceName = %q, want Pixel 8 Pro", last.Chrome.DeviceName)
		}

		if _, err := SharedScreenshotViewport("https://example.com", 390, 844, nil); err != nil {
			t.Fatalf("SharedScreenshotViewport() error = %v", err)
		}
		if last.Chrome.WindowX != 390 || last.Chrome.WindowY != 844 {
			t.Fatalf("viewport = %dx%d, want 390x844", last.Chrome.WindowX, last.Chrome.WindowY)
		}

		if _, err := SharedScreenshotWithJS("https://example.com", "document.body.dataset.ready='1'", nil); err != nil {
			t.Fatalf("SharedScreenshotWithJS() error = %v", err)
		}
		if last.Scan.JavaScript != "document.body.dataset.ready='1'" ||
			last.Scan.RunJSBefore || !last.Scan.RunJSAfter {
			t.Fatalf("after-load JS options = %+v", last.Scan)
		}

		if _, err := SharedScreenshotWithJSBefore("https://example.com", "window.preload=true", nil); err != nil {
			t.Fatalf("SharedScreenshotWithJSBefore() error = %v", err)
		}
		if last.Scan.JavaScript != "window.preload=true" ||
			!last.Scan.RunJSBefore || last.Scan.RunJSAfter {
			t.Fatalf("before-load JS options = %+v", last.Scan)
		}

		if _, err := SharedScreenshotWithJSFile("https://example.com", "preload.js", true, nil); err != nil {
			t.Fatalf("SharedScreenshotWithJSFile() before-load error = %v", err)
		}
		if last.Scan.JavaScriptFile != "preload.js" ||
			!last.Scan.RunJSBefore || last.Scan.RunJSAfter {
			t.Fatalf("before-load JS file options = %+v", last.Scan)
		}

		if _, err := SharedScreenshotWithJSFile("https://example.com", "after.js", false, nil); err != nil {
			t.Fatalf("SharedScreenshotWithJSFile() after-load error = %v", err)
		}
		if last.Scan.JavaScriptFile != "after.js" ||
			last.Scan.RunJSBefore || !last.Scan.RunJSAfter {
			t.Fatalf("after-load JS file options = %+v", last.Scan)
		}
	})

	t.Run("scenario bytes helpers map options", func(t *testing.T) {
		restoreSDKHooks(t)
		var last runner.Options
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			last = *opts
			return &models.Result{URL: target, ScreenshotBytes: []byte("png")}, nil
		}
		requireBytesFlags := func(name string) {
			t.Helper()
			if !last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
				t.Fatalf("%s did not enable byte flags: %+v", name, last.Scan)
			}
		}

		data, _, err := SharedScreenshotElementBytes("https://example.com", "#hero", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotElementBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.Selector != "#hero" {
			t.Fatalf("element bytes data/options = %q/%+v", data, last.Scan)
		}
		requireBytesFlags("SharedScreenshotElementBytes")

		if data, _, err = SharedScreenshotXPathBytes("https://example.com", "//main", nil); err != nil {
			t.Fatalf("SharedScreenshotXPathBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.XPath != "//main" {
			t.Fatalf("xpath bytes data/options = %q/%+v", data, last.Scan)
		}
		requireBytesFlags("SharedScreenshotXPathBytes")

		if data, _, err = SharedScreenshotFullPageBytes("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotFullPageBytes() error = %v", err)
		}
		if string(data) != "png" || !last.Scan.CaptureFullPage {
			t.Fatalf("full-page bytes data/options = %q/%+v", data, last.Scan)
		}
		requireBytesFlags("SharedScreenshotFullPageBytes")

		if data, _, err = SharedScreenshotDeviceBytes("https://example.com", "pixel-8-pro", nil); err != nil {
			t.Fatalf("SharedScreenshotDeviceBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.DeviceName != "Pixel 8 Pro" {
			t.Fatalf("device bytes data/options = %q/%+v", data, last.Chrome)
		}
		requireBytesFlags("SharedScreenshotDeviceBytes")

		if data, _, err = SharedScreenshotViewportBytes("https://example.com", 390, 844, nil); err != nil {
			t.Fatalf("SharedScreenshotViewportBytes() error = %v", err)
		}
		if string(data) != "png" || last.Chrome.WindowX != 390 || last.Chrome.WindowY != 844 {
			t.Fatalf("viewport bytes data/options = %q/%+v", data, last.Chrome)
		}
		requireBytesFlags("SharedScreenshotViewportBytes")

		if data, _, err = SharedScreenshotWithJSBytes("https://example.com", "document.body.dataset.ready='1'", nil); err != nil {
			t.Fatalf("SharedScreenshotWithJSBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.JavaScript != "document.body.dataset.ready='1'" ||
			last.Scan.RunJSBefore || !last.Scan.RunJSAfter {
			t.Fatalf("after-load JS bytes data/options = %q/%+v", data, last.Scan)
		}
		requireBytesFlags("SharedScreenshotWithJSBytes")

		if data, _, err = SharedScreenshotWithJSBeforeBytes("https://example.com", "window.preload=true", nil); err != nil {
			t.Fatalf("SharedScreenshotWithJSBeforeBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.JavaScript != "window.preload=true" ||
			!last.Scan.RunJSBefore || last.Scan.RunJSAfter {
			t.Fatalf("before-load JS bytes data/options = %q/%+v", data, last.Scan)
		}
		requireBytesFlags("SharedScreenshotWithJSBeforeBytes")

		if data, _, err = SharedScreenshotWithJSFileBytes("https://example.com", "preload.js", true, nil); err != nil {
			t.Fatalf("SharedScreenshotWithJSFileBytes() before-load error = %v", err)
		}
		if string(data) != "png" || last.Scan.JavaScriptFile != "preload.js" ||
			!last.Scan.RunJSBefore || last.Scan.RunJSAfter {
			t.Fatalf("before-load JS file bytes data/options = %q/%+v", data, last.Scan)
		}
		requireBytesFlags("SharedScreenshotWithJSFileBytes before-load")

		if data, _, err = SharedScreenshotWithJSFileBytes("https://example.com", "after.js", false, nil); err != nil {
			t.Fatalf("SharedScreenshotWithJSFileBytes() after-load error = %v", err)
		}
		if string(data) != "png" || last.Scan.JavaScriptFile != "after.js" ||
			last.Scan.RunJSBefore || !last.Scan.RunJSAfter {
			t.Fatalf("after-load JS file bytes data/options = %q/%+v", data, last.Scan)
		}
		requireBytesFlags("SharedScreenshotWithJSFileBytes after-load")

		actions := []runner.InteractionAction{{Type: "click", Selector: "#accept"}}
		if data, _, err = SharedScreenshotWithActionsBytes("https://example.com", actions, nil); err != nil {
			t.Fatalf("SharedScreenshotWithActionsBytes() error = %v", err)
		}
		if string(data) != "png" || len(last.Scan.Actions) != 1 ||
			last.Scan.Actions[0].Type != "click" || last.Scan.Actions[0].Selector != "#accept" {
			t.Fatalf("actions bytes data/options = %q/%+v", data, last.Scan.Actions)
		}
		requireBytesFlags("SharedScreenshotWithActionsBytes")

		form := FormWithSubmit("#login", 2*time.Second, FormInput("#user", "admin"))
		if data, _, err = SharedScreenshotWithFormBytes("https://example.com/login", form, nil); err != nil {
			t.Fatalf("SharedScreenshotWithFormBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.Form.SubmitSelector != "#login" ||
			last.Scan.Form.WaitAfterSubmit != 2000 || len(last.Scan.Form.Fields) != 1 {
			t.Fatalf("form bytes data/options = %q/%+v", data, last.Scan.Form)
		}
		requireBytesFlags("SharedScreenshotWithFormBytes")

		cookies := []runner.CustomCookie{{Name: "session", Value: "abc", Domain: "example.com"}}
		if data, _, err = SharedScreenshotWithCookiesBytes("https://example.com/dashboard", cookies, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookiesBytes() error = %v", err)
		}
		if string(data) != "png" || len(last.Scan.Cookies) != 1 ||
			last.Scan.Cookies[0].Name != "session" {
			t.Fatalf("cookies bytes data/options = %q/%+v", data, last.Scan.Cookies)
		}
		requireBytesFlags("SharedScreenshotWithCookiesBytes")
	})

	t.Run("action form and cookie helpers map options", func(t *testing.T) {
		restoreSDKHooks(t)
		var last runner.Options
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			last = *opts
			return &models.Result{URL: target}, nil
		}

		actions := []runner.InteractionAction{{Type: "click", Selector: "#accept"}}
		if _, err := SharedScreenshotWithActions("https://example.com", actions, nil); err != nil {
			t.Fatalf("SharedScreenshotWithActions() error = %v", err)
		}
		if len(last.Scan.Actions) != 1 ||
			last.Scan.Actions[0].Type != "click" || last.Scan.Actions[0].Selector != "#accept" {
			t.Fatalf("actions = %+v", last.Scan.Actions)
		}

		form := FormWithSubmit("#login", 2*time.Second, FormInput("#user", "admin"))
		if _, err := SharedScreenshotWithForm("https://example.com/login", form, nil); err != nil {
			t.Fatalf("SharedScreenshotWithForm() error = %v", err)
		}
		if last.Scan.Form.SubmitSelector != "#login" ||
			last.Scan.Form.WaitAfterSubmit != 2000 || len(last.Scan.Form.Fields) != 1 {
			t.Fatalf("form = %+v", last.Scan.Form)
		}

		cookies := []runner.CustomCookie{{Name: "session", Value: "abc", Domain: "example.com"}}
		if _, err := SharedScreenshotWithCookies("https://example.com/dashboard", cookies, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookies() error = %v", err)
		}
		if len(last.Scan.Cookies) != 1 || last.Scan.Cookies[0].Name != "session" {
			t.Fatalf("cookies = %+v", last.Scan.Cookies)
		}
	})

	t.Run("request profile helpers map options", func(t *testing.T) {
		restoreSDKHooks(t)
		var last runner.Options
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			last = *opts
			return &models.Result{
				URL:             target,
				ScreenshotBytes: []byte("png"),
				Cookies:         []models.Cookie{{Name: "exported", Value: "1", Domain: "example.com", Path: "/"}},
			}, nil
		}

		if _, err := SharedScreenshotWithProxy("https://example.com", "http://static:8080", NewScreenshotOptions(
			WithProxyList(runner.ProxyRandom, "http://old:8080"),
		)); err != nil {
			t.Fatalf("SharedScreenshotWithProxy() error = %v", err)
		}
		if last.Chrome.Proxy != "http://static:8080" || len(last.Chrome.ProxyList) != 0 {
			t.Fatalf("proxy options = %+v", last.Chrome)
		}

		data, _, err := SharedScreenshotWithProxyListBytes("https://example.com", runner.ProxyRoundRobin, []string{
			"http://a:8080",
			"http://b:8080",
		}, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithProxyListBytes() error = %v", err)
		}
		if string(data) != "png" || len(last.Chrome.ProxyList) != 2 ||
			last.Chrome.ProxyStrategy != runner.ProxyRoundRobin ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("proxy list bytes data/options = %q/%+v/%+v", data, last.Chrome, last.Scan)
		}

		if _, err := SharedScreenshotWithProxyFile("https://example.com", "proxies.txt", runner.ProxyRandom, nil); err != nil {
			t.Fatalf("SharedScreenshotWithProxyFile() error = %v", err)
		}
		if last.Chrome.ProxyFile != "proxies.txt" || last.Chrome.ProxyStrategy != runner.ProxyRandom {
			t.Fatalf("proxy file options = %+v", last.Chrome)
		}

		if _, _, err := SharedScreenshotWithProxyURLBytes("https://example.com", "https://proxy-api.example/list", runner.ProxyRoundRobin, nil); err != nil {
			t.Fatalf("SharedScreenshotWithProxyURLBytes() error = %v", err)
		}
		if last.Chrome.ProxyURL != "https://proxy-api.example/list" ||
			last.Chrome.ProxyStrategy != runner.ProxyRoundRobin {
			t.Fatalf("proxy url options = %+v", last.Chrome)
		}

		if _, err := SharedScreenshotWithCustomHeaders("https://example.com", map[string]string{"X-Test": "1"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCustomHeaders() error = %v", err)
		}
		if last.Chrome.CustomHeaders["X-Test"] != "1" {
			t.Fatalf("CustomHeaders = %+v", last.Chrome.CustomHeaders)
		}

		if _, err := SharedScreenshotWithUserAgent("https://example.com", "snir-test-agent", nil); err != nil {
			t.Fatalf("SharedScreenshotWithUserAgent() error = %v", err)
		}
		if last.Chrome.UserAgent != "snir-test-agent" {
			t.Fatalf("UserAgent = %q", last.Chrome.UserAgent)
		}

		if _, err := SharedScreenshotWithAcceptLanguage("https://example.com", "zh-CN,zh;q=0.9", nil); err != nil {
			t.Fatalf("SharedScreenshotWithAcceptLanguage() error = %v", err)
		}
		if last.Chrome.AcceptLanguage != "zh-CN,zh;q=0.9" {
			t.Fatalf("AcceptLanguage = %q", last.Chrome.AcceptLanguage)
		}

		if _, err := SharedScreenshotWithFingerprint("https://example.com", "Linux x86_64", "Google Inc.", "Mesa", "llvmpipe", nil); err != nil {
			t.Fatalf("SharedScreenshotWithFingerprint() error = %v", err)
		}
		if last.Chrome.Platform != "Linux x86_64" ||
			last.Chrome.Vendor != "Google Inc." ||
			last.Chrome.WebGLVendor != "Mesa" ||
			last.Chrome.WebGLRenderer != "llvmpipe" {
			t.Fatalf("fingerprint options = %+v", last.Chrome)
		}

		if _, err := SharedScreenshotWithCookieHeader("https://example.com/path", "sid=abc; theme=dark", nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieHeader() error = %v", err)
		}
		if len(last.Scan.Cookies) != 2 ||
			last.Scan.Cookies[0].Name != "sid" ||
			last.Scan.Cookies[0].Domain != "example.com" {
			t.Fatalf("cookie header cookies = %+v", last.Scan.Cookies)
		}

		jarPath := filepath.Join(t.TempDir(), "cookies.json")
		if _, _, err := SharedScreenshotWithCookieFileBytes("https://example.com", jarPath, true, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieFileBytes() error = %v", err)
		}
		if last.Scan.CookiesFile != jarPath || !last.Scan.CookieWriteBack ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("cookie file bytes options = %+v", last.Scan)
		}

		importFile := filepath.Join(t.TempDir(), "cookies.txt")
		importContent := "# Netscape HTTP Cookie File\n.example.com\tTRUE\t/\tFALSE\t0\timported\tyes\n"
		if err := os.WriteFile(importFile, []byte(importContent), 0644); err != nil {
			t.Fatalf("write import cookie file: %v", err)
		}
		if _, err := SharedScreenshotWithCookieImport("https://example.com", importFile, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieImport() error = %v", err)
		}
		if last.Scan.CookieImport != importFile ||
			len(last.Scan.Cookies) != 1 ||
			last.Scan.Cookies[0].Name != "imported" {
			t.Fatalf("cookie import options = %+v", last.Scan)
		}

		exportFile := filepath.Join(t.TempDir(), "export.txt")
		if data, _, err = SharedScreenshotWithCookieExportBytes("https://example.com", exportFile, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieExportBytes() error = %v", err)
		}
		if string(data) != "png" || last.Scan.CookieExport != exportFile || !last.Scan.SaveCookies ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("cookie export bytes data/options = %q/%+v", data, last.Scan)
		}
		exported, err := runner.LoadNetscapeCookieFile(exportFile)
		if err != nil {
			t.Fatalf("LoadNetscapeCookieFile() error = %v", err)
		}
		if len(exported) != 1 || exported[0].Name != "exported" {
			t.Fatalf("exported cookies = %+v", exported)
		}

		if _, err := SharedScreenshotWithBlacklist("https://example.com", []string{"blocked.example"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithBlacklist() error = %v", err)
		}
		if !last.Scan.EnableBlacklist || last.Scan.DefaultBlacklist ||
			len(last.Scan.BlacklistPatterns) != 1 ||
			last.Scan.BlacklistPatterns[0] != "blocked.example" {
			t.Fatalf("blacklist options = %+v", last.Scan)
		}

		blacklistFile := filepath.Join(t.TempDir(), "blacklist.txt")
		if err := os.WriteFile(blacklistFile, []byte("blocked.example\n"), 0644); err != nil {
			t.Fatalf("write blacklist file: %v", err)
		}
		if _, _, err := SharedScreenshotWithBlacklistFileBytes("https://example.com", blacklistFile, nil); err != nil {
			t.Fatalf("SharedScreenshotWithBlacklistFileBytes() error = %v", err)
		}
		if !last.Scan.EnableBlacklist || last.Scan.BlacklistFile != blacklistFile ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("blacklist file bytes options = %+v", last.Scan)
		}

		blockedOpts := NewScreenshotOptions(WithBlacklist("example.com"))
		if _, err := SharedScreenshotWithoutBlacklist("https://example.com", blockedOpts); err != nil {
			t.Fatalf("SharedScreenshotWithoutBlacklist() error = %v", err)
		}
		if last.Scan.EnableBlacklist || last.Scan.DefaultBlacklist ||
			len(last.Scan.BlacklistPatterns) != 0 || last.Scan.BlacklistFile != "" {
			t.Fatalf("without blacklist options = %+v", last.Scan)
		}

		if _, _, err := SharedScreenshotWithRetriesBytes("https://example.com", 4, nil); err != nil {
			t.Fatalf("SharedScreenshotWithRetriesBytes() error = %v", err)
		}
		if last.Scan.MaxRetries != 4 ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("retries bytes options = %+v", last.Scan)
		}
	})

	t.Run("browser environment helpers map options", func(t *testing.T) {
		restoreSDKHooks(t)
		var last runner.Options
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			last = *opts
			return &models.Result{URL: target, ScreenshotBytes: []byte("png")}, nil
		}

		data, _, err := SharedScreenshotWithDeviceEmulationBytes("https://example.com", 412, 915, 2.75, true, true, nil)
		if err != nil {
			t.Fatalf("SharedScreenshotWithDeviceEmulationBytes() error = %v", err)
		}
		if string(data) != "png" ||
			last.Chrome.WindowX != 412 ||
			last.Chrome.WindowY != 915 ||
			last.Chrome.DeviceScaleFactor != 2.75 ||
			!last.Chrome.IsMobile ||
			!last.Chrome.HasTouch ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("device emulation bytes data/options = %q/%+v/%+v", data, last.Chrome, last.Scan)
		}

		if _, err := SharedScreenshotWithMobileEmulation("https://example.com", 3, nil); err != nil {
			t.Fatalf("SharedScreenshotWithMobileEmulation() error = %v", err)
		}
		if last.Chrome.DeviceScaleFactor != 3 || !last.Chrome.IsMobile || !last.Chrome.HasTouch {
			t.Fatalf("mobile emulation options = %+v", last.Chrome)
		}

		if _, err := SharedScreenshotWithTouchEmulation("https://example.com", false, NewScreenshotOptions(
			WithMobileEmulation(2),
		)); err != nil {
			t.Fatalf("SharedScreenshotWithTouchEmulation() error = %v", err)
		}
		if last.Chrome.HasTouch || !last.Chrome.IsMobile {
			t.Fatalf("touch emulation options = %+v", last.Chrome)
		}

		if _, _, err := SharedScreenshotWithIgnoreCertErrorsBytes("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotWithIgnoreCertErrorsBytes() error = %v", err)
		}
		if !last.Chrome.IgnoreCertErrors ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("ignore cert bytes options = %+v/%+v", last.Chrome, last.Scan)
		}

		if _, err := SharedScreenshotWithPlugins("https://example.com", []string{"Chrome PDF Viewer", "Native Client"}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithPlugins() error = %v", err)
		}
		if len(last.Chrome.Plugins) != 2 || last.Chrome.Plugins[0] != "Chrome PDF Viewer" {
			t.Fatalf("plugins = %+v", last.Chrome.Plugins)
		}

		if _, err := SharedScreenshotWithDisabledWebRTC("https://example.com", nil); err != nil {
			t.Fatalf("SharedScreenshotWithDisabledWebRTC() error = %v", err)
		}
		if !last.Chrome.DisableWebRTC {
			t.Fatalf("DisableWebRTC = %t", last.Chrome.DisableWebRTC)
		}

		if _, _, err := SharedScreenshotWithSpoofedScreenBytes("https://example.com", 1920, 1080, nil); err != nil {
			t.Fatalf("SharedScreenshotWithSpoofedScreenBytes() error = %v", err)
		}
		if !last.Chrome.SpoofScreenSize ||
			last.Chrome.ScreenWidth != 1920 ||
			last.Chrome.ScreenHeight != 1080 ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("spoofed screen bytes options = %+v/%+v", last.Chrome, last.Scan)
		}

		if _, err := SharedScreenshotWithCookieStrings("https://example.com/path", []string{
			"sid=abc",
			"theme=dark; lang=zh",
		}, nil); err != nil {
			t.Fatalf("SharedScreenshotWithCookieStrings() error = %v", err)
		}
		if len(last.Scan.Cookies) != 3 ||
			last.Scan.Cookies[0].Name != "sid" ||
			last.Scan.Cookies[0].Domain != "example.com" ||
			last.Scan.Cookies[2].Name != "lang" {
			t.Fatalf("cookie strings cookies = %+v", last.Scan.Cookies)
		}

		if _, _, err := SharedScreenshotWithDefaultBlacklistBytes("https://example.com", NewScreenshotOptions(
			WithNoBlacklist(),
		)); err != nil {
			t.Fatalf("SharedScreenshotWithDefaultBlacklistBytes() error = %v", err)
		}
		if !last.Scan.EnableBlacklist || !last.Scan.DefaultBlacklist ||
			!last.Scan.ReturnScreenshotBytes || !last.Scan.ScreenshotSkipSave {
			t.Fatalf("default blacklist bytes options = %+v", last.Scan)
		}
	})

	t.Run("batch screenshot uses shared pool", func(t *testing.T) {
		restoreSDKHooks(t)
		var mu sync.Mutex
		optionsByURL := map[string]runner.Options{}
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			mu.Lock()
			optionsByURL[target] = *opts
			mu.Unlock()
			return &models.Result{URL: target, Title: "ok"}, nil
		}

		results := SharedBatchScreenshot([]string{"https://a.example", "https://b.example"}, NewScreenshotOptions(
			WithFullPage(),
		))
		if len(results) != 2 {
			t.Fatalf("SharedBatchScreenshot() len = %d, want 2", len(results))
		}
		if results[0].URL != "https://a.example" || results[0].Error != nil ||
			results[0].Result == nil || results[0].Result.Title != "ok" {
			t.Fatalf("first batch result = %+v", results[0])
		}
		mu.Lock()
		firstOpts := optionsByURL["https://a.example"]
		secondOpts := optionsByURL["https://b.example"]
		mu.Unlock()
		if !firstOpts.Scan.CaptureFullPage || !secondOpts.Scan.CaptureFullPage {
			t.Fatalf("batch options were not merged: %+v / %+v", firstOpts.Scan, secondOpts.Scan)
		}
	})

	t.Run("batch requests bytes preserve names and options", func(t *testing.T) {
		restoreSDKHooks(t)
		var mu sync.Mutex
		optionsByURL := map[string]runner.Options{}
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			mu.Lock()
			optionsByURL[target] = *opts
			mu.Unlock()
			return &models.Result{URL: target, ScreenshotBytes: []byte(target)}, nil
		}

		results := SharedBatchScreenshotRequestsBytes([]ScreenshotRequest{
			{Name: "hero", URL: "https://example.com", Options: NewScreenshotOptions(WithElement("#hero"))},
			{Name: "mobile", URL: "https://m.example.com", Options: NewScreenshotOptions(WithDevice("iphone-15"))},
		})
		if len(results) != 2 {
			t.Fatalf("SharedBatchScreenshotRequestsBytes() len = %d, want 2", len(results))
		}
		if results[0].Name != "hero" || string(results[0].Data) != "https://example.com" ||
			results[0].Error != nil {
			t.Fatalf("first byte request result = %+v", results[0])
		}
		if results[1].Name != "mobile" || string(results[1].Data) != "https://m.example.com" ||
			results[1].Error != nil {
			t.Fatalf("second byte request result = %+v", results[1])
		}

		mu.Lock()
		heroOpts := optionsByURL["https://example.com"]
		mobileOpts := optionsByURL["https://m.example.com"]
		mu.Unlock()
		if heroOpts.Scan.Selector != "#hero" ||
			!heroOpts.Scan.ReturnScreenshotBytes || !heroOpts.Scan.ScreenshotSkipSave {
			t.Fatalf("hero options = %+v", heroOpts.Scan)
		}
		if mobileOpts.Chrome.DeviceName != "iPhone 15" ||
			!mobileOpts.Scan.ReturnScreenshotBytes || !mobileOpts.Scan.ScreenshotSkipSave {
			t.Fatalf("mobile options = chrome:%+v scan:%+v", mobileOpts.Chrome, mobileOpts.Scan)
		}
	})

	t.Run("batch evidence bundles write files", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if !opts.Scan.SaveHTML || !opts.Scan.SaveHeaders || !opts.Scan.SaveConsole ||
				!opts.Scan.SaveCookies || !opts.Scan.SaveNetwork {
				t.Fatalf("evidence options were not enabled: %+v", opts.Scan)
			}
			if !opts.Scan.ReturnScreenshotBytes || !opts.Scan.ScreenshotSkipSave {
				t.Fatalf("byte options were not enabled: %+v", opts.Scan)
			}
			return &models.Result{
				URL:             target,
				HTML:            "<html></html>",
				ScreenshotBytes: []byte("png"),
			}, nil
		}

		dir := t.TempDir()
		results := SharedBatchScreenshotRequestsEvidenceBundles([]ScreenshotRequest{
			{Name: "desktop", URL: "https://example.com", Options: NewScreenshotOptions(WithFullPage())},
			{Name: "mobile", URL: "https://m.example.com", Options: NewScreenshotOptions(WithDevice("iphone-15"))},
		}, dir)
		if len(results) != 2 {
			t.Fatalf("SharedBatchScreenshotRequestsEvidenceBundles() len = %d, want 2", len(results))
		}
		for _, result := range results {
			if result.Error != nil || result.Bundle == nil || result.Result == nil {
				t.Fatalf("bundle result = %+v", result)
			}
			if !strings.Contains(result.Dir, result.Name) {
				t.Fatalf("bundle dir = %q, want name %q", result.Dir, result.Name)
			}
			for _, path := range []string{
				result.Bundle.ManifestJSON,
				result.Bundle.ResultJSON,
				result.Bundle.SummaryJSON,
				result.Bundle.HTML,
				result.Bundle.Screenshot,
			} {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("bundle file %s: %v", path, err)
				}
			}
		}
	})

	t.Run("batch streaming reports canceled context", func(t *testing.T) {
		restoreSDKHooks(t)
		called := false
		sharedScreenshotWithContext = func(context.Context, string, *runner.Options) (*models.Result, error) {
			called = true
			return &models.Result{Title: "unexpected"}, nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ch := SharedBatchScreenshotStreaming(ctx, []string{"https://example.com"}, nil)
		result := <-ch
		if result.URL != "https://example.com" || result.Error == nil {
			t.Fatalf("SharedBatchScreenshotStreaming() result = %+v", result)
		}
		if _, ok := <-ch; ok {
			t.Fatal("SharedBatchScreenshotStreaming() channel still open")
		}
		if called {
			t.Fatal("sharedScreenshotWithContext was called for canceled context")
		}
	})

	t.Run("batch targets bytes callback expands targets", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if !opts.Scan.ReturnScreenshotBytes || !opts.Scan.ScreenshotSkipSave {
				t.Fatalf("byte options were not enabled: %+v", opts.Scan)
			}
			return &models.Result{URL: target, ScreenshotBytes: []byte(target)}, nil
		}

		var results []BatchBytesResult
		SharedBatchScreenshotTargetsBytesCallback(
			context.Background(),
			[]string{"example.com/admin"},
			NewScreenshotOptions(WithHTTPOnly(), WithPorts(8080)),
			func(result BatchBytesResult) {
				results = append(results, result)
			},
		)
		if len(results) != 1 {
			t.Fatalf("callback results len = %d, want 1", len(results))
		}
		if results[0].URL != "http://example.com:8080/admin" ||
			string(results[0].Data) != "http://example.com:8080/admin" ||
			results[0].Error != nil {
			t.Fatalf("callback result = %+v", results[0])
		}
	})

	t.Run("html enables save html and returns source", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if !opts.Scan.SaveHTML {
				t.Fatal("SaveHTML was not enabled")
			}
			return &models.Result{URL: target, HTML: "<html></html>"}, nil
		}
		html, result, err := SharedScreenshotHTML("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotHTML() error = %v", err)
		}
		if html != "<html></html>" || result == nil {
			t.Fatalf("SharedScreenshotHTML() html/result = %q/%+v", html, result)
		}
	})

	t.Run("evidence output timing helpers", func(t *testing.T) {
		restoreSDKHooks(t)
		var got runner.Options
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			got = *opts
			return &models.Result{
				URL:             target,
				Headers:         []models.Header{{Name: "Server", Value: "snir"}},
				Console:         []models.ConsoleLog{{Level: "warn", Message: "careful"}},
				Network:         []models.NetworkLog{{URL: target + "/asset.js", StatusCode: 304}},
				ScreenshotBytes: []byte("jpeg"),
			}, nil
		}

		headers, result, err := SharedScreenshotHeaders("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotHeaders() error = %v", err)
		}
		if len(headers) != 1 || headers[0].Name != "Server" || result == nil {
			t.Fatalf("headers/result = %+v/%+v", headers, result)
		}
		if !got.Scan.SaveHeaders {
			t.Fatal("SharedScreenshotHeaders() did not enable headers")
		}

		data, _, err := SharedScreenshotConsoleBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotConsoleBytes() error = %v", err)
		}
		if string(data) != "jpeg" {
			t.Fatalf("console bytes = %q", data)
		}
		if !got.Scan.SaveConsole || !got.Scan.ReturnScreenshotBytes || !got.Scan.ScreenshotSkipSave {
			t.Fatalf("console byte options = %+v", got.Scan)
		}

		network, _, err := SharedScreenshotNetwork("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotNetwork() error = %v", err)
		}
		if len(network) != 1 || network[0].StatusCode != 304 {
			t.Fatalf("network = %+v", network)
		}
		if !got.Scan.SaveNetwork {
			t.Fatalf("network options = %+v", got.Scan)
		}

		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			got = *opts
			return &models.Result{URL: target, ScreenshotBytes: []byte("jpeg")}, nil
		}

		if _, _, err := SharedScreenshotWithFormatBytes("https://example.com", "jpeg", 77, nil); err != nil {
			t.Fatalf("SharedScreenshotWithFormatBytes() error = %v", err)
		}
		if got.Scan.ScreenshotFormat != "jpeg" || got.Scan.ScreenshotQuality != 77 ||
			!got.Scan.ReturnScreenshotBytes || !got.Scan.ScreenshotSkipSave {
			t.Fatalf("format byte options = %+v", got.Scan)
		}

		if _, err := SharedScreenshotToPath("https://example.com", "captures/shared", nil); err != nil {
			t.Fatalf("SharedScreenshotToPath() error = %v", err)
		}
		if got.Scan.ScreenshotPath != "captures/shared" {
			t.Fatalf("ScreenshotPath = %q", got.Scan.ScreenshotPath)
		}

		if _, _, err := SharedScreenshotWithDelayBytes("https://example.com", 4*time.Second, nil); err != nil {
			t.Fatalf("SharedScreenshotWithDelayBytes() error = %v", err)
		}
		if got.Chrome.Delay != 4 {
			t.Fatalf("Delay = %d", got.Chrome.Delay)
		}

		if _, err := SharedScreenshotWithTimeout("https://example.com", 19*time.Second, nil); err != nil {
			t.Fatalf("SharedScreenshotWithTimeout() error = %v", err)
		}
		if got.Chrome.Timeout != 19 {
			t.Fatalf("Timeout = %d", got.Chrome.Timeout)
		}
	})

	t.Run("evidence enables all evidence flags", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if !opts.Scan.SaveHTML || !opts.Scan.SaveHeaders || !opts.Scan.SaveConsole ||
				!opts.Scan.SaveCookies || !opts.Scan.SaveNetwork {
				t.Fatalf("evidence options were not enabled: %+v", opts.Scan)
			}
			if !opts.Scan.ReturnScreenshotBytes || !opts.Scan.ScreenshotSkipSave {
				t.Fatalf("byte options were not enabled: %+v", opts.Scan)
			}
			return &models.Result{URL: target, ScreenshotBytes: []byte("png")}, nil
		}
		data, result, err := SharedScreenshotEvidenceBytes("https://example.com", nil)
		if err != nil {
			t.Fatalf("SharedScreenshotEvidenceBytes() error = %v", err)
		}
		if string(data) != "png" || result == nil {
			t.Fatalf("SharedScreenshotEvidenceBytes() data/result = %q/%+v", data, result)
		}
	})

	t.Run("evidence bundle writes files", func(t *testing.T) {
		restoreSDKHooks(t)
		sharedScreenshotWithContext = func(_ context.Context, target string, opts *runner.Options) (*models.Result, error) {
			if !opts.Scan.SaveHTML || !opts.Scan.ReturnScreenshotBytes {
				t.Fatalf("bundle options were not enabled: %+v", opts.Scan)
			}
			return &models.Result{
				URL:             target,
				HTML:            "<html></html>",
				ScreenshotBytes: []byte("png"),
			}, nil
		}
		dir := t.TempDir()
		bundle, result, err := SharedCaptureEvidenceBundle("https://example.com", dir, WithFullPage())
		if err != nil {
			t.Fatalf("SharedCaptureEvidenceBundle() error = %v", err)
		}
		if result == nil || bundle == nil {
			t.Fatalf("SharedCaptureEvidenceBundle() bundle/result = %+v/%+v", bundle, result)
		}
		for _, path := range []string{bundle.ManifestJSON, bundle.ResultJSON, bundle.SummaryJSON, bundle.HTML, bundle.Screenshot} {
			if path == "" {
				t.Fatalf("SharedCaptureEvidenceBundle() returned empty bundle path: %+v", bundle)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("bundle file %s: %v", path, err)
			}
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
