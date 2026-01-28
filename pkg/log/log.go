package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	charm "github.com/charmbracelet/log"
	"github.com/fatih/color"
)

// 颜色函数定义
// 这些函数用于在终端输出彩色文本
var (
	// 柔和的颜色，用于非强调内容
	softBlue   = color.New(color.FgBlue).Add(color.Faint).SprintFunc()
	softGreen  = color.New(color.FgHiGreen).Add(color.Faint).SprintFunc()
	softYellow = color.New(color.FgHiYellow).Add(color.Faint).SprintFunc()
	softRed    = color.New(color.FgHiRed).SprintFunc()
	softGray   = color.New(color.FgHiBlack).SprintFunc()

	// 常规颜色文本函数，用于强调内容
	Blue   = color.New(color.FgBlue).SprintFunc()
	Green  = color.New(color.FgGreen).SprintFunc()
	Red    = color.New(color.FgRed).SprintFunc()
	Yellow = color.New(color.FgYellow).SprintFunc()
	Cyan   = color.New(color.FgCyan).SprintFunc()
	White  = color.New(color.FgWhite).SprintFunc()
	Bold   = color.New(color.Bold).SprintFunc()
)

// 日志级别常量，与 charmbracelet/log 包保持一致
const (
	DebugLevel = charm.DebugLevel // 调试级别，最详细
	InfoLevel  = charm.InfoLevel  // 信息级别，常规信息
	WarnLevel  = charm.WarnLevel  // 警告级别，潜在问题
	ErrorLevel = charm.ErrorLevel // 错误级别，运行时错误
	FatalLevel = charm.FatalLevel // 致命级别，严重错误，会导致程序退出
)

// Logger 自定义日志记录器结构体
// 封装日志级别和输出目标
type Logger struct {
	level  charm.Level // 日志级别
	writer io.Writer   // 输出目标
}

// 全局日志记录器实例和相关变量
var (
	// 默认日志记录器，初始级别为 Info，输出到标准错误
	defaultLogger = &Logger{
		level:  InfoLevel,
		writer: os.Stderr,
	}

	// SlogHandler 是一个 slog.Handler，使用我们的日志记录器
	SlogHandler = &slogToCharm{}

	// slogLogger 是一个 slog.Logger 实例，用于与 slog 兼容的日志记录
	slogLogger = slog.New(SlogHandler)
)

// formatLogMessage 格式化日志消息
// 将日志级别、消息内容和键值对参数组合为格式化的字符串
// 参数:
//   - level: 日志级别
//   - msg: 日志消息
//   - args: 键值对参数，按 key1, value1, key2, value2... 的顺序传入
//
// 返回:
//   - 格式化的日志消息字符串
func formatLogMessage(level charm.Level, msg string, args ...interface{}) string {
	// 格式化时间戳
	timestamp := softGray(time.Now().Format(time.Kitchen))

	// 根据日志级别选择不同的标签颜色
	var levelLabel string
	switch level {
	case DebugLevel:
		levelLabel = softGray("DEBG")
	case InfoLevel:
		levelLabel = softGray("INFO")
	case WarnLevel:
		levelLabel = softYellow("WARN")
	case ErrorLevel:
		levelLabel = softRed("ERRO")
	case FatalLevel:
		levelLabel = softRed("FATL")
	}

	// 构建日志前缀，包含时间戳和级别标签
	prefix := fmt.Sprintf("%s %s", timestamp, levelLabel)

	// 处理键值对参数
	var keyValues strings.Builder
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			keyValues.WriteString(" ")
			key := softGray(fmt.Sprintf("%v=", args[i]))
			value := fmt.Sprintf("%v", args[i+1])
			keyValues.WriteString(key)
			keyValues.WriteString(value)
		}
	}

	// 组合完整日志消息：前缀 + 消息 + 键值对
	return fmt.Sprintf("%s %s%s\n", prefix, msg, keyValues.String())
}

// Debug 记录调试级别的消息
// 仅当当前日志级别为 Debug 或更低时才记录
// 参数:
//   - msg: 日志消息
//   - args: 键值对参数
func Debug(msg string, args ...interface{}) {
	if defaultLogger.level <= DebugLevel {
		fmt.Fprint(defaultLogger.writer, formatLogMessage(DebugLevel, msg, args...))
	}
}

// Info 记录信息级别的消息
// 仅当当前日志级别为 Info 或更低时才记录
// 参数:
//   - msg: 日志消息
//   - args: 键值对参数
func Info(msg string, args ...interface{}) {
	if defaultLogger.level <= InfoLevel {
		fmt.Fprint(defaultLogger.writer, formatLogMessage(InfoLevel, msg, args...))
	}
}

// Warn 记录警告级别的消息
// 仅当当前日志级别为 Warn 或更低时才记录
// 参数:
//   - msg: 日志消息
//   - args: 键值对参数
func Warn(msg string, args ...interface{}) {
	if defaultLogger.level <= WarnLevel {
		fmt.Fprint(defaultLogger.writer, formatLogMessage(WarnLevel, msg, args...))
	}
}

// Error 记录错误级别的消息
// 仅当当前日志级别为 Error 或更低时才记录
// 参数:
//   - msg: 日志消息
//   - args: 键值对参数
func Error(msg string, args ...interface{}) {
	if defaultLogger.level <= ErrorLevel {
		fmt.Fprint(defaultLogger.writer, formatLogMessage(ErrorLevel, msg, args...))
	}
}

// Fatal 记录致命级别的消息并退出程序
// 仅当当前日志级别为 Fatal 或更低时才记录
// 记录后程序将以状态码 1 退出
// 参数:
//   - msg: 日志消息
//   - args: 键值对参数
func Fatal(msg string, args ...interface{}) {
	if defaultLogger.level <= FatalLevel {
		fmt.Fprint(defaultLogger.writer, formatLogMessage(FatalLevel, msg, args...))
		os.Exit(1)
	}
}

// Success 打印带有绿色✓标记的成功消息
// 使用 Info 级别记录，但添加了成功标记
// 参数:
//   - msg: 日志消息
//   - args: 键值对参数
func Success(msg string, args ...interface{}) {
	Info(Green("✓ "+msg), args...)
}

// CommandTitle 打印带有蓝色::前缀的命令标题
// 用于标记新命令的开始
// 参数:
//   - title: 命令标题
func CommandTitle(title string) {
	Info(Bold(Blue(":: " + title)))
}

// CommandHelp 为命令帮助文本添加黄色高亮
// 用于格式化命令帮助信息
// 参数:
//   - text: 帮助文本
//
// 返回:
//   - 格式化后的彩色帮助文本
func CommandHelp(text string) string {
	return Yellow(text)
}

// EnableDebug 启用调试级别日志
// 调用此函数后，所有级别的日志都将被记录
func EnableDebug() {
	defaultLogger.level = DebugLevel
}

// EnableSilence 启用静默模式，禁用大部分日志
// 调用此函数后，只有 Fatal 级别的日志会被记录
func EnableSilence() {
	defaultLogger.level = FatalLevel
}

// IsDebugEnabled 检查是否启用了调试日志
// 返回:
//   - 如果调试日志已启用则返回 true，否则返回 false
func IsDebugEnabled() bool {
	return defaultLogger.level <= DebugLevel
}

// GetLogger 返回 slog.Logger 实例
// 用于与 slog 兼容的日志记录
// 返回:
//   - 当前配置的 slog.Logger 实例
func GetLogger() *slog.Logger {
	return slogLogger
}

// slogToCharm 是一个 slog.Handler 实现
// 将 slog 日志适配到我们的自定义日志系统
type slogToCharm struct{}

// Enabled 检查给定级别的日志是否启用
// 参数:
//   - ctx: 上下文
//   - level: 日志级别
//
// 返回:
//   - 如果给定级别的日志已启用则返回 true，否则返回 false
func (h *slogToCharm) Enabled(ctx context.Context, level slog.Level) bool {
	switch level {
	case slog.LevelDebug:
		return defaultLogger.level <= DebugLevel
	case slog.LevelInfo:
		return defaultLogger.level <= InfoLevel
	case slog.LevelWarn:
		return defaultLogger.level <= WarnLevel
	case slog.LevelError:
		return defaultLogger.level <= ErrorLevel
	default:
		return true
	}
}

// Handle 处理 slog 日志记录请求
// 将 slog 日志适配到我们的自定义日志系统
// 参数:
//   - ctx: 上下文
//   - r: 日志记录
//
// 返回:
//   - 错误（如果有）
func (h *slogToCharm) Handle(ctx context.Context, r slog.Record) error {
	// 将 slog 级别转换为我们的日志级别
	level := InfoLevel
	switch r.Level {
	case slog.LevelDebug:
		level = DebugLevel
	case slog.LevelInfo:
		level = InfoLevel
	case slog.LevelWarn:
		level = WarnLevel
	case slog.LevelError:
		level = ErrorLevel
	}

	// 收集属性
	attrs := make([]interface{}, 0, r.NumAttrs()*2)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a.Key, a.Value.Any())
		return true
	})

	// 根据级别调用相应的日志函数
	switch level {
	case DebugLevel:
		Debug(r.Message, attrs...)
	case InfoLevel:
		Info(r.Message, attrs...)
	case WarnLevel:
		Warn(r.Message, attrs...)
	case ErrorLevel:
		Error(r.Message, attrs...)
	}

	return nil
}

// WithAttrs 返回一个带有附加属性的处理程序
// 此实现简单返回自身，因为我们在 Handle 中直接处理属性
// 参数:
//   - attrs: 附加属性
//
// 返回:
//   - 带有附加属性的处理程序
func (h *slogToCharm) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup 返回一个带有属性分组的处理程序
// 此实现简单返回自身，因为我们不支持分组
// 参数:
//   - name: 分组名称
//
// 返回:
//   - 带有分组的处理程序
func (h *slogToCharm) WithGroup(name string) slog.Handler {
	return h
}
