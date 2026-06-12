package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/models"
)

// PoolDriver 适配 DriverPool 到 Driver 接口
// 让 Runner/Scanner 能通过标准 Driver 接口使用 DriverPool
//
// 与直接使用 ChromeDP 的区别：
//   - ChromeDP：每次 Witness 都在同一个 Chrome 标签页中操作（单线程）
//   - PoolDriver：每次 Witness 从池中获取新标签页，支持并发截图
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

// Witness 实现 Driver 接口
// 通过 DriverPool 执行截图，在共享的 Chrome 进程中创建新标签页
func (d *PoolDriver) Witness(target string, opts *Options) (*models.Result, error) {
	screenshotOpts := opts
	if screenshotOpts == nil {
		screenshotOpts = d.opts
	}

	ctx := context.Background()
	if screenshotOpts.Chrome.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(screenshotOpts.Chrome.Timeout)*time.Second)
		defer cancel()
	}

	result, err := d.pool.ScreenshotWithContext(ctx, target, screenshotOpts)
	if err != nil {
		return nil, err
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
