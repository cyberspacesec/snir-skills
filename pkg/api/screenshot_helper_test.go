package api

import (
	"testing"

	"github.com/cyberspacesec/go-snir/pkg/runner"
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
