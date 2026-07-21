package api

import (
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestEnsureProtocol(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		https    bool
		http     bool
		expected string
	}{
		{
			name:     "已有HTTPS前缀",
			url:      "https://example.com",
			https:    true,
			http:     false,
			expected: "https://example.com",
		},
		{
			name:     "已有HTTP前缀",
			url:      "http://example.com",
			https:    true,
			http:     false,
			expected: "http://example.com",
		},
		{
			name:     "无前缀，HTTPS优先",
			url:      "example.com",
			https:    true,
			http:     false,
			expected: "https://example.com",
		},
		{
			name:     "无前缀，HTTP优先",
			url:      "example.com",
			https:    false,
			http:     true,
			expected: "http://example.com",
		},
		{
			name:     "无前缀，HTTPS和HTTP都不启用，默认HTTPS",
			url:      "example.com",
			https:    false,
			http:     false,
			expected: "https://example.com",
		},
		{
			name:     "空URL",
			url:      "",
			https:    true,
			http:     false,
			expected: "",
		},
		{
			name:     "仅有域名部分前缀",
			url:      "htt://example.com",
			https:    true,
			http:     false,
			expected: "htt://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureProtocol(tt.url, tt.https, tt.http)
			if result != tt.expected {
				t.Errorf("ensureProtocol() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCreateRunnerOptions(t *testing.T) {
	tests := []struct {
		name       string
		req        ScreenshotRequest
		serverOpts ServerOptions
		checkFunc  func(t *testing.T, opts runner.Options)
	}{
		{
			name: "基本选项",
			req: ScreenshotRequest{
				URL:     "example.com",
				HTTPS:   true,
				Timeout: 30,
				Delay:   2000,
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Chrome.Timeout != 30 {
					t.Errorf("超时设置错误: 得到 %v, 期望 %v", opts.Chrome.Timeout, 30)
				}
				if opts.Chrome.Delay != 2000 {
					t.Errorf("延迟设置错误: 得到 %v, 期望 %v", opts.Chrome.Delay, 2000)
				}
				// 注意: HTTPS和HTTP设置不在createRunnerOptions中设置，它们通过UrlWithProtocol函数在其他地方设置
			},
		},
		{
			name: "用户代理设置",
			req: ScreenshotRequest{
				URL:       "example.com",
				UserAgent: "Custom User Agent",
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Chrome.UserAgent != "Custom User Agent" {
					t.Errorf("用户代理设置错误: 得到 %v, 期望 %v", opts.Chrome.UserAgent, "Custom User Agent")
				}
			},
		},
		{
			name: "代理设置",
			req: ScreenshotRequest{
				URL:   "example.com",
				Proxy: "http://proxy.example.com:8080",
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Chrome.Proxy != "http://proxy.example.com:8080" {
					t.Errorf("代理设置错误: 得到 %v, 期望 %v", opts.Chrome.Proxy, "http://proxy.example.com:8080")
				}
			},
		},
		{
			name: "忽略证书错误",
			req: ScreenshotRequest{
				URL:              "example.com",
				IgnoreCertErrors: true,
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if !opts.Chrome.IgnoreCertErrors {
					t.Error("应该忽略证书错误")
				}
			},
		},
		{
			name: "JavaScript注入",
			req: ScreenshotRequest{
				URL:        "example.com",
				JavaScript: "console.log('test');",
				RunJSAfter: true,
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Scan.JavaScript != "console.log('test');" {
					t.Errorf("JavaScript设置错误: 得到 %v, 期望 %v", opts.Scan.JavaScript, "console.log('test');")
				}
				// 注意: RunJSAfter和RunJSBefore不在createRunnerOptions中设置
			},
		},
		{
			name: "截图路径设置",
			req: ScreenshotRequest{
				URL: "example.com",
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/custom/screenshots/path",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Scan.ScreenshotPath != "/custom/screenshots/path" {
					t.Errorf("截图路径设置错误: 得到 %v, 期望 %v", opts.Scan.ScreenshotPath, "/custom/screenshots/path")
				}
			},
		},
		{
			name: "截图格式质量和跳过保存",
			req: ScreenshotRequest{
				URL:               "example.com",
				ScreenshotFormat:  "jpeg",
				ScreenshotQuality: 80,
				SkipSave:          true,
				SaveHTML:          true,
				SaveHeaders:       true,
				SaveConsole:       true,
				SaveCookies:       true,
				SaveNetwork:       true,
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Scan.ScreenshotFormat != "jpeg" {
					t.Errorf("截图格式设置错误: 得到 %v, 期望 jpeg", opts.Scan.ScreenshotFormat)
				}
				if opts.Scan.ScreenshotQuality != 80 {
					t.Errorf("截图质量设置错误: 得到 %v, 期望 80", opts.Scan.ScreenshotQuality)
				}
				if !opts.Scan.ScreenshotSkipSave {
					t.Error("应该跳过保存截图")
				}
				if !opts.Scan.SaveHTML || !opts.Scan.SaveHeaders || !opts.Scan.SaveConsole || !opts.Scan.SaveCookies || !opts.Scan.SaveNetwork {
					t.Error("应该启用所有数据采集开关")
				}
			},
		},
		{
			name: "黑名单设置",
			req: ScreenshotRequest{
				URL: "example.com",
			},
			serverOpts: ServerOptions{
				ScreenshotPath:    "/tmp/screenshots",
				EnableBlacklist:   true,
				DefaultBlacklist:  true,
				BlacklistPatterns: []string{"example.org", "test.com"},
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if !opts.Scan.EnableBlacklist {
					t.Error("应该启用黑名单")
				}
				if !opts.Scan.DefaultBlacklist {
					t.Error("应该使用默认黑名单")
				}
				if len(opts.Scan.BlacklistPatterns) != 2 {
					t.Errorf("黑名单模式数量错误: 得到 %v, 期望 %v", len(opts.Scan.BlacklistPatterns), 2)
				}
			},
		},
		{
			name: "设备预设",
			req: ScreenshotRequest{
				URL:    "example.com",
				Device: "iphone-15",
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Chrome.DeviceName != "iPhone 15" {
					t.Errorf("设备名称错误: 得到 %q, 期望 iPhone 15", opts.Chrome.DeviceName)
				}
				if opts.Chrome.WindowX != 393 || opts.Chrome.WindowY != 852 {
					t.Errorf("设备视口错误: 得到 %dx%d, 期望 393x852", opts.Chrome.WindowX, opts.Chrome.WindowY)
				}
				if opts.Chrome.DeviceScaleFactor != 3 {
					t.Errorf("设备 DPR 错误: 得到 %v, 期望 3", opts.Chrome.DeviceScaleFactor)
				}
				if !opts.Chrome.IsMobile || !opts.Chrome.HasTouch {
					t.Error("设备预设应该启用 mobile 和 touch")
				}
				if !opts.Chrome.SpoofScreenSize || opts.Chrome.ScreenWidth != 393 || opts.Chrome.ScreenHeight != 852 {
					t.Errorf("设备屏幕伪装错误: spoof=%t screen=%dx%d", opts.Chrome.SpoofScreenSize, opts.Chrome.ScreenWidth, opts.Chrome.ScreenHeight)
				}
			},
		},
		{
			name: "设备预设允许显式指纹覆盖",
			req: ScreenshotRequest{
				URL:    "example.com",
				Device: "iphone-15",
				Fingerprint: BrowserFingerprint{
					UserAgent:   "custom-agent",
					Platform:    "CustomOS",
					ScreenWidth: 360,
				},
			},
			serverOpts: ServerOptions{
				ScreenshotPath: "/tmp/screenshots",
			},
			checkFunc: func(t *testing.T, opts runner.Options) {
				if opts.Chrome.UserAgent != "custom-agent" {
					t.Errorf("UserAgent 未覆盖: %q", opts.Chrome.UserAgent)
				}
				if opts.Chrome.Platform != "CustomOS" {
					t.Errorf("Platform 未覆盖: %q", opts.Chrome.Platform)
				}
				if opts.Chrome.ScreenWidth != 360 {
					t.Errorf("ScreenWidth 未覆盖: %d", opts.Chrome.ScreenWidth)
				}
				if opts.Chrome.ScreenHeight != 852 {
					t.Errorf("未显式提供的 ScreenHeight 应保留设备值, got %d", opts.Chrome.ScreenHeight)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 直接调用createRunnerOptions函数
			opts := createRunnerOptions(tt.req, tt.serverOpts)

			// 运行特定的检查函数
			tt.checkFunc(t, opts)
		})
	}
}

// TestCreateRunnerOptions_AdvancedBranches 覆盖 createRunnerOptions 的指纹/cookie/action/form/未知设备分支。
func TestCreateRunnerOptions_AdvancedBranches(t *testing.T) {
	t.Run("unknown device preset warns and continues", func(t *testing.T) {
		opts := createRunnerOptions(ScreenshotRequest{
			URL:    "example.com",
			Device: "nonexistent-device-xyz",
		}, ServerOptions{ScreenshotPath: "/tmp/x"})
		// 未知设备不应 panic，UA 等保持默认
		if opts.Chrome.DeviceName != "" {
			t.Fatalf("DeviceName 应为空, got %q", opts.Chrome.DeviceName)
		}
	})

	t.Run("fingerprint full override", func(t *testing.T) {
		opts := createRunnerOptions(ScreenshotRequest{
			URL: "example.com",
			Fingerprint: BrowserFingerprint{
				UserAgent:       "ua",
				AcceptLanguage:  "zh",
				Platform:        "Win32",
				Vendor:          "Google",
				Plugins:         []string{"PDF"},
				WebGLVendor:     "vendor",
				WebGLRenderer:   "renderer",
				CustomHeaders:   map[string]string{"X-Custom": "1"},
				DisableWebRTC:   true,
				SpoofScreenSize: true,
				ScreenWidth:     1920,
				ScreenHeight:    1080,
			},
		}, ServerOptions{ScreenshotPath: "/tmp/x"})
		c := opts.Chrome
		if c.UserAgent != "ua" || c.AcceptLanguage != "zh" || c.Platform != "Win32" || c.Vendor != "Google" {
			t.Fatalf("指纹字段未覆盖: %+v", c)
		}
		if len(c.Plugins) != 1 || c.Plugins[0] != "PDF" {
			t.Fatalf("Plugins = %+v", c.Plugins)
		}
		if c.WebGLVendor != "vendor" || c.WebGLRenderer != "renderer" {
			t.Fatalf("WebGL = %q/%q", c.WebGLVendor, c.WebGLRenderer)
		}
		if c.CustomHeaders["X-Custom"] != "1" {
			t.Fatalf("CustomHeaders = %+v", c.CustomHeaders)
		}
		if !c.DisableWebRTC || !c.SpoofScreenSize || c.ScreenWidth != 1920 || c.ScreenHeight != 1080 {
			t.Fatalf("spoof/webrtc = %v/%v/%d/%d", c.DisableWebRTC, c.SpoofScreenSize, c.ScreenWidth, c.ScreenHeight)
		}
	})

	t.Run("cookies and cookie header parsed", func(t *testing.T) {
		opts := createRunnerOptions(ScreenshotRequest{
			URL:             "example.com",
			Cookies:         []CustomCookie{{Name: "sid", Value: "1", Domain: "example.com", Path: "/", Secure: true, HttpOnly: true}},
			CookieHeader:    "x=2",
			CookieFile:      "cookies.json",
			CookieWriteBack: true,
		}, ServerOptions{ScreenshotPath: "/tmp/x"})
		if len(opts.Scan.Cookies) < 2 {
			t.Fatalf("Cookies 数量 = %d, 期望 >=2", len(opts.Scan.Cookies))
		}
		if opts.Scan.Cookies[0].Name != "sid" || !opts.Scan.Cookies[0].Secure || !opts.Scan.Cookies[0].HttpOnly {
			t.Fatalf("cookie0 = %+v", opts.Scan.Cookies[0])
		}
		if opts.Scan.CookiesFile != "cookies.json" || !opts.Scan.CookieWriteBack {
			t.Fatalf("cookie file/writeback = %q/%v", opts.Scan.CookiesFile, opts.Scan.CookieWriteBack)
		}
	})

	t.Run("cookie import missing file warns", func(t *testing.T) {
		opts := createRunnerOptions(ScreenshotRequest{
			URL:          "example.com",
			CookieImport: "/nonexistent/cookies.txt",
		}, ServerOptions{ScreenshotPath: "/tmp/x"})
		// 导入失败仅 warn，不应 panic，cookies 切片为空
		if len(opts.Scan.Cookies) != 0 {
			t.Fatalf("Cookies 应为空, got %d", len(opts.Scan.Cookies))
		}
	})

	t.Run("actions and form mapped", func(t *testing.T) {
		opts := createRunnerOptions(ScreenshotRequest{
			URL: "example.com",
			Actions: []InteractionAction{
				{Type: "click", Selector: "#btn", XPath: "//btn", Value: "v", WaitTime: 100, WaitVisible: true},
			},
			Form: Form{
				Fields:          []FormField{{Selector: "#user", XPath: "//u", Value: "admin", Type: "input"}},
				SubmitSelector:  "#submit",
				SubmitXPath:     "//submit",
				WaitAfterSubmit: 500,
			},
		}, ServerOptions{ScreenshotPath: "/tmp/x"})
		if len(opts.Scan.Actions) != 1 || opts.Scan.Actions[0].Type != "click" || opts.Scan.Actions[0].WaitTime != 100 || !opts.Scan.Actions[0].WaitVisible {
			t.Fatalf("Actions = %+v", opts.Scan.Actions)
		}
		if opts.Scan.Form.SubmitSelector != "#submit" || opts.Scan.Form.SubmitXPath != "//submit" || opts.Scan.Form.WaitAfterSubmit != 500 {
			t.Fatalf("Form = %+v", opts.Scan.Form)
		}
		if len(opts.Scan.Form.Fields) != 1 || opts.Scan.Form.Fields[0].Selector != "#user" {
			t.Fatalf("Form.Fields = %+v", opts.Scan.Form.Fields)
		}
	})

	t.Run("javascript defaults to run-after when no flag", func(t *testing.T) {
		opts := createRunnerOptions(ScreenshotRequest{
			URL:        "example.com",
			JavaScript: "console.log(1)",
		}, ServerOptions{ScreenshotPath: "/tmp/x"})
		if !opts.Scan.RunJSAfter {
			t.Fatal("RunJSAfter 应被默认启用")
		}
	})
}
