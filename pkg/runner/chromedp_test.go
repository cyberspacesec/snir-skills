package runner

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func makeTestOptions() Options {
	return Options{
		Chrome: struct {
			Path              string
			UserAgent         string
			Proxy             string
			Timeout           int
			Delay             int
			WindowX           int
			WindowY           int
			WSS               string
			Headless          bool
			IgnoreCertErrors  bool
			ProxyList         []string
			ProxyFile         string
			ProxyURL          string
			ProxyStrategy     ProxyStrategy
			AcceptLanguage    string
			Platform          string
			Vendor            string
			Plugins           []string
			WebGLVendor       string
			WebGLRenderer     string
			CustomHeaders     map[string]string
			DisableWebRTC     bool
			SpoofScreenSize   bool
			ScreenWidth       int
			ScreenHeight      int
			DeviceName        string
			DeviceScaleFactor float64
			IsMobile          bool
			HasTouch          bool
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

func TestBuildFingerprintScript(t *testing.T) {
	opts := makeTestOptions()

	if script := buildFingerprintScript(&opts); script != "" {
		t.Fatalf("empty fingerprint options should not generate script, got %q", script)
	}

	opts.Chrome.Platform = `Win"32`
	opts.Chrome.Vendor = "Google Inc."
	opts.Chrome.Plugins = []string{"Chrome PDF Viewer"}
	opts.Chrome.WebGLVendor = "Intel Inc."
	opts.Chrome.WebGLRenderer = "Intel Iris"
	opts.Chrome.SpoofScreenSize = true
	opts.Chrome.ScreenWidth = 1920
	opts.Chrome.ScreenHeight = 1080
	opts.Chrome.DisableWebRTC = true

	script := buildFingerprintScript(&opts)
	for _, want := range []string{
		`"Win\"32"`,
		"navigator, 'vendor'",
		"Chrome PDF Viewer",
		"37445",
		"37446",
		"RTCPeerConnection",
		"width: 1920",
		"height: 1080",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("fingerprint script missing %q:\n%s", want, script)
		}
	}
}

func TestBuildFingerprintScript_DisableWebRTCOnly(t *testing.T) {
	opts := makeTestOptions()
	opts.Chrome.DisableWebRTC = true

	script := buildFingerprintScript(&opts)
	if !strings.Contains(script, "RTCPeerConnection") {
		t.Fatalf("DisableWebRTC alone should generate a WebRTC override script, got %q", script)
	}
}

func TestBuildDeviceEmulationActions(t *testing.T) {
	opts := makeTestOptions()
	opts.Chrome.WindowX = 393
	opts.Chrome.WindowY = 852
	opts.Chrome.DeviceScaleFactor = 3
	opts.Chrome.IsMobile = true
	opts.Chrome.HasTouch = true

	actions := buildDeviceEmulationActions(&opts)
	if len(actions) != 2 {
		t.Fatalf("buildDeviceEmulationActions() returned %d actions, want 2", len(actions))
	}
}

func TestBuildDeviceEmulationActions_DesktopNoOverride(t *testing.T) {
	opts := makeTestOptions()
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800

	actions := buildDeviceEmulationActions(&opts)
	if len(actions) != 0 {
		t.Fatalf("desktop options without device emulation should not create actions, got %d", len(actions))
	}
}

// TestBuildDeviceEmulationActions_EdgeCases 覆盖 nil opts、ScreenWidth/Height 覆盖、
// width<=0 早返回、以及 SpoofScreenSize 触发 scaleFactor=1 的分支。
func TestBuildDeviceEmulationActions_EdgeCases(t *testing.T) {
	// nil opts → nil
	if actions := buildDeviceEmulationActions(nil); actions != nil {
		t.Fatalf("nil opts 应返回 nil, got %v", actions)
	}

	// ScreenWidth/Height 覆盖 WindowX/Y（IsMobile=true 触发 scaleFactor=1；无 HasTouch）
	opts := makeTestOptions()
	opts.Chrome.WindowX = 1
	opts.Chrome.WindowY = 1
	opts.Chrome.ScreenWidth = 393
	opts.Chrome.ScreenHeight = 852
	opts.Chrome.IsMobile = true
	actions := buildDeviceEmulationActions(&opts)
	if len(actions) != 1 {
		t.Fatalf("ScreenWidth/Height 覆盖应返回 1 action (仅 metrics), got %d", len(actions))
	}

	// width<=0 早返回 nil（即便 ScreenWidth 也<=0）
	opts2 := makeTestOptions()
	opts2.Chrome.WindowX = 0
	opts2.Chrome.WindowY = 0
	opts2.Chrome.ScreenWidth = 0
	opts2.Chrome.ScreenHeight = 0
	opts2.Chrome.IsMobile = true
	if actions := buildDeviceEmulationActions(&opts2); actions != nil {
		t.Fatalf("width<=0 应返回 nil, got %v", actions)
	}

	// SpoofScreenSize 触发 scaleFactor=1（非 mobile、无 touch）
	opts3 := makeTestOptions()
	opts3.Chrome.WindowX = 1280
	opts3.Chrome.WindowY = 800
	opts3.Chrome.SpoofScreenSize = true
	actions = buildDeviceEmulationActions(&opts3)
	if len(actions) != 1 {
		t.Fatalf("SpoofScreenSize 应返回 1 action (仅 metrics), got %d", len(actions))
	}

	// HasTouch 但无 mobile/scaleFactor=1（scaleFactor 由 HasTouch 触发）
	opts4 := makeTestOptions()
	opts4.Chrome.WindowX = 393
	opts4.Chrome.WindowY = 852
	opts4.Chrome.HasTouch = true
	actions = buildDeviceEmulationActions(&opts4)
	if len(actions) != 2 {
		t.Fatalf("HasTouch 触发 scaleFactor=1 应返回 2 actions (metrics+touch), got %d", len(actions))
	}
}

func TestChromeDP_Witness_MissingJavaScriptFile(t *testing.T) {
	opts := makeTestOptions()
	opts.Scan.JavaScriptFile = "/nonexistent/js/file.js"

	driver := &ChromeDP{opts: &opts}
	result, err := driver.Witness("https://example.com", &opts)
	if err == nil {
		t.Fatal("Witness should fail when JavaScriptFile is missing")
	}
	if result == nil || !result.Failed {
		t.Fatalf("Witness should return a failed result, got %#v", result)
	}
	if !strings.Contains(result.FailedReason, "no such file") {
		t.Fatalf("unexpected failed reason: %q", result.FailedReason)
	}
}

func TestEncodeScreenshot(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{G: 255, A: 255})
	src.Set(0, 1, color.RGBA{B: 255, A: 255})
	src.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 128})

	var input bytes.Buffer
	if err := png.Encode(&input, src); err != nil {
		t.Fatalf("encode source png: %v", err)
	}

	pngOut, err := encodeScreenshot(input.Bytes(), "png", 90)
	if err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if !bytes.HasPrefix(pngOut, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("png output has unexpected signature: %x", pngOut[:4])
	}

	jpegOut, err := encodeScreenshot(input.Bytes(), "jpeg", 80)
	if err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	if !bytes.HasPrefix(jpegOut, []byte{0xff, 0xd8}) {
		t.Fatalf("jpeg output has unexpected signature: %x", jpegOut[:2])
	}

	if _, err := encodeScreenshot(input.Bytes(), "gif", 90); err == nil {
		t.Fatal("unsupported format should return error")
	}
}

func TestBuildInteractionActions(t *testing.T) {
	tasks := buildInteractionActions([]InteractionAction{
		{Type: "wait", WaitTime: 250},
		{Type: "wait"},
		{Type: "wait", Selector: "#ready", WaitVisible: true},
		{Type: "click"},
		{Type: "scroll", XPath: "//main", Value: "100"},
		{Type: "hover", XPath: "//nav"},
	})

	if len(tasks) != 4 {
		t.Fatalf("buildInteractionActions returned %d tasks, want 4", len(tasks))
	}

	start := time.Now()
	if err := tasks[0].Do(context.Background()); err != nil {
		t.Fatalf("wait action returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("wait action elapsed %v, want at least 200ms", elapsed)
	}

	if got := buildInteractionActions(nil); got != nil {
		t.Fatalf("buildInteractionActions(nil) = %v, want nil", got)
	}
}

func CreateTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{}))
}

type discardWriter struct{}

func (w *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
