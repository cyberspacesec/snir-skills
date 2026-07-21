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

// TestBuildInteractionActions_AllBranches 覆盖 buildInteractionActions 的所有剩余分支：
// type、scroll(selector)、hover(selector)、空 selector+xpath 都空 continue、
// wait 默认 1000ms（WaitTime<=0）、wait+WaitVisible+XPath。
func TestBuildInteractionActions_AllBranches(t *testing.T) {
	tasks := buildInteractionActions([]InteractionAction{
		{Type: "type", Selector: "#input", Value: "hello"},
		{Type: "scroll", Selector: "#main", Value: "200"},
		{Type: "hover", Selector: "#menu"},
		{Type: "click"},                                     // 无 selector+xpath → continue
		{Type: "unknown", Selector: "#x"},                   // 未知 type → 无 append
		{Type: "wait", WaitTime: 0},                         // 默认 1000ms 分支
		{Type: "wait", XPath: "//ready", WaitVisible: true}, // wait+WaitVisible+XPath
	})
	// type/scroll/hover/wait(default)/wait(WaitVisible) = 5 个 action
	if len(tasks) != 5 {
		t.Fatalf("buildInteractionActions returned %d tasks, want 5", len(tasks))
	}

	// 验证 wait 默认 1000ms 分支：tasks[3] 应 sleep ~1s
	start := time.Now()
	if err := tasks[3].Do(context.Background()); err != nil {
		t.Fatalf("wait default action error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("wait default 应 sleep ~1000ms, elapsed %v", elapsed)
	}
}

// TestParseSameSite_AllBranches 覆盖 parseSameSite 的所有分支。
func TestParseSameSite_AllBranches(t *testing.T) {
	tests := []struct {
		in   string
		want string // network.CookieSameSite 的字符串表示
	}{
		{"strict", "Strict"},
		{"lax", "Lax"},
		{"none", "None"},
		{"", "Lax"},          // 默认 Lax
		{"unknown", "Lax"},   // 默认 Lax
		{"STRICT", "Strict"}, // 大写转小写
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := parseSameSite(tt.in).String()
			if got != tt.want {
				t.Errorf("parseSameSite(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func CreateTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{}))
}

type discardWriter struct{}

func (w *discardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// TestEncodeScreenshot_EdgeBranches 覆盖 encodeScreenshot 的边缘分支：
// 空 format 默认 png、jpg 别名、直接返回（已匹配 magic）、quality 边界、decode 失败。
func TestEncodeScreenshot_EdgeBranches(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})

	var pngInput bytes.Buffer
	if err := png.Encode(&pngInput, src); err != nil {
		t.Fatalf("encode source png: %v", err)
	}
	pngBytes := pngInput.Bytes()

	// 空 format → 默认 png，且 magic 匹配直接返回
	out, err := encodeScreenshot(pngBytes, "", 90)
	if err != nil {
		t.Fatalf("empty format: %v", err)
	}
	if !bytes.Equal(out, pngBytes) {
		t.Error("空 format 应默认 png 并直接返回原 buf")
	}

	// 大写 PNG → toLower 后直接返回
	out, err = encodeScreenshot(pngBytes, "PNG", 90)
	if err != nil {
		t.Fatalf("uppercase PNG: %v", err)
	}
	if !bytes.Equal(out, pngBytes) {
		t.Error("大写 PNG 应直接返回原 buf")
	}

	// jpg 别名 → 走 jpeg 编码（png 输入需 decode 后重新编码为 jpeg）
	jpgOut, err := encodeScreenshot(pngBytes, "jpg", 0) // quality=0 → 默认 90
	if err != nil {
		t.Fatalf("jpg alias quality=0: %v", err)
	}
	if !bytes.HasPrefix(jpgOut, []byte{0xff, 0xd8}) {
		t.Error("jpg 输出应有 jpeg magic")
	}

	// quality>100 → 默认 90
	jpgOut2, err := encodeScreenshot(pngBytes, "jpeg", 150)
	if err != nil {
		t.Fatalf("jpeg quality=150: %v", err)
	}
	if len(jpgOut2) == 0 {
		t.Error("jpeg quality=150 应有输出")
	}

	// 直接返回 jpeg（输入已是 jpeg magic）
	fakeJPEG := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	out, err = encodeScreenshot(fakeJPEG, "jpeg", 90)
	if err != nil {
		t.Fatalf("jpeg direct return: %v", err)
	}
	if !bytes.Equal(out, fakeJPEG) {
		t.Error("jpeg magic 匹配应直接返回原 buf")
	}

	// decode 失败：无效图像数据 + png format（magic 不匹配 → 尝试 decode → 失败）
	_, err = encodeScreenshot([]byte("not an image at all"), "png", 90)
	if err == nil {
		t.Fatal("无效图像数据应返回 decode 错误")
	}
}

// TestJsStringLiteral 覆盖 jsStringLiteral：正常编码与错误回退（理论上 json.Marshal 对 string 不失败，
// 但覆盖正常路径即可）。
func TestJsStringLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"空串", ""},
		{"普通", "hello"},
		{"含引号", `he said "hi"`},
		{"含换行", "line1\nline2"},
		{"含中文", "你好世界"},
		{"含反斜杠", `path\to\file`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsStringLiteral(tt.input)
			if len(got) == 0 {
				t.Fatalf("jsStringLiteral 返回空串")
			}
			// 应以引号包裹
			if got[0] != '"' || got[len(got)-1] != '"' {
				t.Errorf("jsStringLiteral(%q) = %q, 应以引号包裹", tt.input, got)
			}
		})
	}
}

// TestAddScriptToEvaluateOnNewDocument 构造 Action 并验证其类型（不执行，避免依赖浏览器）。
func TestAddScriptToEvaluateOnNewDocument(t *testing.T) {
	action := addScriptToEvaluateOnNewDocument("(() => {})()")
	if action == nil {
		t.Fatal("addScriptToEvaluateOnNewDocument 不应返回 nil")
	}
}

// TestAddScriptToEvaluateOnNewDocument_ExecuteWithPlainCtx 覆盖
// addScriptToEvaluateOnNewDocument 闭包体的执行路径（chromedp.go:852-855）。
// 用普通 context.Background() 执行 Do，chromedp 会因缺少 cdp session 返回错误，
// 覆盖闭包内的 err 返回分支（不启动浏览器）。
func TestAddScriptToEvaluateOnNewDocument_ExecuteWithPlainCtx(t *testing.T) {
	action := addScriptToEvaluateOnNewDocument("(() => {})()")
	// chromedp.ActionFunc 的执行需通过其 Do 方法
	type executer interface{ Do(context.Context) error }
	ex, ok := action.(executer)
	if !ok {
		t.Skip("action 不实现 Do(context.Context) error，跳过")
	}
	// 普通 ctx 执行，预期返回错误（无 cdp session），覆盖闭包体
	err := ex.Do(context.Background())
	if err == nil {
		t.Log("Do 在普通 ctx 上意外成功（可能 chromedp 版本行为不同）")
	}
	// 关键：不 panic、覆盖 line 853-854
}
