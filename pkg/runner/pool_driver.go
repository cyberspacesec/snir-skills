package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// PoolDriver 适配 DriverPool 到 Driver 接口
// 让 Runner/Scanner 能通过标准 Driver 接口使用 DriverPool
//
// 与直接使用 ChromeDP 的区别：
//   - ChromeDP：每次 Witness 都在同一个 Chrome 标签页中操作（单线程）
//   - PoolDriver：每次 Witness 从池中获取新标签页，支持并发截图
//   - PoolDriver：内置智能重试，浏览器进程错误自动重建，网络临时错误指数退避重试
type PoolDriver struct {
	pool *DriverPool
	opts *Options
}

// NewPoolDriver 创建一个基于 DriverPool 的 Driver 实现
// opts: 用于初始化池的配置
// maxConcurrent: 最大并发截图数
func NewPoolDriver(opts *Options, maxConcurrent int) (*PoolDriver, error) {
	pool, err := NewDriverPool(opts, maxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("创建连接池驱动失败: %v", err)
	}

	return &PoolDriver{
		pool: pool,
		opts: opts,
	}, nil
}

// isRetriableError 判断错误是否值得重试
// 返回 true 表示可以重试，false 表示应立即放弃
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()

	// 不可重试的错误：DNS 解析失败、连接被拒绝（目标不存在）
	nonRetriable := []string{
		"net::ERR_NAME_NOT_RESOLVED",
		"net::ERR_CONNECTION_REFUSED",
		"net::ERR_ADDRESS_UNREACHABLE",
		"net::ERR_ACCESS_DENIED",
	}
	for _, nr := range nonRetriable {
		if strings.Contains(msg, nr) {
			return false
		}
	}

	// 可重试的错误：网络临时故障、浏览器进程问题
	retriable := []string{
		"net::ERR_CONNECTION_RESET",
		"net::ERR_CONNECTION_TIMED_OUT",
		"net::ERR_TIMED_OUT",
		"net::ERR_CONNECTION_CLOSED",
		"net::ERR_NETWORK_CHANGED",
		"net::ERR_INTERNET_DISCONNECTED",
		"Could not find node with given id",
		"context deadline exceeded",
		"timeout",
		"浏览器进程不可用",
		"截图取消",
	}
	for _, r := range retriable {
		if strings.Contains(msg, r) {
			return true
		}
	}

	// 默认：未知错误不重试
	return false
}

// Witness 实现 Driver 接口
// 通过 DriverPool 执行截图，在共享的 Chrome 进程中创建新标签页
// 内置智能重试：
//   - 可重试的网络临时错误：指数退避重试（最多 MaxRetries 次）
//   - 浏览器进程错误：自动重建进程后重试
//   - 不可重试错误（DNS 失败、连接被拒绝）：立即返回
func (d *PoolDriver) Witness(target string, opts *Options) (*models.Result, error) {
	screenshotOpts := opts
	if screenshotOpts == nil {
		screenshotOpts = d.opts
	}

	maxRetries := screenshotOpts.Scan.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var result *models.Result
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：2, 4, 6... 秒
			backoff := time.Duration(2*attempt) * time.Second
			log.Info(fmt.Sprintf("第 %d 次重试截图", attempt), "url", target, "backoff", backoff)
			time.Sleep(backoff)
		}

		ctx := context.Background()
		if screenshotOpts.Chrome.Timeout > 0 {
			var cancel context.CancelFunc
			// 重试时增加超时时间，给更多机会
			timeout := time.Duration(screenshotOpts.Chrome.Timeout)*time.Second + time.Duration(attempt)*5*time.Second
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		result, lastErr = d.pool.ScreenshotWithContext(ctx, target, screenshotOpts)

		// 成功
		if lastErr == nil {
			if result != nil && result.Failed {
				// 截图执行了但结果标记为失败，检查是否可重试
				if isRetriableError(fmt.Errorf("%s", result.FailedReason)) {
					lastErr = fmt.Errorf("%s", result.FailedReason)
					continue
				}
			}
			return result, nil
		}

		// 不可重试的错误，立即返回
		if !isRetriableError(lastErr) {
			return nil, lastErr
		}

		// 可重试错误，继续循环
		log.Warn("截图失败，准备重试", "url", target, "attempt", attempt+1, "error", lastErr)
	}

	// 所有重试都失败
	if lastErr != nil {
		return nil, fmt.Errorf("截图失败（已重试 %d 次）: %v", maxRetries, lastErr)
	}

	return result, nil
}

// Close 实现 Driver 接口
// 优雅关闭连接池
func (d *PoolDriver) Close() {
	d.pool.Close()
}

// Pool 返回底层 DriverPool，供需要高级功能（如 Stats、SetIdleTimeout）的调用方使用
func (d *PoolDriver) Pool() *DriverPool {
	return d.pool
}

// Stats 返回连接池统计信息
func (d *PoolDriver) Stats() PoolStats {
	return d.pool.Stats()
}

// SetIdleTimeout 设置空闲超时
func (d *PoolDriver) SetIdleTimeout(timeout time.Duration) {
	d.pool.SetIdleTimeout(timeout)
}

// On 注册池事件监听器
func (d *PoolDriver) On(handler PoolEventHandler) {
	d.pool.On(handler)
}

// logPoolStats 定期打印池统计信息（调试用）
func logPoolStats(driver *PoolDriver) {
	stats := driver.Stats()
	log.Debug("连接池状态",
		"active", stats.ActiveCount,
		"max_concurrent", stats.MaxConcurrent,
		"total", stats.TotalScreenshots,
		"failed", stats.FailedScreenshots,
		"reconnects", stats.ReconnectCount,
	)
}
