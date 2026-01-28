package log

import (
	"bytes"
	"log/slog"
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
