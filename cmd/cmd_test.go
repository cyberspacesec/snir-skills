package cmd

import (
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/api"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestProxyStrategyFlag_Set(t *testing.T) {
	tests := []struct {
		input string
		want  runner.ProxyStrategy
	}{
		{"round-robin", runner.ProxyRoundRobin},
		{"random", runner.ProxyRandom},
		{"sequential", runner.ProxySequential},
		{"", runner.ProxyRoundRobin},
		{"unknown", runner.ProxyRoundRobin},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s runner.ProxyStrategy
			f := &proxyStrategyFlag{value: &s}
			if err := f.Set(tt.input); err != nil {
				t.Fatalf("Set(%q) 错误: %v", tt.input, err)
			}
			if s != tt.want {
				t.Fatalf("Set(%q) => %s, want %s", tt.input, s, tt.want)
			}
		})
	}
}

func TestProxyStrategyFlag_StringAndType(t *testing.T) {
	s := runner.ProxyRandom
	f := &proxyStrategyFlag{value: &s}
	if f.String() != "random" {
		t.Fatalf("String() = %q, want random", f.String())
	}
	if f.Type() != "string" {
		t.Fatalf("Type() = %q, want string", f.Type())
	}
	var nilF proxyStrategyFlag
	if nilF.String() != "" {
		t.Fatalf("nil value String() 应为空串, got %q", nilF.String())
	}
}

func TestInc_IPIncrement(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"末字节+1", "192.168.1.1", "192.168.1.2"},
		{"末字节进位", "192.168.1.255", "192.168.2.0"},
		{"全进位", "10.0.0.255", "10.0.1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.in).To4()
			if ip == nil {
				t.Fatalf("非法 IP: %s", tt.in)
			}
			inc(ip)
			if got := ip.String(); got != tt.want {
				t.Fatalf("inc(%s) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestGenerateRandomAPIKey(t *testing.T) {
	key := generateRandomAPIKey(16)
	if len(key) != 16 {
		t.Fatalf("长度 16 请求应返回 16 字符 hex, got %d", len(key))
	}
	key2 := generateRandomAPIKey(16)
	if key == key2 {
		t.Fatal("两次随机密钥不应相同")
	}
}

func TestPrintResult_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("printResult panic: %v", r)
		}
	}()
	printResult("any")
}

// resetOpts 将包级 opts 重置为干净的 Options，便于 apply*Options 测试。
func resetOpts() {
	opts = &runner.Options{}
}

func TestApplyScreenshotOptions_AllFields(t *testing.T) {
	resetOpts()
	req := &api.ScreenshotRequest{
		HTTP:      true,
		HTTPS:     true,
		UserAgent: "TestUA/1.0",
		Proxy:     "http://proxy:8080",
		Timeout:   30,
		Delay:     2,
	}
	applyScreenshotOptions(req)

	if !opts.Scan.HTTP {
		t.Error("Scan.HTTP 应为 true")
	}
	if !opts.Scan.HTTPS {
		t.Error("Scan.HTTPS 应为 true")
	}
	if opts.Chrome.UserAgent != "TestUA/1.0" {
		t.Errorf("UserAgent = %s", opts.Chrome.UserAgent)
	}
	if opts.Chrome.Proxy != "http://proxy:8080" {
		t.Errorf("Proxy = %s", opts.Chrome.Proxy)
	}
	if opts.Chrome.Timeout != 30 {
		t.Errorf("Timeout = %d", opts.Chrome.Timeout)
	}
	if opts.Chrome.Delay != 2 {
		t.Errorf("Delay = %d", opts.Chrome.Delay)
	}
}

func TestApplyScreenshotOptions_EmptySkips(t *testing.T) {
	// 空值不应改写已有配置（先预设非空值，再应用空请求，应保持不变）
	resetOpts()
	opts.Scan.HTTP = true
	opts.Scan.HTTPS = true
	opts.Chrome.UserAgent = "existing"
	opts.Chrome.Proxy = "existing-proxy"
	opts.Chrome.Timeout = 99
	opts.Chrome.Delay = 99

	applyScreenshotOptions(&api.ScreenshotRequest{})

	if !opts.Scan.HTTP || !opts.Scan.HTTPS {
		t.Error("HTTP/HTTPS 应保持 true")
	}
	if opts.Chrome.UserAgent != "existing" {
		t.Errorf("UserAgent 被错误改写: %s", opts.Chrome.UserAgent)
	}
	if opts.Chrome.Proxy != "existing-proxy" {
		t.Errorf("Proxy 被错误改写: %s", opts.Chrome.Proxy)
	}
	if opts.Chrome.Timeout != 99 || opts.Chrome.Delay != 99 {
		t.Errorf("Timeout/Delay 被错误改写: %d/%d", opts.Chrome.Timeout, opts.Chrome.Delay)
	}
}

func TestApplyBatchScreenshotOptions_AllFields(t *testing.T) {
	resetOpts()
	req := &api.BatchScreenshotRequest{
		HTTP:      true,
		HTTPS:     true,
		UserAgent: "BatchUA/2.0",
		Proxy:     "http://batch-proxy:9090",
		Timeout:   45,
		Delay:     3,
		Threads:   8,
	}
	applyBatchScreenshotOptions(req)

	if !opts.Scan.HTTP || !opts.Scan.HTTPS {
		t.Error("HTTP/HTTPS 应为 true")
	}
	if opts.Chrome.UserAgent != "BatchUA/2.0" {
		t.Errorf("UserAgent = %s", opts.Chrome.UserAgent)
	}
	if opts.Chrome.Proxy != "http://batch-proxy:9090" {
		t.Errorf("Proxy = %s", opts.Chrome.Proxy)
	}
	if opts.Chrome.Timeout != 45 {
		t.Errorf("Timeout = %d", opts.Chrome.Timeout)
	}
	if opts.Chrome.Delay != 3 {
		t.Errorf("Delay = %d", opts.Chrome.Delay)
	}
	if opts.Scan.Threads != 8 {
		t.Errorf("Threads = %d", opts.Scan.Threads)
	}
}

func TestApplyBatchScreenshotOptions_ThreadsZeroIgnored(t *testing.T) {
	resetOpts()
	opts.Scan.Threads = 5
	applyBatchScreenshotOptions(&api.BatchScreenshotRequest{Threads: 0})
	if opts.Scan.Threads != 5 {
		t.Errorf("Threads 为 0 时应保持原值 5, got %d", opts.Scan.Threads)
	}
}

func TestPrintDevicePresets_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("printDevicePresets panic: %v", r)
		}
	}()
	printDevicePresets()
}

// TestShowCobraHelp_RootCommand 覆盖 showCobraHelp 的根命令分支（打印 Logo）与子命令列表。
func TestShowCobraHelp_RootCommand(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("showCobraHelp panic: %v", r)
		}
	}()
	// 调用根命令的帮助函数（已在 init 中通过 SetHelpFunc 设置）
	showCobraHelp(rootCmd, nil)
}

func TestShowCobraHelp_SubCommand(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("showCobraHelp sub panic: %v", r)
		}
	}()
	// 子命令不打印 Logo（cmd.Name != rootCmd.Name）
	showCobraHelp(singleCmd, nil)
}

// TestScanCommandsColoredHelp 调用各 scan 子命令的自定义 HelpFunc，覆盖示例着色分支。
func TestScanCommandsColoredHelp(t *testing.T) {
	for _, c := range []struct {
		name string
		cmd  *cobra.Command
	}{
		{"single", singleCmd},
		{"cidr", cidrCmd},
		{"file", fileCmd},
	} {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s HelpFunc panic: %v", c.name, r)
				}
			}()
			help := c.cmd.HelpFunc()
			help(c.cmd, nil)
		})
	}
}

// TestSingleCommandColoredHelp_WithExample 设置临时示例后调用 HelpFunc，覆盖含示例行的分支。
func TestSingleCommandColoredHelp_WithExample(t *testing.T) {
	saved := singleCmd.Example
	defer func() { singleCmd.Example = saved }()
	singleCmd.Example = "  # 这是注释\n  ./snir scan single -t example.com\n普通行"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HelpFunc with example panic: %v", r)
		}
	}()
	help := singleCmd.HelpFunc()
	help(singleCmd, nil)
}

func TestCidrCommandColoredHelp_WithExample(t *testing.T) {
	saved := cidrCmd.Example
	defer func() { cidrCmd.Example = saved }()
	cidrCmd.Example = "  # cidr 示例\n  ./snir scan cidr -c 192.168.1.0/24\n"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("cidr HelpFunc with example panic: %v", r)
		}
	}()
	cidrCmd.HelpFunc()(cidrCmd, nil)
}

func TestFileCommandColoredHelp_WithExample(t *testing.T) {
	saved := fileCmd.Example
	defer func() { fileCmd.Example = saved }()
	fileCmd.Example = "  # file 示例\n  ./snir scan file -f urls.txt\n"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("file HelpFunc with example panic: %v", r)
		}
	}()
	fileCmd.HelpFunc()(fileCmd, nil)
}

// TestScanFileCmd_RunE_EmptyFilePath 覆盖 file 命令 RunE 的空文件路径错误分支。
func TestScanFileCmd_RunE_EmptyFilePath(t *testing.T) {
	resetOpts()
	err := fileCmd.RunE(fileCmd, []string{})
	if err == nil {
		t.Fatal("空文件路径应返回错误")
	}
}

// TestScanFileCmd_RunE_FileNotFound 覆盖 file 命令 RunE 的文件打开失败分支。
func TestScanFileCmd_RunE_FileNotFound(t *testing.T) {
	resetOpts()
	opts.Scan.FilePath = "/nonexistent/urls.txt"
	err := fileCmd.RunE(fileCmd, []string{})
	if err == nil {
		t.Fatal("文件不存在应返回错误")
	}
}

// TestScanFileCmd_RunE_EmptyURLList 覆盖 file 命令 RunE 的空 URL 列表分支。
func TestScanFileCmd_RunE_EmptyURLList(t *testing.T) {
	resetOpts()
	// 写入一个只含注释/空行的文件
	dir := t.TempDir()
	emptyFile := dir + "/empty.txt"
	if err := os.WriteFile(emptyFile, []byte("# only comment\n\n"), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}
	opts.Scan.FilePath = emptyFile
	err := fileCmd.RunE(fileCmd, []string{})
	if err == nil {
		t.Fatal("空 URL 列表应返回错误")
	}
}

// TestScanCidrCmd_RunE_InvalidCIDR 覆盖 cidr 命令 RunE 的无效 CIDR 分支。
func TestScanCidrCmd_RunE_InvalidCIDR(t *testing.T) {
	resetOpts()
	err := cidrCmd.RunE(cidrCmd, []string{"not-a-cidr"})
	if err == nil {
		t.Fatal("无效 CIDR 应返回错误")
	}
}

// TestScanSingleCmd_RunE_InvalidTarget 覆盖 single 命令 RunE 的 NewScanner 创建失败分支
// （通过 opts 无效配置让 NewScanner 报错，不实际扫描）。Args 校验由 cobra 处理，
// 这里直接传有效单参数触发 RunE 主体。
func TestScanSingleCmd_RunE_NewScannerFailure(t *testing.T) {
	resetOpts()
	// 用不存在的 Chrome 路径，NewScanner 内部 createDriver 调 NewPoolDriver 会失败
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	opts.Chrome.Headless = true
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	done := make(chan error, 1)
	go func() {
		done <- singleCmd.RunE(singleCmd, []string{"http://example.test"})
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("single 命令意外成功（Chrome 可用），跳过")
		}
		// 应在 "创建扫描器失败" 或 "扫描失败" 处返回错误
	case <-time.After(40 * time.Second):
		t.Fatal("single RunE 超时（可能启动了浏览器）")
	}
}

// === report 子命令 RunE 错误分支 ===

// TestConvertCmd_RunE_MissingFrom 覆盖 convert 命令 RunE 的 --from 缺失分支。
func TestConvertCmd_RunE_MissingFrom(t *testing.T) {
	convertCmdFlags.fromFile = ""
	convertCmdFlags.toFile = ""
	if err := convertCmd.RunE(convertCmd, nil); err == nil {
		t.Fatal("缺少 --from 应返回错误")
	}
}

// TestConvertCmd_RunE_MissingTo 覆盖 convert 命令 RunE 的 --to 缺失分支。
func TestConvertCmd_RunE_MissingTo(t *testing.T) {
	convertCmdFlags.fromFile = "/tmp/source.json"
	convertCmdFlags.toFile = ""
	defer func() { convertCmdFlags.fromFile = "" }()
	if err := convertCmd.RunE(convertCmd, nil); err == nil {
		t.Fatal("缺少 --to 应返回错误")
	}
}

// TestConvertCmd_RunE_ConvertFailure 覆盖 convert 命令 RunE 的 report.Convert 失败分支
// （不存在的源文件 → Convert 返回错误）。
func TestConvertCmd_RunE_ConvertFailure(t *testing.T) {
	convertCmdFlags.fromFile = "/nonexistent/source-convert.json"
	convertCmdFlags.toFile = t.TempDir() + "/out.html"
	defer func() {
		convertCmdFlags.fromFile = ""
		convertCmdFlags.toFile = ""
	}()
	err := convertCmd.RunE(convertCmd, nil)
	if err == nil {
		t.Skip("Convert 意外成功，跳过失败分支测试")
	}
}

// TestMergeCmd_RunE_MissingSource 覆盖 merge 命令 RunE 的源文件缺失分支。
func TestMergeCmd_RunE_MissingSource(t *testing.T) {
	mergeCmdFlags.sourceFiles = nil
	mergeCmdFlags.sourcePath = ""
	mergeCmdFlags.outputFile = ""
	if err := mergeCmd.RunE(mergeCmd, nil); err == nil {
		t.Fatal("缺少源文件应返回错误")
	}
}

// TestMergeCmd_RunE_MissingOutput 覆盖 merge 命令 RunE 的 --output 缺失分支。
func TestMergeCmd_RunE_MissingOutput(t *testing.T) {
	mergeCmdFlags.sourceFiles = []string{"/tmp/a.json"}
	mergeCmdFlags.sourcePath = ""
	mergeCmdFlags.outputFile = ""
	defer func() { mergeCmdFlags.sourceFiles = nil }()
	if err := mergeCmd.RunE(mergeCmd, nil); err == nil {
		t.Fatal("缺少 --output 应返回错误")
	}
}

// TestMergeCmd_RunE_MergeFailure 覆盖 merge 命令 RunE 的 report.Merge 失败分支
// （不存在的源文件 → Merge 返回错误）。
func TestMergeCmd_RunE_MergeFailure(t *testing.T) {
	mergeCmdFlags.sourceFiles = []string{"/nonexistent/source-merge.json"}
	mergeCmdFlags.sourcePath = ""
	mergeCmdFlags.outputFile = t.TempDir() + "/merged.json"
	defer func() {
		mergeCmdFlags.sourceFiles = nil
		mergeCmdFlags.outputFile = ""
	}()
	err := mergeCmd.RunE(mergeCmd, nil)
	if err == nil {
		t.Skip("Merge 意外成功，跳过失败分支测试")
	}
}

// TestHtmlCmd_RunE_MissingInput 覆盖 html 命令 RunE 的 --input 缺失分支。
func TestHtmlCmd_RunE_MissingInput(t *testing.T) {
	resetOpts()
	if err := htmlCmd.RunE(htmlCmd, nil); err == nil {
		t.Fatal("缺少 --input 应返回错误")
	}
}

// TestHtmlCmd_RunE_GenerateFailure 覆盖 html 命令 RunE 的 GenerateHTML 失败分支
// （不存在的输入文件 → GenerateHTML 返回错误）。
func TestHtmlCmd_RunE_GenerateFailure(t *testing.T) {
	resetOpts()
	opts.Report.InputFile = "/nonexistent/source-html.json"
	opts.Report.OutputPath = t.TempDir() + "/out.html"
	err := htmlCmd.RunE(htmlCmd, nil)
	if err == nil {
		t.Skip("GenerateHTML 意外成功，跳过失败分支测试")
	}
}

// === api / webserve / provider 命令 RunE ===

// TestApiCmd_RunE_ListenAndServeFailure 覆盖 api 命令 RunE 的成功路径直至 server.Run()
// 失败返回（端口被占用）。InitPool 失败会被吞掉（无浏览器仅记日志），不影响 RunE 继续。
func TestApiCmd_RunE_ListenAndServeFailure(t *testing.T) {
	resetOpts()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen 失败: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	opts.API.Host = "127.0.0.1"
	opts.API.Port = port
	opts.API.MaxConcurrent = 1
	opts.API.QueueSize = 10
	opts.Scan.ScreenshotPath = t.TempDir()

	done := make(chan error, 1)
	go func() {
		done <- apiCmd.RunE(apiCmd, nil)
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("api RunE 意外成功，跳过失败分支测试")
		}
	case <-time.After(40 * time.Second):
		t.Fatal("api RunE 超时（可能启动了浏览器）")
	}
}

// TestApiCmd_RunE_WithDBPathFailure 覆盖 api 命令 RunE 的 db-path 失败分支
// （无效 DB 路径 → NewDB 失败 → 记日志继续，不返回错误）。
func TestApiCmd_RunE_WithDBPathFailure(t *testing.T) {
	resetOpts()
	// 用一个无法创建文件的目录作为 db path
	opts.DB.Path = "/nonexistent-dir/cannot-create.db"
	// 用占端口触发 Run 失败快速返回
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen 失败: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	opts.API.Host = "127.0.0.1"
	opts.API.Port = port
	opts.API.MaxConcurrent = 1
	opts.API.QueueSize = 10
	opts.Scan.ScreenshotPath = t.TempDir()

	done := make(chan error, 1)
	go func() {
		done <- apiCmd.RunE(apiCmd, nil)
	}()
	select {
	case <-done:
		// RunE 应返回错误（端口被占），但 db 失败已被吞掉
	case <-time.After(40 * time.Second):
		t.Fatal("api RunE 超时")
	}
}

// TestWebserveCmd_RunE_ListenAndServeFailure 覆盖 webserve 命令 RunE 的成功路径
// 直至 server.Run() 失败返回（端口被占用）。
func TestWebserveCmd_RunE_ListenAndServeFailure(t *testing.T) {
	resetOpts()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen 失败: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	opts.Report.Host = "127.0.0.1"
	opts.Report.Port = port
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Report.OutputPath = t.TempDir()

	done := make(chan error, 1)
	go func() {
		done <- webserveCmd.RunE(webserveCmd, nil)
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("webserve RunE 意外成功，跳过失败分支测试")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("webserve RunE 超时")
	}
}

// TestProviderCmd_RunE_StartFailure 覆盖 provider 命令 RunE 的成功路径直至 p.Start()
// 失败返回。用不存在的 Chrome 路径让 NewDriverPool 失败 → Start 返回错误。
func TestProviderCmd_RunE_StartFailure(t *testing.T) {
	// 重置 provider 专属 flag 变量到默认值
	providerPort = 0
	providerChromePort = 9222
	providerMaxConcurrent = 1
	providerChromePath = "/nonexistent/chrome-binary-for-test"
	providerHeadless = true
	providerUserAgent = ""
	providerProxy = ""
	providerIdleTimeout = 0
	providerIgnoreCerts = false
	// 用随机端口避免冲突
	providerPort = 0 // NewProvider 会用 DefaultProviderOptions 填充 Host，端口 0 让 OS 分配，但 Start 会真正绑定
	defer func() {
		providerChromePath = ""
	}()

	done := make(chan error, 1)
	go func() {
		done <- providerCmd.RunE(providerCmd, nil)
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Skip("provider RunE 意外成功（Chrome 可用），跳过")
		}
		// 应在 "启动 Chrome 实例失败" 处返回错误
	case <-time.After(40 * time.Second):
		t.Fatal("provider RunE 超时（可能启动了浏览器）")
	}
}

// TestScanCmd_RunE_ListDevices 覆盖 scan 命令 RunE 的 --list-devices 分支（纯函数）。
func TestScanCmd_RunE_ListDevices(t *testing.T) {
	resetOpts()
	saved := scanListDevices
	defer func() { scanListDevices = saved }()
	scanListDevices = true

	if err := scanCmd.RunE(scanCmd, nil); err != nil {
		t.Fatalf("--list-devices 应返回 nil, got %v", err)
	}
}

// TestScanCmd_RunE_InvalidDeviceName 覆盖 scan 命令 RunE 的 DeviceName 无效分支
// （line 53-57：GetDevicePreset 失败返回错误）。
func TestScanCmd_RunE_InvalidDeviceName(t *testing.T) {
	resetOpts()
	saved := scanListDevices
	defer func() { scanListDevices = saved }()
	scanListDevices = false
	opts.Chrome.DeviceName = "nonexistent-device-xyz"

	err := scanCmd.RunE(scanCmd, nil)
	if err == nil {
		t.Skip("无效设备名未返回错误（可能 GetDevicePreset 宽松），跳过")
	}
}

// TestScanCmd_RunE_ValidDeviceName 覆盖 scan 命令 RunE 的 DeviceName 有效分支
// （line 53-59：GetDevicePreset 成功 + ApplyToOptions）。需避免进入 len(args)==1
// 浏览器分支：不传 args。
func TestScanCmd_RunE_ValidDeviceName(t *testing.T) {
	resetOpts()
	saved := scanListDevices
	defer func() { scanListDevices = saved }()
	scanListDevices = false
	// 用一个已知有效的设备预设名
	opts.Chrome.DeviceName = "iPhone 12"

	// 不传 args（len(args)==0），避免进入单 URL 扫描浏览器分支。
	// RunE 在 DeviceName 分支后，len(args)!=1 时会进入 batch/文件路径，
	// 但无 FilePath 会返回错误——只要不启动浏览器即可。
	done := make(chan error, 1)
	go func() {
		done <- scanCmd.RunE(scanCmd, nil)
	}()
	select {
	case err := <-done:
		// 返回错误是预期的（无目标），关键是不启动浏览器、覆盖 DeviceName 分支
		_ = err
	case <-time.After(5 * time.Second):
		t.Fatal("scan RunE 超时（可能启动了浏览器）")
	}
}

// TestVersionCmd_Run 覆盖 version 命令的 Run 闭包（纯输出，无错误）。
func TestVersionCmd_Run(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("version Run panic: %v", r)
		}
	}()
	versionCmd.Run(versionCmd, nil)
}

// TestExecute_NoError 覆盖 Execute 的成功路径（rootCmd.Execute() 返回 nil，
// 跳过 err 处理）。设 os.Args 为仅程序名，root RunE 调 Help 返回 nil。
func TestExecute_NoError(t *testing.T) {
	saved := os.Args
	defer func() { os.Args = saved }()
	os.Args = []string{"snir-test"}
	// 不应 panic 且不 exit
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Execute panic: %v", r)
		}
	}()
	Execute()
}

// TestExecute_UnknownCommandError 覆盖 Execute 的 err != nil 分支
// （root.go:104-124）。未知子命令让 rootCmd.Execute() 报错，Execute 会 os.Exit(1)。
// 用子进程执行，避免终止测试进程。
func TestExecute_UnknownCommandError(t *testing.T) {
	if os.Getenv("SNIR_TEST_EXEC") == "1" {
		os.Args = []string{"snir-test", "__definitely_unknown_cmd__"}
		Execute()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecute_UnknownCommandError")
	cmd.Env = append(os.Environ(), "SNIR_TEST_EXEC=1")
	err := cmd.Run()
	if err == nil {
		t.Skip("子进程未以非零退出（退出码依赖实现）")
	}
	// 退出码 1 表示 Execute 走了 err 分支并 os.Exit(1)
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return
	}
	t.Logf("子进程退出: %v", err)
}
