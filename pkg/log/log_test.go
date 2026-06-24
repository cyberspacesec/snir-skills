package log

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	charm "github.com/charmbracelet/log"
)

// 自定义 Writer 用于捕获日志输出
type testWriter struct {
	buffer bytes.Buffer
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return w.buffer.Write(p)
}

func (w *testWriter) String() string {
	return w.buffer.String()
}

func (w *testWriter) Reset() {
	w.buffer.Reset()
}

func setupTestLogger() (*testWriter, *Logger) {
	// 创建测试 Writer
	w := &testWriter{}

	// 创建测试 Logger
	l := &Logger{
		level:  InfoLevel,
		writer: w,
	}

	// 保存原始 Logger
	origLogger := defaultLogger

	// 设置为测试 Logger
	defaultLogger = l

	return w, origLogger
}

func resetLogger(orig *Logger) {
	defaultLogger = orig
}

func TestLogLevels(t *testing.T) {
	w, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	tests := []struct {
		name      string
		logFunc   func(msg string, args ...interface{})
		level     charm.Level
		expectLog bool
	}{
		{
			name:      "Debug级别 - 仅当Debug启用时才记录",
			logFunc:   Debug,
			level:     DebugLevel,
			expectLog: false, // 默认是 InfoLevel，不应记录 Debug
		},
		{
			name:      "Info级别 - 默认应记录",
			logFunc:   Info,
			level:     InfoLevel,
			expectLog: true,
		},
		{
			name:      "Warn级别 - 默认应记录",
			logFunc:   Warn,
			level:     WarnLevel,
			expectLog: true,
		},
		{
			name:      "Error级别 - 默认应记录",
			logFunc:   Error,
			level:     ErrorLevel,
			expectLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置缓冲区
			w.Reset()

			// 调用日志函数
			tt.logFunc("测试消息", "key", "value")

			// 检查输出
			if tt.expectLog && w.String() == "" {
				t.Errorf("期望日志消息，但没有得到任何输出")
			}

			if !tt.expectLog && w.String() != "" {
				t.Errorf("期望没有日志消息，但得到了输出: %s", w.String())
			}

			// 如果有输出，检查格式
			if w.String() != "" {
				// 应该包含消息
				if !strings.Contains(w.String(), "测试消息") {
					t.Errorf("日志输出应该包含消息，但得到: %s", w.String())
				}

				// 应该包含键值对
				if !strings.Contains(w.String(), "key=value") {
					t.Errorf("日志输出应该包含键值对，但得到: %s", w.String())
				}
			}
		})
	}
}

func TestEnableDebug(t *testing.T) {
	w, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	// 先测试默认级别（Info）
	Debug("这不应该被记录")
	if w.String() != "" {
		t.Errorf("Debug消息不应被记录，但得到了输出: %s", w.String())
	}

	// 启用 Debug
	EnableDebug()
	w.Reset()

	// 现在应该记录 Debug 消息
	Debug("这应该被记录")
	if w.String() == "" {
		t.Error("启用Debug后，Debug消息应被记录，但没有得到输出")
	}

	if !strings.Contains(w.String(), "这应该被记录") {
		t.Errorf("日志输出应该包含Debug消息，但得到: %s", w.String())
	}
}

func TestEnableSilence(t *testing.T) {
	w, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	// 先测试默认级别（Info）
	Info("这应该被记录")
	if w.String() == "" {
		t.Error("Info消息应被记录，但没有得到输出")
	}
	w.Reset()

	// 启用静默模式
	EnableSilence()

	// 现在不应该记录任何消息（除了 Fatal，但我们不测试它，因为它会退出程序）
	Info("这不应该被记录")
	Debug("这不应该被记录")
	Warn("这不应该被记录")
	Error("这不应该被记录")

	if w.String() != "" {
		t.Errorf("静默模式下不应有日志输出，但得到了: %s", w.String())
	}
}

func TestSuccess(t *testing.T) {
	w, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	Success("成功消息")

	if w.String() == "" {
		t.Error("Success应该记录消息，但没有得到输出")
	}

	if !strings.Contains(w.String(), "成功消息") {
		t.Errorf("Success输出应该包含消息，但得到: %s", w.String())
	}

	// Success应该使用绿色标记
	if !strings.Contains(w.String(), "✓") {
		t.Errorf("Success输出应该包含✓标记，但得到: %s", w.String())
	}
}

func TestCommandTitle(t *testing.T) {
	w, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	CommandTitle("测试命令")

	if w.String() == "" {
		t.Error("CommandTitle应该记录消息，但没有得到输出")
	}

	if !strings.Contains(w.String(), "测试命令") {
		t.Errorf("CommandTitle输出应该包含消息，但得到: %s", w.String())
	}

	// CommandTitle应该包含::前缀
	if !strings.Contains(w.String(), "::") {
		t.Errorf("CommandTitle输出应该包含::前缀，但得到: %s", w.String())
	}
}

func TestGetLogger(t *testing.T) {
	logger := GetLogger()

	if logger == nil {
		t.Error("GetLogger应该返回非nil的Logger")
	}

	// 确保返回的是*slog.Logger类型
	if _, ok := interface{}(logger).(*slog.Logger); !ok {
		t.Errorf("GetLogger应该返回*slog.Logger类型，但得到: %T", logger)
	}
}

func TestIsDebugEnabled(t *testing.T) {
	_, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	// 默认级别（Info）
	if IsDebugEnabled() {
		t.Error("默认情况下，IsDebugEnabled应该返回false")
	}

	// 启用Debug
	EnableDebug()

	if !IsDebugEnabled() {
		t.Error("启用Debug后，IsDebugEnabled应该返回true")
	}

	// 启用静默模式
	EnableSilence()

	if IsDebugEnabled() {
		t.Error("静默模式下，IsDebugEnabled应该返回false")
	}
}

// TestCommandHelp 测试 CommandHelp 函数返回黄色高亮的文本
func TestCommandHelp(t *testing.T) {
	result := CommandHelp("帮助文本")

	// CommandHelp 应该返回包含原始文本的字符串
	if !strings.Contains(result, "帮助文本") {
		t.Errorf("CommandHelp输出应该包含原始文本，但得到: %s", result)
	}

	// 验证 CommandHelp 使用了 Yellow 函数（非空结果）
	if result == "" {
		t.Error("CommandHelp不应该返回空字符串")
	}
}

// TestFormatLogMessage 测试 formatLogMessage 内部函数
func TestFormatLogMessage(t *testing.T) {
	// 测试奇数个参数（覆盖 i+1 < len(args) 为 false 的分支）
	result := formatLogMessage(InfoLevel, "消息", "key1", "value1", "oddkey")
	if !strings.Contains(result, "消息") {
		t.Errorf("formatLogMessage输出应该包含消息，但得到: %s", result)
	}
	// 奇数参数的最后一个键不应形成 key=value 对
	if strings.Contains(result, "oddkey=") {
		t.Errorf("奇数参数的最后一个键不应出现 key= 格式，但得到: %s", result)
	}
	// 正常的键值对应该存在
	if !strings.Contains(result, "key1=value1") {
		t.Errorf("formatLogMessage输出应该包含键值对，但得到: %s", result)
	}

	// 测试空参数
	resultEmpty := formatLogMessage(InfoLevel, "空参数消息")
	if !strings.Contains(resultEmpty, "空参数消息") {
		t.Errorf("formatLogMessage输出应该包含消息，但得到: %s", resultEmpty)
	}

	// 测试所有日志级别的标签
	tests := []struct {
		name       string
		level      charm.Level
		wantLabel  string
	}{
		{name: "Debug级别标签", level: DebugLevel, wantLabel: "DEBG"},
		{name: "Info级别标签", level: InfoLevel, wantLabel: "INFO"},
		{name: "Warn级别标签", level: WarnLevel, wantLabel: "WARN"},
		{name: "Error级别标签", level: ErrorLevel, wantLabel: "ERRO"},
		{name: "Fatal级别标签", level: FatalLevel, wantLabel: "FATL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLogMessage(tt.level, "消息")
			if !strings.Contains(result, tt.wantLabel) {
				t.Errorf("formatLogMessage输出应该包含级别标签 %s，但得到: %s", tt.wantLabel, result)
			}
		})
	}
}

// TestSlogToCharmEnabled 测试 slogToCharm.Enabled 方法
func TestSlogToCharmEnabled(t *testing.T) {
	_, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	handler := &slogToCharm{}
	ctx := context.Background()

	// 默认级别是 Info
	tests := []struct {
		name   string
		level  slog.Level
		expect bool
	}{
		{name: "Debug级别 - 默认不启用", level: slog.LevelDebug, expect: false},
		{name: "Info级别 - 默认启用", level: slog.LevelInfo, expect: true},
		{name: "Warn级别 - 默认启用", level: slog.LevelWarn, expect: true},
		{name: "Error级别 - 默认启用", level: slog.LevelError, expect: true},
		{name: "自定义级别(低于Debug) - default分支返回true", level: slog.Level(-10), expect: true},
		{name: "自定义级别(高于Error) - default分支返回true", level: slog.Level(20), expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.Enabled(ctx, tt.level)
			if result != tt.expect {
				t.Errorf("Enabled(%v) = %v, 期望 %v", tt.level, result, tt.expect)
			}
		})
	}
}

// TestSlogToCharmHandle 测试 slogToCharm.Handle 方法
func TestSlogToCharmHandle(t *testing.T) {
	w, origLogger := setupTestLogger()
	defer resetLogger(origLogger)

	handler := &slogToCharm{}
	ctx := context.Background()

	tests := []struct {
		name   string
		level  slog.Level
	}{
		{name: "处理Debug级别记录", level: slog.LevelDebug},
		{name: "处理Info级别记录", level: slog.LevelInfo},
		{name: "处理Warn级别记录", level: slog.LevelWarn},
		{name: "处理Error级别记录", level: slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.Reset()

			// Debug级别需要先启用
			if tt.level == slog.LevelDebug {
				EnableDebug()
			}

			record := slog.Record{
				Level:  tt.level,
				Message: "slog消息",
			}

			err := handler.Handle(ctx, record)
			if err != nil {
				t.Errorf("Handle不应该返回错误，但得到: %v", err)
			}

			// Debug级别在默认Info级别下不会输出
			if tt.level == slog.LevelDebug {
				if w.String() == "" {
					t.Errorf("Handle应该在Debug级别启用后输出日志")
				}
			} else {
				if w.String() == "" {
					t.Errorf("Handle应该在%s级别输出日志", tt.level.String())
				}
				if !strings.Contains(w.String(), "slog消息") {
					t.Errorf("Handle输出应该包含消息，但得到: %s", w.String())
				}
			}
		})
	}

	// 测试带有属性的记录
	t.Run("带属性的记录", func(t *testing.T) {
		w.Reset()
		record := slog.Record{
			Level:   slog.LevelInfo,
			Message: "带属性消息",
		}
		record.AddAttrs(slog.String("attr1", "val1"), slog.Int("attr2", 42))

		err := handler.Handle(ctx, record)
		if err != nil {
			t.Errorf("Handle不应该返回错误，但得到: %v", err)
		}
		if !strings.Contains(w.String(), "带属性消息") {
			t.Errorf("Handle输出应该包含消息，但得到: %s", w.String())
		}
	})
}

// TestSlogToCharmWithAttrs 测试 slogToCharm.WithAttrs 方法
func TestSlogToCharmWithAttrs(t *testing.T) {
	handler := &slogToCharm{}
	attrs := []slog.Attr{slog.String("key", "value")}

	result := handler.WithAttrs(attrs)

	// WithAttrs 应该返回同一个处理程序
	if result != handler {
		t.Errorf("WithAttrs应该返回同一个处理程序，但得到: %v", result)
	}
}

// TestSlogToCharmWithGroup 测试 slogToCharm.WithGroup 方法
func TestSlogToCharmWithGroup(t *testing.T) {
	handler := &slogToCharm{}

	result := handler.WithGroup("testgroup")

	// WithGroup 应该返回同一个处理程序
	if result != handler {
		t.Errorf("WithGroup应该返回同一个处理程序，但得到: %v", result)
	}
}

// TestFatal 测试 Fatal 函数
// 使用子进程模式来测试 os.Exit(1) 的行为
func TestFatal(t *testing.T) {
	if os.Getenv("BE_FATAL") == "1" {
		w, _ := setupTestLogger()
		Fatal("fatal message", "key", "value")
		// Fatal应该已经退出，如果到这里说明没有退出，需要输出一些内容以便父进程检测
		if w.String() == "" {
			os.Exit(0) // 不应该到这里
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "BE_FATAL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return // 期望的行为 - 以非零状态退出
	}
	t.Fatalf("Fatal应该以非零状态退出，但得到: %v", err)
}
