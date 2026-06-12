package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/provider"
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
  ./snir provider --max-concurrent 20 --idle-timeout 10m`,
	RunE: func(cmd *cobra.Command, args []string) error {
		providerOpts := provider.DefaultProviderOptions()

		// 从命令行参数读取
		providerOpts.ChromePath = opts.Chrome.Path
		providerOpts.Headless = opts.Chrome.Headless
		providerOpts.WindowWidth = opts.Chrome.WindowX
		providerOpts.WindowHeight = opts.Chrome.WindowY
		providerOpts.UserAgent = opts.Chrome.UserAgent
		providerOpts.Proxy = opts.Chrome.Proxy
		providerOpts.IgnoreCertErrors = opts.Chrome.IgnoreCertErrors

		// 创建 Provider
		p := provider.NewProvider(providerOpts)

		log.CommandTitle("启动CDP Provider服务")
		log.Info("服务地址", "host", log.Cyan(providerOpts.Host), "port", log.Cyan(fmt.Sprintf("%d", providerOpts.Port)))
		log.Info("Chrome调试端口", "port", log.Cyan(fmt.Sprintf("%d", providerOpts.ChromeDebugPort)))
		log.Info("最大并发", "max_concurrent", log.Cyan(fmt.Sprintf("%d", providerOpts.MaxConcurrent)))

		// 等待信号优雅关闭
		go provider.WaitForSignal(p)

		return p.Start()
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)

	// Provider 专属选项
	providerCmd.Flags().IntVar(&opts.API.Port, "port", 9223, log.Cyan("Provider服务监听端口"))
	providerCmd.Flags().IntVar(&opts.Chrome.WindowX, "chrome-port", 9222, log.Cyan("Chrome远程调试端口"))
	providerCmd.Flags().IntVar(&opts.API.MaxConcurrent, "max-concurrent", 10, log.Cyan("最大并发截图数"))

	log.Debug(log.Green("已注册provider命令"))
}
