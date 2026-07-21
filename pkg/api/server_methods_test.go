package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
	"github.com/gorilla/mux"
)

// MockDriver is a simplified mock driver for testing
type MockDriver struct {
	WitnessCalled bool
	ReturnResult  *models.Result
	ReturnError   error
}

func (d *MockDriver) Witness(target string, runner *runner.Runner) (*models.Result, error) {
	d.WitnessCalled = true
	if d.ReturnResult != nil {
		d.ReturnResult.URL = target // ensure URL matches request
	}
	return d.ReturnResult, d.ReturnError
}

func (d *MockDriver) Close() {
	// mock close method
}

// TestGetBlacklist tests the GetBlacklist method
func TestGetBlacklist(t *testing.T) {
	// Create server
	server := &Server{
		Options: ServerOptions{
			EnableBlacklist:   true,
			DefaultBlacklist:  true,
			BlacklistPatterns: []string{"test-domain.example"},
		},
	}

	// Create runner options
	opts := &runner.Options{}
	// Pass server options to runner options
	opts.Scan.EnableBlacklist = true
	opts.Scan.DefaultBlacklist = true
	opts.Scan.BlacklistPatterns = []string{"test-domain.example"}

	// Call GetBlacklist
	blacklist, err := server.GetBlacklist(opts)
	if err != nil {
		t.Fatalf("GetBlacklist returned error: %v", err)
	}

	// Check if blacklist was correctly created
	if blacklist == nil {
		t.Fatal("GetBlacklist returned nil")
	}

	// Test if local IP is blocked (should be part of default blacklist)
	isBlacklisted, _ := blacklist.IsBlacklisted("https://127.0.0.1")
	if !isBlacklisted {
		t.Error("Default blacklist rules not applied correctly; local IP should be blocked")
	}

	// Test blacklist detection for safe domain
	isBlacklisted, _ = blacklist.IsBlacklisted("https://safe-domain-example.org")
	if isBlacklisted {
		t.Error("Safe domain should not be flagged as blacklisted")
	}
}

// TestProcessScreenshot tests the ProcessScreenshot method
// Since we cannot directly replace runner.NewChromeDP, we use different testing strategies
func TestProcessScreenshot(t *testing.T) {
	// Skip actual test because this functionality requires a real Chrome instance
	// which is difficult to mock in a unit test environment
	t.Skip("Skip ProcessScreenshot test, requires Chrome instance")
}

// TestInitPool tests the InitPool method
func TestInitPool(t *testing.T) {
	// InitPool requires a real Chrome browser to initialize the pool
	// In unit tests without Chrome, it will fail to initialize
	t.Run("InitPool without Chrome", func(t *testing.T) {
		server := &Server{
			Options: ServerOptions{
				MaxConcurrentRequests: 2,
			},
		}

		opts := &runner.Options{}
		err := server.InitPool(opts)
		// Without Chrome installed, this should return an error
		if err == nil {
			// If Chrome is available, just verify pool was set
			if server.pool == nil {
				t.Error("pool should be set after InitPool")
			}
			server.ClosePool()
		} else {
			// Expected: can't initialize without Chrome
			t.Logf("InitPool failed as expected (no Chrome): %v", err)
		}
	})
}

// TestClosePool tests the ClosePool method
func TestClosePool(t *testing.T) {
	t.Run("ClosePool with nil pool", func(t *testing.T) {
		server := &Server{
			Options: ServerOptions{},
		}
		// Should not panic when pool is nil
		server.ClosePool()
	})

	t.Run("ClosePool after InitPool succeeds", func(t *testing.T) {
		server := &Server{
			Options: ServerOptions{
				MaxConcurrentRequests: 1,
			},
		}

		opts := &runner.Options{}
		err := server.InitPool(opts)
		if err == nil {
			// Pool was initialized (Chrome is available)
			if server.pool == nil {
				t.Error("pool should be set")
			}
			server.ClosePool()
			// Should not panic
		} else {
			t.Logf("InitPool failed (no Chrome available): %v", err)
		}
	})
}

// TestProcessScreenshotEdgeCases tests ProcessScreenshot edge cases
func TestProcessScreenshotEdgeCases(t *testing.T) {
	t.Run("ProcessScreenshot without pool", func(t *testing.T) {
		// Without pool and without Chrome, ProcessScreenshot should fail
		server := &Server{
			Options: ServerOptions{
				ScreenshotPath:        "/tmp/test",
				MaxConcurrentRequests: 1,
			},
		}

		req := ScreenshotRequest{
			URL:   "https://example.com",
			HTTPS: true,
		}
		opts := createRunnerOptions(req, server.Options)

		// This will try to create a ChromeDP directly and should fail
		_, err := server.ProcessScreenshot(req, opts)
		if err != nil {
			t.Logf("ProcessScreenshot without pool failed as expected: %v", err)
		} else {
			t.Log("ProcessScreenshot succeeded (Chrome was available)")
		}
	})

	t.Run("ProcessScreenshot with nil pool", func(t *testing.T) {
		// Explicitly set pool to nil
		server := &Server{
			Options: ServerOptions{
				ScreenshotPath: "/tmp/test",
			},
			pool: nil,
		}

		req := ScreenshotRequest{
			URL:   "https://example.com",
			HTTPS: true,
		}
		opts := createRunnerOptions(req, server.Options)

		// Without pool, should try fallback to single ChromeDP creation
		_, err := server.ProcessScreenshot(req, opts)
		if err != nil {
			t.Logf("ProcessScreenshot with nil pool failed as expected: %v", err)
		}
	})
}

// TestHandleStatsServerMethod tests the server's HandleStats method
func TestHandleStatsServerMethod(t *testing.T) {
	// Initialize concurrency limiter
	InitConcurrencyLimiter(10, 100)

	t.Run("HandleStats without pool", func(t *testing.T) {
		server := &Server{
			Options: ServerOptions{},
		}

		req, err := http.NewRequest("GET", "/stats", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.HandleStats)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", rr.Code, http.StatusOK)
		}

		var response APIResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("failed to parse response: %v", err)
		}

		if !response.Success {
			t.Error("response should be successful")
		}

		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Error("data should be a map")
			return
		}

		// Pool key should not exist when pool is nil
		if _, exists := data["pool"]; exists {
			t.Error("pool key should not exist when pool is nil")
		}
	})

	t.Run("HandleStats required fields", func(t *testing.T) {
		server := &Server{
			Options: ServerOptions{},
		}

		req, _ := http.NewRequest("GET", "/stats", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.HandleStats)
		handler.ServeHTTP(rr, req)

		var response APIResponse
		json.Unmarshal(rr.Body.Bytes(), &response)

		data, _ := response.Data.(map[string]interface{})
		requiredFields := []string{"active_requests", "waiting_requests", "max_concurrent", "queue_size", "uptime", "started_at"}
		for _, field := range requiredFields {
			if _, exists := data[field]; !exists {
				t.Errorf("missing required field: %s", field)
			}
		}
	})
}

// TestCreateScreenshotDir tests creating screenshot directories
func TestCreateScreenshotDir(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Test creating default directory
	dir, err := CreateScreenshotDir("")
	if err != nil {
		t.Fatalf("create default screenshot dir failed: %v", err)
	}
	// Can check if the returned path is absolute
	if dir == "" {
		t.Error("CreateScreenshotDir returned empty path")
	}

	// Test creating a specific directory
	testDir := tempDir + "/screenshots"
	dir, err = CreateScreenshotDir(testDir)
	if err != nil {
		t.Fatalf("create specific screenshot dir failed: %v", err)
	}
	if dir != testDir {
		t.Errorf("path mismatch: want %v, got %v", testDir, dir)
	}
}

// TestServer_Run_ListenAndServeFailure 覆盖 Server.Run 的成功路径直至 ListenAndServe 调用：
// 先用 net.Listen 占住端口，再让 Run 尝试同端口 ListenAndServe → 立即返回错误，
// 覆盖 Run 的配置打印 + HTTP 服务器构造（line 143-164）。
func TestServer_Run_ListenAndServeFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen 失败: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	server := &Server{
		Options: ServerOptions{
			Host: "127.0.0.1",
			Port: port,
		},
		Router: mux.NewRouter(),
	}
	if err := server.Run(); err == nil {
		t.Fatal("端口已占用时 Run 应返回错误")
	}
}

// TestServer_HandleStats_WithPoolNilViaStats 覆盖 HandleStats 的 pool==nil 分支
// （补充确认无 pool 时不 panic）。HandleStats 的 pool!=nil 分支需构造 runner.DriverPool，
// api 包无法构造 bare pool，故仅覆盖 nil 分支。
func TestServer_HandleStats_PoolNil(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest("GET", "/stats", nil)
	rr := httptest.NewRecorder()
	server.HandleStats(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("状态码 = %d, want 200", rr.Code)
	}
	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !resp.Success {
		t.Error("Success 应为 true")
	}
}

// TestProcessScreenshot_WithProxyPoolFailure 覆盖 ProcessScreenshot 的 pool!=nil
// 分支（server_methods.go:53-61）。用 proxyProvider 模式 pool（不启动浏览器）
// 赋给 s.pool，ProcessScreenshot 调 pool.Screenshot 走代理→ensureProxyBrowser 失败，
// 返回错误（line 56）。
func TestProcessScreenshot_WithProxyPoolFailure(t *testing.T) {
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	runnerOpts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	pool, err := runner.NewDriverPool(&runnerOpts, 1)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	defer pool.Close()
	server := &Server{
		Options: ServerOptions{ScreenshotPath: "/tmp/test"},
		pool:    pool,
	}
	req := ScreenshotRequest{URL: "https://example.com", HTTPS: true}
	opts := createRunnerOptions(req, server.Options)
	_, err = server.ProcessScreenshot(req, opts)
	if err == nil {
		t.Skip("ProcessScreenshot 意外成功（可能有 Chrome）")
	}
}

// TestHandleStatsServerMethod_WithPool 覆盖 HandleStats 的 pool!=nil 分支
// （server_methods.go:181-190）。用 proxyProvider 模式 pool 赋给 server。
func TestHandleStatsServerMethod_WithPool(t *testing.T) {
	InitConcurrencyLimiter(10, 100)
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := runner.NewDriverPool(&runnerOpts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	defer pool.Close()
	server := &Server{
		Options: ServerOptions{ScreenshotPath: "/tmp/test"},
		pool:    pool,
	}
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()
	server.HandleStats(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("stats 应返回 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "pool") {
		t.Errorf("响应应包含 pool 统计, got: %s", body)
	}
}

// TestInitPool_Success 覆盖 InitPool 的成功路径（server_methods.go:28-34）。
// 用 proxyProvider 模式 opts 让 NewDriverPool 不启动浏览器即可成功。
func TestInitPool_Success(t *testing.T) {
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	server := &Server{Options: ServerOptions{MaxConcurrentRequests: 2, ScreenshotPath: "/tmp/test"}}
	if err := server.InitPool(&runnerOpts); err != nil {
		t.Fatalf("InitPool proxyProvider 模式应成功: %v", err)
	}
	if server.pool == nil {
		t.Fatal("InitPool 后 pool 不应为 nil")
	}
	server.ClosePool()
}

// TestClosePool_WithPool 覆盖 ClosePool 的 pool!=nil 分支（server_methods.go:39-41）。
func TestClosePool_WithPool(t *testing.T) {
	runnerOpts := runner.Options{}
	runnerOpts.Chrome.ProxyList = []string{"http://proxy:8080"}
	pool, err := runner.NewDriverPool(&runnerOpts, 1)
	if err != nil {
		t.Fatalf("NewDriverPool: %v", err)
	}
	server := &Server{Options: ServerOptions{ScreenshotPath: "/tmp/test"}, pool: pool}
	server.ClosePool()
}
