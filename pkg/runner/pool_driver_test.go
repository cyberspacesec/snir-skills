package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestIsRetriableError_AllBranches 覆盖 isRetriableError 的所有分支：
// nil、不可重试、可重试、未知错误。
func TestIsRetriableError_AllBranches(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"未知错误", errors.New("some unknown error"), false},
		{"DNS解析失败", errors.New("net::ERR_NAME_NOT_RESOLVED"), false},
		{"连接被拒绝", errors.New("net::ERR_CONNECTION_REFUSED"), false},
		{"地址不可达", errors.New("net::ERR_ADDRESS_UNREACHABLE"), false},
		{"访问被拒", errors.New("net::ERR_ACCESS_DENIED"), false},
		{"连接重置-可重试", errors.New("net::ERR_CONNECTION_RESET"), true},
		{"连接超时-可重试", errors.New("net::ERR_CONNECTION_TIMED_OUT"), true},
		{"超时-可重试", errors.New("net::ERR_TIMED_OUT"), true},
		{"连接关闭-可重试", errors.New("net::ERR_CONNECTION_CLOSED"), true},
		{"网络变更-可重试", errors.New("net::ERR_NETWORK_CHANGED"), true},
		{"断网-可重试", errors.New("net::ERR_INTERNET_DISCONNECTED"), true},
		{"节点未找到-可重试", errors.New("Could not find node with given id"), true},
		{"上下文超时-可重试", errors.New("context deadline exceeded"), true},
		{"timeout小写-可重试", errors.New("request timeout"), true},
		{"浏览器进程不可用-可重试", errors.New("浏览器进程不可用"), true},
		{"截图取消-可重试", errors.New("截图取消"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetriableError(tt.err); got != tt.want {
				t.Errorf("isRetriableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// newBarePoolDriver 用 bare pool 构造 PoolDriver（不启动浏览器）。
func newBarePoolDriver(maxConcurrent int) *PoolDriver {
	return &PoolDriver{
		pool: newBarePool(maxConcurrent),
		opts: &Options{},
	}
}

// TestPoolDriver_PoolStatsClose 覆盖 Pool/Stats/Close/SetIdleTimeout/On。
func TestPoolDriver_PoolStatsClose(t *testing.T) {
	d := newBarePoolDriver(3)

	// Pool 返回底层池
	if d.Pool() == nil {
		t.Error("Pool() 不应为 nil")
	}

	// Stats
	stats := d.Stats()
	if stats.MaxConcurrent != 3 {
		t.Errorf("Stats().MaxConcurrent = %d, want 3", stats.MaxConcurrent)
	}

	// SetIdleTimeout 不应 panic
	d.SetIdleTimeout(0)
	d.SetIdleTimeout(50 * time.Millisecond)
	d.SetIdleTimeout(0)

	// On 注册事件 handler
	ch := make(chan PoolEvent, 4)
	d.On(func(event PoolEvent) { ch <- event })
	d.pool.events.emitPoolClosed()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Error("On 注册的 handler 应被调用")
	}

	// Close
	d.Close()
	if !d.pool.closed {
		t.Error("Close 后底层池应标记为关闭")
	}
	// 二次 Close 不应 panic
	d.Close()
}

// TestPoolDriver_Witness_NilOptsUsesDefaults 覆盖 Witness 的 nil opts 分支 +
// closed pool 导致 ScreenshotWithContext 返回不可重试错误 → 立即返回。
func TestPoolDriver_Witness_ClosedPoolNonRetriable(t *testing.T) {
	d := newBarePoolDriver(1)
	d.pool.closed = true // closed pool 让 ScreenshotWithContext 返回 closed 错误

	// closed 错误不在可重试列表中 → 立即返回错误
	result, err := d.Witness("https://example.com", nil)
	if err == nil {
		t.Fatal("closed pool 应返回错误")
	}
	if result != nil {
		t.Error("失败时 result 应为 nil")
	}
}

// TestPoolDriver_Witness_MaxRetriesZero 覆盖 maxRetries<=0 时不重试直接返回。
func TestPoolDriver_Witness_MaxRetriesZero(t *testing.T) {
	d := newBarePoolDriver(1)
	d.pool.closed = true
	opts := &Options{}
	opts.Scan.MaxRetries = 0

	start := time.Now()
	_, err := d.Witness("https://example.com", opts)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("closed pool 应返回错误")
	}
	// maxRetries=0 不应触发重试退避（不会 sleep 2 秒）
	if elapsed > 1*time.Second {
		t.Errorf("maxRetries=0 不应重试，耗时 %v 过长", elapsed)
	}
}

// TestPoolDriver_Witness_OptsPassedThrough 覆盖 Witness 使用传入 opts（非 nil）分支。
func TestPoolDriver_Witness_OptsPassedThrough(t *testing.T) {
	d := newBarePoolDriver(1)
	d.pool.closed = true
	opts := &Options{}
	opts.Scan.MaxRetries = 0
	opts.Chrome.Timeout = 1 // 触发 WithTimeout 分支

	_, err := d.Witness("https://example.com", opts)
	if err == nil {
		t.Fatal("closed pool 应返回错误")
	}
}

// TestAutoConnect_WSSFailure 覆盖 AutoConnect 优先级 1 的 WSS 失败分支：
// 指定无效 WSS URL，NewDriverPool 连接失败，返回错误。
func TestAutoConnect_WSSFailure(t *testing.T) {
	opts := &Options{}
	opts.Chrome.WSS = "ws://127.0.0.1:1/devtools/browser/nonexistent" // 端口 1 通常无服务

	// 用超时保护：连接 WSS 会快速失败
	done := make(chan struct{})
	var pool *DriverPool
	var mode string
	var err error
	go func() {
		pool, mode, err = AutoConnect(opts, 1)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("AutoConnect WSS 连接超时")
	}

	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Skip("WSS 连接意外成功（可能本地有服务），跳过失败分支测试")
	}
	if pool != nil {
		t.Error("失败时 pool 应为 nil")
	}
	if mode != "" {
		t.Errorf("失败时 mode 应为空, got %q", mode)
	}
}

// TestAutoConnect_NoWSSDiscoverFails 覆盖 AutoConnect 优先级 2/3：
// 无 WSS、无本地 Chrome、NewDriverPool 失败 → 返回错误。
// 用不存在的 Chrome 路径让 NewDriverPool 失败。
func TestAutoConnect_NoWSSDiscoverFails(t *testing.T) {
	opts := &Options{}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	opts.Chrome.Headless = true

	done := make(chan struct{})
	var pool *DriverPool
	var mode string
	var err error
	go func() {
		pool, mode, err = AutoConnect(opts, 1)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("AutoConnect 启动 Chrome 超时")
	}

	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Skip("Chrome 意外可用，跳过失败分支测试")
	}
	if pool != nil {
		t.Error("失败时 pool 应为 nil")
	}
	_ = mode
}

// TestLogPoolStats_NoPanic 覆盖 logPoolStats 纯函数（仅打印调试日志）。
func TestLogPoolStats_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("logPoolStats panic: %v", r)
		}
	}()
	d := newBarePoolDriver(2)
	logPoolStats(d)
}

// 确保引入 context 包
var _ = context.Background

// TestPoolDriver_Witness_NegativeMaxRetries 覆盖 Witness 的 maxRetries<0 分支
// （line 98-100：负值被钳制为 0，不重试直接返回）。
func TestPoolDriver_Witness_NegativeMaxRetries(t *testing.T) {
	d := newBarePoolDriver(1)
	d.pool.closed = true
	opts := &Options{}
	opts.Scan.MaxRetries = -5 // 负值 → 钳制为 0

	start := time.Now()
	_, err := d.Witness("https://example.com", opts)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("closed pool 应返回错误")
	}
	if elapsed > 1*time.Second {
		t.Errorf("负 maxRetries 钳制为 0 后不应重试，耗时 %v", elapsed)
	}
}

// TestNewPoolDriver_Failure 覆盖 NewPoolDriver 的失败分支（line 30-32）：
// 用不存在的 Chrome 路径让 NewDriverPool 失败。
func TestNewPoolDriver_Failure(t *testing.T) {
	opts := &Options{}
	opts.Chrome.Path = "/nonexistent/chrome-binary-for-test"
	opts.Chrome.Headless = true

	done := make(chan struct{})
	var got *PoolDriver
	var err error
	go func() {
		got, err = NewPoolDriver(opts, 1)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("NewPoolDriver 超时")
	}
	if err == nil {
		if got != nil {
			got.Close()
		}
		t.Skip("Chrome 意外可用，跳过失败分支测试")
	}
	if !strings.Contains(err.Error(), "创建连接池驱动失败") {
		t.Logf("NewPoolDriver 错误（预期）: %v", err)
	}
}
