package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
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
				ScreenshotPath: "/tmp/test",
				MaxConcurrentRequests: 1,
			},
		}

		req := ScreenshotRequest{
			URL: "https://example.com",
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
			URL: "https://example.com",
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
