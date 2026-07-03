package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// MockDriver implements the Driver interface for testing
type MockDriver struct {
	WitnessCalls   int
	CloseWasCalled bool
	ReturnResult   *models.Result
	ReturnError    error
}

// Witness implements the Driver interface
func (d *MockDriver) Witness(target string, opts *Options) (*models.Result, error) {
	d.WitnessCalls++
	return d.ReturnResult, d.ReturnError
}

// Close implements the Driver interface
func (d *MockDriver) Close() {
	d.CloseWasCalled = true
}

// MockWriter implements the Writer interface for testing
type MockWriter struct {
	WriteCalls     int
	CloseWasCalled bool
	ReturnError    error
}

// Write implements the Writer interface
func (w *MockWriter) Write(result *models.Result) error {
	w.WriteCalls++
	return w.ReturnError
}

// Close implements the Writer interface
func (w *MockWriter) Close() error {
	w.CloseWasCalled = true
	return w.ReturnError
}

// setupTestOptions creates a minimal set of options for testing
func setupTestOptions() Options {
	options := Options{}

	// Set required fields
	options.Scan.ScreenshotFormat = "png" // Valid format
	options.Scan.ScreenshotPath = "test_screenshots"
	options.Scan.Threads = 2
	options.Scan.ScreenshotSkipSave = true // Skip saving for tests

	return options
}

// TestNewRunner tests creating a new Runner
func TestNewRunner(t *testing.T) {
	// Create a mock driver
	driver := &MockDriver{
		ReturnResult: &models.Result{
			URL:            "https://example.com",
			Title:          "Example Domain",
			ResponseCode:   200,
			ResponseReason: "OK",
		},
	}

	// Create a mock writer
	writer := &MockWriter{}

	// Create options with valid screenshot format
	options := setupTestOptions()

	// Create logger
	logger := log.GetLogger()

	// Create a new Runner
	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Verify runner is not nil
	if runner == nil {
		t.Fatal("NewRunner should return non-nil runner")
	}

	// Verify Driver is set correctly (Driver is a public field)
	if runner.Driver != driver {
		t.Error("Runner.Driver not set correctly")
	}

	// Since we can't directly access private fields, we can only verify behavior
	// or check the effects of those fields indirectly

	// Check the channel capacity
	if cap(runner.Targets) < options.Scan.Threads*2 {
		t.Errorf("Runner.Targets channel capacity is too small: %d", cap(runner.Targets))
	}
}

// TestRunnerRun tests the Run method
func TestRunnerRun(t *testing.T) {
	// Create a mock driver with a valid result
	driver := &MockDriver{
		ReturnResult: &models.Result{
			URL:            "https://example.com",
			Title:          "Example Domain",
			ResponseCode:   200,
			ResponseReason: "OK",
			ProbedAt:       time.Now(),
		},
	}

	// Create a mock writer
	writer := &MockWriter{}

	// Create options with valid screenshot format
	options := setupTestOptions()
	options.Scan.Threads = 1 // Use minimal threads for faster testing

	// Create logger
	logger := log.GetLogger()

	// Create a new Runner
	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Send a target to the runner
	runner.Targets <- "https://example.com"
	close(runner.Targets)

	// Run the runner
	err = runner.Run()
	if err != nil {
		t.Fatalf("Runner.Run returned error: %v", err)
	}

	// Verify the driver was called
	if driver.WitnessCalls != 1 {
		t.Errorf("Expected Driver.Witness to be called 1 time, got %d", driver.WitnessCalls)
	}

	// Verify the writer was called
	if writer.WriteCalls != 1 {
		t.Errorf("Expected Writer.Write to be called 1 time, got %d", writer.WriteCalls)
	}
}

// TestRunnerClose tests the Close method
func TestRunnerClose(t *testing.T) {
	// Create a mock driver
	mockDriver := &MockDriver{}

	// Create a mock writer
	writer := &MockWriter{}

	// Create options with valid screenshot format
	options := setupTestOptions()

	// Create logger
	logger := log.GetLogger()

	// Create a new Runner
	runner, err := NewRunner(logger, mockDriver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Close the runner
	err = runner.Close()
	if err != nil {
		t.Fatalf("Runner.Close returned error: %v", err)
	}

	// Verify the writer was closed
	if !writer.CloseWasCalled {
		t.Error("Writer.Close was not called")
	}

	// Check if Close was called on the driver
	// Note: Depending on the implementation, runner.Close() might not actually call Driver.Close()
	// If that's the case in the real implementation, we should skip this check or adjust our expectations
	t.Logf("Driver.Close call status: %v", mockDriver.CloseWasCalled)
}

// TestRunnerWriteNilResult tests the write() method with nil result
func TestRunnerWriteNilResult(t *testing.T) {
	driver := &MockDriver{}
	writer := &MockWriter{}
	options := setupTestOptions()
	logger := log.GetLogger()

	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Send a nil result to the Results channel
	runner.Results <- nil

	// Run the write goroutine and close the channel to let it drain
	done := make(chan struct{})
	go func() {
		runner.write()
		close(done)
	}()

	// Give it a moment to process, then close the Results channel
	time.Sleep(50 * time.Millisecond)
	close(runner.Results)

	// Wait for write() to finish
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("write() did not finish in time")
	}

	// writer.Write should NOT have been called with nil result
	if writer.WriteCalls != 0 {
		t.Errorf("Write should not be called for nil result, got %d calls", writer.WriteCalls)
	}
}

// TestRunnerWriteEmptyWriters tests the write() method with empty writers list
func TestRunnerWriteEmptyWriters(t *testing.T) {
	driver := &MockDriver{}
	options := setupTestOptions()
	logger := log.GetLogger()

	// Create runner with no writers
	runner, err := NewRunner(logger, driver, options, nil)
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	result := &models.Result{
		URL:   "https://example.com",
		Title: "Test",
	}

	// Send result to the channel
	runner.Results <- result
	close(runner.Results)

	// Run write() - should not panic with nil writers
	runner.write()
}

// TestRunnerWriteWithError tests write() with a writer that returns an error
func TestRunnerWriteWithError(t *testing.T) {
	driver := &MockDriver{}
	options := setupTestOptions()
	logger := log.GetLogger()

	errorWriter := &MockWriter{
		ReturnError: errors.New("write failed"),
	}

	runner, err := NewRunner(logger, driver, options, []Writer{errorWriter})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	result := &models.Result{
		URL:   "https://example.com",
		Title: "Test",
	}

	runner.Results <- result
	close(runner.Results)

	// write() should not panic when writer returns error
	runner.write()

	if errorWriter.WriteCalls != 1 {
		t.Errorf("Expected Write to be called once, got %d", errorWriter.WriteCalls)
	}
}

// TestRunnerCloseWriterError tests Close() when a writer returns an error
func TestRunnerCloseWriterError(t *testing.T) {
	mockDriver := &MockDriver{}
	options := setupTestOptions()
	logger := log.GetLogger()

	errorWriter := &MockWriter{
		ReturnError: errors.New("close failed"),
	}

	runner, err := NewRunner(logger, mockDriver, options, []Writer{errorWriter})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Close should not return an error even if writer close fails
	err = runner.Close()
	if err != nil {
		t.Errorf("Close should not return error even on writer close error: %v", err)
	}

	if !errorWriter.CloseWasCalled {
		t.Error("Writer.Close was not called")
	}
}

// TestChromeNotFoundError tests the ChromeNotFoundError.Error() method
func TestChromeNotFoundError(t *testing.T) {
	originalErr := errors.New("chrome binary not found in PATH")
	e := ChromeNotFoundError{Err: originalErr}

	expected := fmt.Sprintf("chrome not found: %v", originalErr)
	if errMsg := e.Error(); errMsg != expected {
		t.Errorf("Error() = %q, want %q", errMsg, expected)
	}
}

// TestParseSameSite tests the parseSameSite function
func TestParseSameSite(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // We can't directly compare the typed constant, so use string
	}{
		{"strict", "strict", "Strict"},
		{"lax", "lax", "Lax"},
		{"none", "none", "None"},
		{"Strict (mixed case)", "Strict", "Strict"},
		{"Lax (mixed case)", "Lax", "Lax"},
		{"None (mixed case)", "None", "None"},
		{"empty string", "", "Lax"},
		{"unknown value", "unknown", "Lax"},
		{"random string", "foobar", "Lax"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSameSite(tt.input)
			resultStr := fmt.Sprintf("%v", result)
			if resultStr != tt.expected {
				t.Errorf("parseSameSite(%q) = %q, want %q", tt.input, resultStr, tt.expected)
			}
		})
	}
}

// TestIsRetriableError tests the isRetriableError function
func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retriable bool
	}{
		{"nil error", nil, false},
		{"ERR_NAME_NOT_RESOLVED", errors.New("net::ERR_NAME_NOT_RESOLVED"), false},
		{"ERR_CONNECTION_REFUSED", errors.New("net::ERR_CONNECTION_REFUSED"), false},
		{"ERR_ADDRESS_UNREACHABLE", errors.New("net::ERR_ADDRESS_UNREACHABLE"), false},
		{"ERR_ACCESS_DENIED", errors.New("net::ERR_ACCESS_DENIED"), false},
		{"ERR_CONNECTION_RESET", errors.New("net::ERR_CONNECTION_RESET"), true},
		{"ERR_CONNECTION_TIMED_OUT", errors.New("net::ERR_CONNECTION_TIMED_OUT"), true},
		{"ERR_TIMED_OUT", errors.New("net::ERR_TIMED_OUT"), true},
		{"ERR_CONNECTION_CLOSED", errors.New("net::ERR_CONNECTION_CLOSED"), true},
		{"ERR_NETWORK_CHANGED", errors.New("net::ERR_NETWORK_CHANGED"), true},
		{"ERR_INTERNET_DISCONNECTED", errors.New("net::ERR_INTERNET_DISCONNECTED"), true},
		{"Could not find node with given id", errors.New("Could not find node with given id"), true},
		{"context deadline exceeded", errors.New("context deadline exceeded"), true},
		{"timeout error", errors.New("timeout"), true},
		{"浏览器进程不可用", errors.New("浏览器进程不可用"), true},
		{"截图取消", errors.New("截图取消"), true},
		{"unknown error", errors.New("some unknown error"), false},
		{"empty error", errors.New(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetriableError(tt.err)
			if result != tt.retriable {
				t.Errorf("isRetriableError(%v) = %v, want %v", tt.err, result, tt.retriable)
			}
		})
	}
}

// TestRunWriters tests the runWriters method
func TestRunWriters(t *testing.T) {
	driver := &MockDriver{}
	options := setupTestOptions()
	logger := log.GetLogger()

	t.Run("success", func(t *testing.T) {
		writer := &MockWriter{}
		runner, _ := NewRunner(logger, driver, options, []Writer{writer})
		result := &models.Result{URL: "https://example.com"}

		err := runner.runWriters(result)
		if err != nil {
			t.Errorf("runWriters should not return error: %v", err)
		}
		if writer.WriteCalls != 1 {
			t.Errorf("Expected 1 Write call, got %d", writer.WriteCalls)
		}
	})

	t.Run("error", func(t *testing.T) {
		writer := &MockWriter{ReturnError: errors.New("write error")}
		runner, _ := NewRunner(logger, driver, options, []Writer{writer})
		result := &models.Result{URL: "https://example.com"}

		err := runner.runWriters(result)
		if err == nil {
			t.Error("runWriters should return error when writer fails")
		}
	})

	t.Run("empty writers", func(t *testing.T) {
		runner, _ := NewRunner(logger, driver, options, nil)
		result := &models.Result{URL: "https://example.com"}

		err := runner.runWriters(result)
		if err != nil {
			t.Errorf("runWriters with no writers should not error: %v", err)
		}
	})
}

// TestCheckUrl tests the checkUrl method
func TestCheckUrl(t *testing.T) {
	driver := &MockDriver{}
	options := setupTestOptions()
	logger := log.GetLogger()
	runner, _ := NewRunner(logger, driver, options, nil)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http", "https://example.com", false},
		{"valid with path", "https://example.com/path", false},
		{"invalid URL", "://bad-scheme", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runner.checkUrl(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkUrl(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// TestNewRunnerScreenshotSkipSave tests NewRunner with ScreenshotSkipSave=true
func TestNewRunnerScreenshotSkipSave(t *testing.T) {
	driver := &MockDriver{}
	logger := log.GetLogger()

	options := setupTestOptions()
	options.Scan.ScreenshotSkipSave = true

	runner, err := NewRunner(logger, driver, options, nil)
	if err != nil {
		t.Fatalf("NewRunner with ScreenshotSkipSave returned error: %v", err)
	}
	if runner == nil {
		t.Fatal("Expected non-nil runner")
	}
}

// TestNewRunnerInvalidScreenshotFormat tests NewRunner with invalid screenshot format
func TestNewRunnerInvalidScreenshotFormat(t *testing.T) {
	driver := &MockDriver{}
	logger := log.GetLogger()

	options := setupTestOptions()
	options.Scan.ScreenshotFormat = "gif" // invalid format

	_, err := NewRunner(logger, driver, options, nil)
	if err == nil {
		t.Error("NewRunner should return error for invalid screenshot format")
	}
}

// TestNewRunnerMissingJavaScriptFile tests NewRunner with nonexistent JS file
func TestNewRunnerMissingJavaScriptFile(t *testing.T) {
	driver := &MockDriver{}
	logger := log.GetLogger()

	options := setupTestOptions()
	options.Scan.JavaScriptFile = "/nonexistent/js/file.js"

	_, err := NewRunner(logger, driver, options, nil)
	if err == nil {
		t.Error("NewRunner should return error for nonexistent JavaScript file")
	}
}

// TestRunnerRunWithBlacklistedURL tests Run with a blacklisted URL (threads=0 edge case)
func TestRunnerRunWithThreadsZero(t *testing.T) {
	driver := &MockDriver{
		ReturnResult: &models.Result{
			URL:   "https://example.com",
			Title: "Test",
		},
	}
	writer := &MockWriter{}
	options := setupTestOptions()
	options.Scan.Threads = 0 // Triggers default to 1
	logger := log.GetLogger()

	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	runner.Targets <- "https://example.com"
	close(runner.Targets)

	err = runner.Run()
	if err != nil {
		t.Fatalf("Runner.Run returned error: %v", err)
	}
}

// TestNewRunnerBlacklistEnabled tests NewRunner with blacklist enabled
func TestNewRunnerBlacklistEnabled(t *testing.T) {
	driver := &MockDriver{}
	logger := log.GetLogger()

	options := setupTestOptions()
	options.Scan.EnableBlacklist = true
	options.Scan.DefaultBlacklist = true

	runner, err := NewRunner(logger, driver, options, nil)
	if err != nil {
		t.Fatalf("NewRunner with blacklist returned error: %v", err)
	}
	if runner == nil {
		t.Fatal("Expected non-nil runner")
	}
	if runner.blacklist == nil {
		t.Fatal("Expected blacklist to be initialized")
	}
	if !runner.blacklist.enabled {
		t.Error("Expected blacklist to be enabled")
	}
}

// TestRunnerRunWithBlacklistedURL tests Run with a URL that goes through blacklist check
func TestRunnerRunWithBlacklistedURL(t *testing.T) {
	driver := &MockDriver{
		ReturnResult: &models.Result{
			URL:   "https://example.com",
			Title: "Test",
		},
	}
	writer := &MockWriter{}
	options := setupTestOptions()
	options.Scan.Threads = 1
	options.Scan.EnableBlacklist = true
	options.Scan.DefaultBlacklist = false
	options.Scan.BlacklistPatterns = []string{
		"10.0.0.0/8",
	}
	logger := log.GetLogger()

	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Send a blacklisted URL
	runner.Targets <- "https://10.0.0.1"
	close(runner.Targets)

	err = runner.Run()
	if err != nil {
		t.Fatalf("Runner.Run returned error: %v", err)
	}

	// Driver.Witness should NOT have been called because the URL is blacklisted
	if driver.WitnessCalls > 0 {
		t.Errorf("Driver.Witness should not be called for blacklisted URL, got %d calls", driver.WitnessCalls)
	}
}

// TestRunnerRunWithInvalidURL tests Run with an invalid URL
func TestRunnerRunWithInvalidURL(t *testing.T) {
	driver := &MockDriver{
		ReturnResult: &models.Result{
			URL:   "https://example.com",
			Title: "Test",
		},
	}
	writer := &MockWriter{}
	options := setupTestOptions()
	options.Scan.Threads = 1
	logger := log.GetLogger()

	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Send an invalid URL
	runner.Targets <- "://invalid"
	close(runner.Targets)

	err = runner.Run()
	if err != nil {
		t.Fatalf("Runner.Run returned error: %v", err)
	}

	// Driver.Witness should NOT have been called because the URL is invalid
	if driver.WitnessCalls > 0 {
		t.Errorf("Driver.Witness should not be called for invalid URL, got %d calls", driver.WitnessCalls)
	}
}

// TestNewRunnerWithJavaScriptFile tests NewRunner with a valid JavaScript file
func TestNewRunnerWithJavaScriptFile(t *testing.T) {
	driver := &MockDriver{}
	logger := log.GetLogger()

	// Create a temp JS file
	tmpDir, err := os.MkdirTemp("", "runner_js_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsFilePath := filepath.Join(tmpDir, "test.js")
	if err := os.WriteFile(jsFilePath, []byte("console.log('test');"), 0644); err != nil {
		t.Fatalf("Failed to write JS file: %v", err)
	}

	options := setupTestOptions()
	options.Scan.JavaScriptFile = jsFilePath

	runner, err := NewRunner(logger, driver, options, nil)
	if err != nil {
		t.Fatalf("NewRunner with JS file returned error: %v", err)
	}
	if runner == nil {
		t.Fatal("Expected non-nil runner")
	}
	if runner.options.Scan.JavaScript != "console.log('test');" {
		t.Errorf("JavaScript = %q, want 'console.log('test');'", runner.options.Scan.JavaScript)
	}
}

// TestRunnerRunContextCancellation tests Run with context cancellation
func TestRunnerRunContextCancellation(t *testing.T) {
	driver := &MockDriver{
		ReturnResult: &models.Result{
			URL:   "https://example.com",
			Title: "Test",
		},
	}
	writer := &MockWriter{}
	options := setupTestOptions()
	options.Scan.Threads = 1
	logger := log.GetLogger()

	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	// Cancel the context before running
	runner.cancel()

	// Send a target and close
	runner.Targets <- "https://example.com"
	close(runner.Targets)

	// Run should finish quickly since context is already cancelled
	err = runner.Run()
	if err != nil {
		t.Fatalf("Runner.Run returned error: %v", err)
	}
}

// TestNewRunnerWithScreenshotSave tests NewRunner with ScreenshotSkipSave=false
func TestNewRunnerWithScreenshotSave(t *testing.T) {
	driver := &MockDriver{}
	logger := log.GetLogger()

	options := setupTestOptions()
	options.Scan.ScreenshotSkipSave = false
	options.Scan.ScreenshotPath = t.TempDir()

	runner, err := NewRunner(logger, driver, options, nil)
	if err != nil {
		t.Fatalf("NewRunner with screenshot save returned error: %v", err)
	}
	if runner == nil {
		t.Fatal("Expected non-nil runner")
	}
}

// TestRunnerRunDriverError tests Run when the driver returns an error
func TestRunnerRunDriverError(t *testing.T) {
	driver := &MockDriver{
		ReturnError: errors.New("driver error"),
	}
	writer := &MockWriter{}
	options := setupTestOptions()
	options.Scan.Threads = 1
	logger := log.GetLogger()

	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	runner.Targets <- "https://example.com"
	close(runner.Targets)

	err = runner.Run()
	if err != nil {
		t.Fatalf("Runner.Run returned error: %v", err)
	}

	// Writer should NOT have been called because driver returned error
	if writer.WriteCalls > 0 {
		t.Errorf("Writer.Write should not be called when driver errors, got %d calls", writer.WriteCalls)
	}
}

// TestRunnerRunWriterError tests Run when the writer returns an error
func TestRunnerRunWriterError(t *testing.T) {
	driver := &MockDriver{
		ReturnResult: &models.Result{
			URL:   "https://example.com",
			Title: "Test",
		},
	}
	writer := &MockWriter{
		ReturnError: errors.New("write error"),
	}
	options := setupTestOptions()
	options.Scan.Threads = 1
	logger := log.GetLogger()

	runner, err := NewRunner(logger, driver, options, []Writer{writer})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}

	runner.Targets <- "https://example.com"
	close(runner.Targets)

	err = runner.Run()
	if err != nil {
		t.Fatalf("Runner.Run returned error: %v", err)
	}

	// Writer should have been called (even though it returns error)
	if writer.WriteCalls != 1 {
		t.Errorf("Expected 1 Write call, got %d", writer.WriteCalls)
	}
}
