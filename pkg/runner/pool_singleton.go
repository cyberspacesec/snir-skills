package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// 全局单例池
var (
	globalPool     *DriverPool
	globalPoolOnce sync.Once
	globalPoolMu   sync.RWMutex
)

// InitSharedPool 初始化进程级共享池
// 使用 sync.Once 保证只初始化一次
// 必须在使用 GetSharedPool() 之前调用
// 如果未调用，GetSharedPool() 会使用默认配置自动初始化
func InitSharedPool(opts *Options, maxConcurrent int) error {
	var initErr error
	globalPoolOnce.Do(func() {
		pool, err := NewDriverPool(opts, maxConcurrent)
		if err != nil {
			initErr = err
			return
		}

		globalPoolMu.Lock()
		globalPool = pool
		globalPoolMu.Unlock()

		log.Info("全局共享浏览器连接池已初始化", "max_concurrent", maxConcurrent)
	})
	return initErr
}

// GetSharedPool 获取进程级共享池
// 如果尚未初始化，使用默认配置自动创建
// 多个包/模块 import 后自动复用同一池
func GetSharedPool() (*DriverPool, error) {
	globalPoolMu.RLock()
	if globalPool != nil {
		p := globalPool
		globalPoolMu.RUnlock()
		return p, nil
	}
	globalPoolMu.RUnlock()

	// 自动初始化（使用默认配置）
	var initErr error
	globalPoolOnce.Do(func() {
		defaultOpts := &Options{}
		defaultOpts.Chrome.Headless = true
		defaultOpts.Chrome.WindowX = 1280
		defaultOpts.Chrome.WindowY = 800
		defaultOpts.Chrome.Timeout = 30
		defaultOpts.Scan.ScreenshotPath = "screenshots"
		defaultOpts.Scan.ScreenshotFormat = "png"

		pool, err := NewDriverPool(defaultOpts, 4)
		if err != nil {
			initErr = err
			return
		}

		globalPoolMu.Lock()
		globalPool = pool
		globalPoolMu.Unlock()

		log.Info("全局共享浏览器连接池已自动初始化（默认配置）", "max_concurrent", 4)
	})

	if initErr != nil {
		return nil, fmt.Errorf("初始化共享池失败: %v", initErr)
	}

	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()
	return globalPool, nil
}

// GetSharedPoolWithConfig 获取共享池，如果尚未初始化则使用指定配置
// 已初始化则返回现有池（忽略配置参数）
func GetSharedPoolWithConfig(opts *Options, maxConcurrent int) (*DriverPool, error) {
	globalPoolMu.RLock()
	if globalPool != nil {
		p := globalPool
		globalPoolMu.RUnlock()
		return p, nil
	}
	globalPoolMu.RUnlock()

	err := InitSharedPool(opts, maxConcurrent)
	if err != nil {
		return nil, err
	}

	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()
	return globalPool, nil
}

// CloseSharedPool 关闭共享池
// 通常在程序退出时调用
func CloseSharedPool() {
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	if globalPool != nil {
		globalPool.Close()
		globalPool = nil
		log.Info("全局共享浏览器连接池已关闭")
	}
}

// SharedPoolStats 返回共享池的统计信息
func SharedPoolStats() (PoolStats, error) {
	pool, err := GetSharedPool()
	if err != nil {
		return PoolStats{}, err
	}
	return pool.Stats(), nil
}

// SharedScreenshot 使用共享池执行截图
// 便捷方法，无需手动获取池实例
func SharedScreenshot(target string, opts *Options) (*models.Result, error) {
	return SharedScreenshotWithContext(context.Background(), target, opts)
}

// SharedScreenshotWithContext 使用共享池执行截图（支持取消）
func SharedScreenshotWithContext(ctx context.Context, target string, opts *Options) (*models.Result, error) {
	pool, err := GetSharedPool()
	if err != nil {
		return nil, fmt.Errorf("获取共享池失败: %v", err)
	}
	return pool.ScreenshotWithContext(ctx, target, opts)
}

// SharedSetIdleTimeout 设置共享池的空闲超时
func SharedSetIdleTimeout(timeout time.Duration) error {
	pool, err := GetSharedPool()
	if err != nil {
		return err
	}
	pool.SetIdleTimeout(timeout)
	return nil
}
