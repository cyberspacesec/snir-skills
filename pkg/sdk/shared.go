package sdk

import (
	"context"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

var (
	sharedScreenshotWithContext = runner.SharedScreenshotWithContext
	sharedSetIdleTimeout        = runner.SharedSetIdleTimeout
	sharedPoolStats             = runner.SharedPoolStats
	closeSharedPool             = runner.CloseSharedPool
)

// SharedScreenshot 使用进程级共享池执行截图
// 多个包/模块调用此函数会自动复用同一个 Chrome 进程
// 首次调用时自动初始化池，后续调用直接复用
func SharedScreenshot(url string, opts *ScreenshotOptions) (*models.Result, error) {
	return SharedScreenshotWithContext(context.Background(), url, opts)
}

// SharedScreenshotWithContext 使用共享池执行截图（支持取消）
func SharedScreenshotWithContext(ctx context.Context, url string, opts *ScreenshotOptions) (*models.Result, error) {
	runnerOpts := defaultRunnerOptions()
	runnerOpts = mergeWithScreenshotOptions(runnerOpts, opts)
	appendCookieSources(url, &runnerOpts)
	if result, err := blacklistedResult(url, &runnerOpts); err != nil {
		return nil, err
	} else if result != nil {
		return result, nil
	}

	result, err := sharedScreenshotWithContext(ctx, url, &runnerOpts)
	if err != nil {
		return nil, err
	}

	if result != nil && len(result.Cookies) > 0 && runnerOpts.Scan.CookieExport != "" {
		if err := runner.ExportResultCookiesToNetscape(runnerOpts.Scan.CookieExport, result.Cookies, url); err != nil {
			log.Warn("SDK: 导出 Netscape Cookie 失败", "file", runnerOpts.Scan.CookieExport, "error", err)
		}
	}

	if result.Failed {
		return result, nil
	}

	return result, nil
}

// SharedSetIdleTimeout 设置共享池的空闲超时
func SharedSetIdleTimeout(timeout time.Duration) {
	if err := sharedSetIdleTimeout(timeout); err != nil {
		log.Error("设置共享池空闲超时失败", "error", err)
	}
}

// SharedStats 返回共享池统计信息
func SharedStats() (runner.PoolStats, error) {
	return sharedPoolStats()
}

// CloseSharedPool 关闭共享池
// 通常在程序退出时调用（可用 defer）
func CloseSharedPool() {
	closeSharedPool()
}

// defaultRunnerOptions 返回默认 runner 配置
func defaultRunnerOptions() runner.Options {
	opts := runner.Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = "screenshots"
	opts.Scan.ScreenshotFormat = "png"
	return opts
}
