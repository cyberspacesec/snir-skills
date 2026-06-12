package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/provider"
)

// Provider 命令的本地变量（避免与全局 opts 冲突）
var (
	providerPort          int
	providerChromePort    int
	providerMaxConcurrent int
	providerIdleTimeout   time.Duration
	providerChromePath    string
	providerUserAgent     string
	providerProxy         string
	providerHeadless      bool
	providerIgnoreCerts   bool
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: log.Yellow("启动CDP Provider服务"),
	Long: log.Yellow(`启动一个CDP Provider服务，共享Chrome实例给其他工具使用。

在大系统中，多个工具可能都需要Chrome/CDP能力。
Provider启动一个Chrome实例并暴露WebSocket URL，
其他工具通过连接Provider来共享同一个Chrome实例。

架构示例:
  ┌──────────────────────┐
  │   大系统 (多个工具)    │
  │  ┌────────┐ ┌──────┐ │
  │  │工具 A  │ │工具B │ │
  │  └───┬────┘ └──┬───┘ │
  │      │         │     │
  │      ▼         ▼     │
  │  ┌──────────────────┐ │
  │  │  CDP Provider    │ │
  │  │  (1个Chrome实例) │ │
  │  └──────────────────┘ │
  └──────────────────────┘

其他工具连接方式:
  # Go SDK
  client, _ := sdk.NewRemoteClient("ws://provider-host:9222/devtools/browser/xxx", 4)

  # 查询WebSocket URL
  curl http://provider-host:9223/ws

  # 自动发现
  client, mode, _ := sdk.AutoConnectClient(sdk.DefaultClientOptions())`),
	Example: `  # 启动Provider（默认端口9223，Chrome调试端口9222）
  ./snir provider

  # 自定义端口
  ./snir provider --port 8090 --chrome-port 9222

  # 非无头模式（方便调试）
  ./snir provider --no-headless

  # 设置最大并发和空闲超时
  ./snir provider --max-concurrent 20 --idle-timeout 10m

  # 使用代理和自定义Chrome路径
  ./snir provider --chrome-path /usr/bin/chromium --proxy http://127.0.0.1:8080

  # 忽略证书错误
  ./snir provider --ignore-cert-errors`,
	RunE: func(cmd *cobra.Command, args []string) error {
		providerOpts := provider.DefaultProviderOptions()

		// 从命令行参数读取
		providerOpts.Port = providerPort
		providerOpts.ChromeDebugPort = providerChromePort
		providerOpts.MaxConcurrent = providerMaxConcurrent
		providerOpts.IdleTimeout = providerIdleTimeout
		providerOpts.ChromePath = providerChromePath
		providerOpts.Headless = providerHeadless
		providerOpts.UserAgent = providerUserAgent
		providerOpts.Proxy = providerProxy
		providerOpts.IgnoreCertErrors = providerIgnoreCerts

		// 创建 Provider
		p := provider.NewProvider(providerOpts)

		log.CommandTitle("启动CDP Provider服务")
		log.Info("服务地址", "host", log.Cyan(providerOpts.Host), "port", log.Cyan(fmt.Sprintf("%d", providerOpts.Port)))
		log.Info("Chrome调试端口", "port", log.Cyan(fmt.Sprintf("%d", providerOpts.ChromeDebugPort)))
		log.Info("最大并发", "max_concurrent", log.Cyan(fmt.Sprintf("%d", providerOpts.MaxConcurrent)))
		if providerOpts.IdleTimeout > 0 {
			log.Info("空闲超时", "timeout", log.Cyan(providerOpts.IdleTimeout.String()))
		}

		// 等待信号优雅关闭
		go provider.WaitForSignal(p)

		return p.Start()
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)

	// Provider 专属选项
	providerCmd.Flags().IntVar(&providerPort, "port", 9223, log.Cyan("Provider服务监听端口"))
	providerCmd.Flags().IntVar(&providerChromePort, "chrome-port", 9222, log.Cyan("Chrome远程调试端口"))
	providerCmd.Flags().IntVar(&providerMaxConcurrent, "max-concurrent", 10, log.Cyan("最大并发截图数"))
	providerCmd.Flags().DurationVar(&providerIdleTimeout, "idle-timeout", 0, log.Cyan("浏览器空闲超时 (如 5m, 0=不自动关闭)"))
	providerCmd.Flags().StringVar(&providerChromePath, "chrome-path", "", log.Cyan("Chrome可执行文件路径"))
	providerCmd.Flags().StringVar(&providerUserAgent, "user-agent", "", log.Cyan("自定义User-Agent"))
	providerCmd.Flags().StringVar(&providerProxy, "proxy", "", log.Cyan("代理服务器地址"))
	providerCmd.Flags().BoolVar(&providerHeadless, "headless", true, log.Cyan("使用无头模式"))
	providerCmd.Flags().BoolVar(&providerIgnoreCerts, "ignore-cert-errors", false, log.Cyan("忽略证书错误"))

	log.Debug(log.Green("已注册provider命令"))
}
