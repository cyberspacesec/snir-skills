package runner

import (
	"log/slog"
	"os"
	"testing"
)

func makeTestOptions() Options {
	return Options{
		Chrome: struct {
			Path             string
			UserAgent        string
			Proxy            string
			Timeout          int
			Delay            int
			WindowX          int
			WindowY          int
			WSS              string
			Headless         bool
			IgnoreCertErrors bool
			ProxyList        []string
			ProxyFile        string
			ProxyURL         string
			ProxyStrategy    ProxyStrategy
			AcceptLanguage   string
			Platform         string
			Vendor           string
			Plugins          []string
			WebGLVendor      string
			WebGLRenderer    string
			CustomHeaders    map[string]string
			DisableWebRTC    bool
			SpoofScreenSize  bool
			ScreenWidth      int
			ScreenHeight     int
		}{
			Headless: true,
			WindowX:  1280,
			WindowY:  800,
			Timeout:  30,
		},
	}
}

func TestNewChromeDP(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
		wantErr bool
	}{
		{
			name:    "基本配置",
			options: func() *Options { o := makeTestOptions(); o.Chrome.UserAgent = "测试用户代理"; return &o }(),
			wantErr: false,
		},
		{
			name:    "带代理配置",
			options: func() *Options { o := makeTestOptions(); o.Chrome.Proxy = "http://localhost:8080"; return &o }(),
			wantErr: false,
		},
		{
			name: "带指纹配置",
			options: func() *Options {
				o := makeTestOptions()
				o.Chrome.Platform = "Win32"
				o.Chrome.Vendor = "Google Inc."
				o.Chrome.WebGLVendor = "测试WebGL厂商"
				o.Chrome.WebGLRenderer = "测试WebGL渲染器"
				o.Chrome.SpoofScreenSize = true
				o.Chrome.ScreenWidth = 1920
				o.Chrome.ScreenHeight = 1080
				o.Chrome.DisableWebRTC = true
				return &o
			}(),
			wantErr: false,
		},
		{
			name:    "忽略证书错误",
			options: func() *Options { o := makeTestOptions(); o.Chrome.IgnoreCertErrors = true; return &o }(),
			wantErr: false,
		},
		{
			name: "代理列表轮换",
			options: func() *Options {
				o := makeTestOptions()
				o.Chrome.ProxyList = []string{"http://a:8080", "http://b:8080"}
				o.Chrome.ProxyStrategy = ProxyRoundRobin
				return &o
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, err := NewChromeDP(tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewChromeDP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if driver == nil {
					t.Error("NewChromeDP() returned nil driver without error")
				} else {
					if driver.opts == nil {
						t.Error("NewChromeDP() driver has nil options")
					} else if driver.opts != tt.options {
						t.Error("NewChromeDP() driver options not set correctly")
					}
					if driver.ctx == nil {
						t.Error("NewChromeDP() driver has nil context")
					}
					if driver.cancel == nil {
						t.Error("NewChromeDP() driver has nil cancel function")
					}
					driver.Close()
				}
			}
		})
	}
}

func TestChromeDP_Close(t *testing.T) {
	opts := func() *Options {
		o := makeTestOptions()
		o.Chrome.Timeout = 5
		return &o
	}()

	driver, err := NewChromeDP(opts)
	if err != nil {
		t.Fatalf("无法创建驱动进行测试: %v", err)
	}

	t.Run("关闭方法", func(t *testing.T) {
		driver.Close()
		driver.Close()
	})
}

func TestChromeDP_Witness(t *testing.T) {
	if os.Getenv("RUNNER_INTEGRATION_TEST") != "true" {
		t.Skip("跳过集成测试，设置 RUNNER_INTEGRATION_TEST=true 环境变量以启用")
	}

	tempDir, err := os.MkdirTemp("", "chromedp_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	opts := func() *Options {
		o := makeTestOptions()
		o.Chrome.UserAgent = "Go-Snir Test Agent"
		o.Scan.ScreenshotPath = tempDir
		o.Scan.ScreenshotFormat = "png"
		o.Scan.ScreenshotQuality = 90
		return &o
	}()

	driver, err := NewChromeDP(opts)
	if err != nil {
		t.Fatalf("无法创建驱动: %v", err)
	}
	defer driver.Close()

	runnerLogger := CreateTestLogger()
	r, err := NewRunner(runnerLogger, driver, *opts, nil)
	if err != nil {
		t.Fatalf("无法创建Runner: %v", err)
	}
	defer r.Close()

	t.Run("基本截图", func(t *testing.T) {
		result, err := driver.Witness("https://example.com", opts)
		if err != nil {
			t.Fatalf("Witness() error = %v", err)
		}

		if result == nil {
			t.Fatal("Witness() result is nil")
		}
		if result.URL != "https://example.com" {
			t.Errorf("Witness() result.URL = %v, want %v", result.URL, "https://example.com")
		}
		if result.Title == "" {
			t.Error("Witness() result.Title is empty")
		}
	})
}

func CreateTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{}))
}

type discardWriter struct{}

func (w *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
