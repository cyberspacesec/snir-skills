package sdk

import (
	"fmt"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

var autoConnect = func(opts *runner.Options, maxConcurrent int) (driverPool, string, error) {
	return runner.AutoConnect(opts, maxConcurrent)
}

// AutoConnectMode 自动连接模式返回的连接类型
type AutoConnectMode string

const (
	// AutoConnectRemote 连接到指定的远程 Chrome
	AutoConnectRemote AutoConnectMode = "remote"
	// AutoConnectDiscovered 自动发现并连接本地 Chrome
	AutoConnectDiscovered AutoConnectMode = "discovered"
	// AutoConnectLocal 启动新的本地 Chrome 进程
	AutoConnectLocal AutoConnectMode = "local"
)

// AutoConnectClient 创建一个自动连接的截图客户端
// 优先级：
// 1. 如果 ClientOptions.WSSURL 已设置，连接到该远程 Chrome
// 2. 尝试发现本地已运行的 Chrome 实例（扫描 9222/9223/9224 端口）
// 3. 如果都没有，启动新的本地 Chrome 进程
//
// 这样，在大系统中：
// - 如果已有 Provider 服务运行，自动发现并连接
// - 如果没有，自动启动本地 Chrome，不影响功能
// - 用户也可以显式指定 WSSURL 来强制使用特定实例
func AutoConnectClient(opts ClientOptions) (*Client, AutoConnectMode, error) {
	runnerOpts := toRunnerOptions(opts)

	pool, mode, err := autoConnect(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		return nil, "", fmt.Errorf("自动连接失败: %v", err)
	}

	modeStr := AutoConnectMode(mode)
	switch modeStr {
	case AutoConnectRemote:
		log.Info("截图SDK客户端已连接到远程Chrome", "ws_url", opts.WSSURL, "max_concurrent", opts.MaxConcurrent)
	case AutoConnectDiscovered:
		log.Info("截图SDK客户端已自动发现并连接本地Chrome", "max_concurrent", opts.MaxConcurrent)
	case AutoConnectLocal:
		log.Info("截图SDK客户端已启动本地Chrome", "max_concurrent", opts.MaxConcurrent)
	}

	return &Client{
		pool: pool,
		opts: opts,
	}, modeStr, nil
}
