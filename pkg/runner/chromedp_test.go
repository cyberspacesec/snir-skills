package runner

import (
	"log/slog"
	"os"
	"testing"
)

// TestNewChromeDP 测试 Chrome 驱动的创建
func TestNewChromeDP(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
		wantErr bool
	}{
		{
			name: "基本配置",
			options: &Options{
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
					Headless:  true,
					WindowX:   1280,
					WindowY:   800,
					UserAgent: "测试用户代理",
					Timeout:   30,
				},
			},
			wantErr: false,
		},
		{
			name: "带代理配置",
			options: &Options{
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
					Proxy:    "http://localhost:8080",
					Timeout:  30,
				},
			},
			wantErr: false,
		},
		{
			name: "带指纹配置",
			options: &Options{
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
					Headless:        true,
					WindowX:         1280,
					WindowY:         800,
					Platform:        "Win32",
					Vendor:          "Google Inc.",
					WebGLVendor:     "测试WebGL厂商",
					WebGLRenderer:   "测试WebGL渲染器",
					SpoofScreenSize: true,
					ScreenWidth:     1920,
					ScreenHeight:    1080,
					DisableWebRTC:   true,
					Timeout:         30,
				},
			},
			wantErr: false,
		},
		{
			name: "忽略证书错误",
			options: &Options{
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
					Headless:         true,
					WindowX:          1280,
					WindowY:          800,
					IgnoreCertErrors: true,
					Timeout:          30,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建驱动
			driver, err := NewChromeDP(tt.options)

			// 检查错误
			if (err != nil) != tt.wantErr {
				t.Errorf("NewChromeDP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果没有错误，检查驱动是否被正确创建
			if err == nil {
				if driver == nil {
					t.Error("NewChromeDP() returned nil driver without error")
				} else {
					// 检查选项是否被正确设置
					if driver.opts == nil {
						t.Error("NewChromeDP() driver has nil options")
					} else if driver.opts != tt.options {
						t.Error("NewChromeDP() driver options not set correctly")
					}

					// 检查上下文是否被创建
					if driver.ctx == nil {
						t.Error("NewChromeDP() driver has nil context")
					}

					// 检查取消函数是否被创建
					if driver.cancel == nil {
						t.Error("NewChromeDP() driver has nil cancel function")
					}

					// 关闭驱动
					driver.Close()
				}
			}
		})
	}
}

// TestChromeDP_Close 测试关闭方法
func TestChromeDP_Close(t *testing.T) {
	// 创建驱动
	driver, err := NewChromeDP(&Options{
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
			Timeout:  5,
		},
	})

	if err != nil {
		t.Fatalf("无法创建驱动进行测试: %v", err)
	}

	// 测试关闭方法
	t.Run("关闭方法", func(t *testing.T) {
		// 确认没有panic
		driver.Close()
		// 再次调用不应当panic
		driver.Close()
	})
}

// TestChromeDP_Witness 测试Witness方法
func TestChromeDP_Witness(t *testing.T) {
	// 由于这个测试需要真实的Chrome浏览器和网络连接，
	// 我们默认跳过它，除非显式设置了RUNNER_INTEGRATION_TEST环境变量
	if os.Getenv("RUNNER_INTEGRATION_TEST") != "true" {
		t.Skip("跳过集成测试，设置 RUNNER_INTEGRATION_TEST=true 环境变量以启用")
	}

	// 以下测试代码只有在启用集成测试时才会运行
	// 创建临时截图目录
	tempDir, err := os.MkdirTemp("", "chromedp_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建选项
	opts := &Options{
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
			Headless:  true,
			WindowX:   1280,
			WindowY:   800,
			Timeout:   30,
			UserAgent: "Go-Snir Test Agent",
		},
		Scan: struct {
			Driver             string
			Threads            int
			ScreenshotPath     string
			ScreenshotFormat   string
			ScreenshotQuality  int
			ScreenshotSkipSave bool
			SaveHTML           bool
			SaveHeaders        bool
			SaveConsole        bool
			SaveCookies        bool
			SaveNetwork        bool
			HTTP               bool
			HTTPS              bool
			Ports              []int
			Timeout            int
			MaxRetries         int
			JavaScript         string
			JavaScriptFile     string
			FilePath           string
			EnableBlacklist    bool
			DefaultBlacklist   bool
			BlacklistPatterns  []string
			BlacklistFile      string
			RunJSBefore        bool
			RunJSAfter         bool
			Cookies            []CustomCookie
			Selector           string
			XPath              string
			CaptureFullPage    bool
			Actions            []InteractionAction
			Form               Form
		}{
			ScreenshotPath:    tempDir,
			ScreenshotFormat:  "png",
			ScreenshotQuality: 90,
		},
	}

	// 创建驱动
	driver, err := NewChromeDP(opts)
	if err != nil {
		t.Fatalf("无法创建驱动: %v", err)
	}
	defer driver.Close()

	// 创建Runner
	runnerLogger := CreateTestLogger()
	runner, err := NewRunner(runnerLogger, driver, *opts, nil)
	if err != nil {
		t.Fatalf("无法创建Runner: %v", err)
	}
	defer runner.Close()

	// 测试Witness方法
	t.Run("基本截图", func(t *testing.T) {
		// 测试简单的静态网站
		result, err := driver.Witness("https://example.com", opts)
		if err != nil {
			t.Fatalf("Witness() error = %v", err)
		}

		// 验证结果
		if result == nil {
			t.Fatal("Witness() result is nil")
		}

		// 验证基本字段
		if result.URL != "https://example.com" {
			t.Errorf("Witness() result.URL = %v, want %v", result.URL, "https://example.com")
		}

		if result.Title == "" {
			t.Error("Witness() result.Title is empty")
		}

		if result.ResponseCode == 0 {
			t.Error("Witness() result.ResponseCode is 0")
		}

		// 验证截图被创建
		if result.Filename == "" {
			t.Error("Witness() result.Filename is empty")
		} else if _, err := os.Stat(result.Filename); os.IsNotExist(err) {
			t.Errorf("Witness() file not created at %v", result.Filename)
		}

		// 验证HTML内容
		if result.HTML == "" {
			t.Error("Witness() result.HTML is empty")
		}
	})
}

// CreateTestLogger 创建用于测试的Logger
func CreateTestLogger() *slog.Logger {
	// 创建一个丢弃所有日志的Logger
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{}))
}

// discardWriter 是一个丢弃所有写入内容的Writer
type discardWriter struct{}

func (w *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
