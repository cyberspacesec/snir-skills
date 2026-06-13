package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/api"
	"github.com/cyberspacesec/snir-skills/pkg/log"
)

// 生成随机API密钥
func generateRandomAPIKey(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		log.Error("生成API密钥失败", "error", log.Red(err.Error()))
		return "snir-random-api-key"
	}
	return hex.EncodeToString(bytes)
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: log.Yellow("启动API服务"),
	Long:  log.Yellow("启动一个RESTful API服务，用于进行网页截图和信息收集"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 如果未指定API密钥，则生成一个随机密钥
		if opts.API.APIKey == "" {
			opts.API.APIKey = generateRandomAPIKey(32)
		}

		// 创建API服务配置
		apiOptions := api.ServerOptions{
			Port:                  opts.API.Port,
			Host:                  opts.API.Host,
			ScreenshotPath:        opts.Scan.ScreenshotPath,
			APIKey:                opts.API.APIKey,
			EnableBlacklist:       opts.Scan.EnableBlacklist,
			DefaultBlacklist:      opts.Scan.DefaultBlacklist,
			BlacklistPatterns:     opts.Scan.BlacklistPatterns,
			BlacklistFile:         opts.Scan.BlacklistFile,
			MaxConcurrentRequests: opts.API.MaxConcurrent,
			RequestQueueSize:      opts.API.QueueSize,
		}

		// 创建API服务
		server := api.NewServer(apiOptions)

		// 初始化浏览器连接池（复用 Chrome 进程）
		if err := server.InitPool(opts); err != nil {
			log.Error("初始化浏览器连接池失败，将使用单次模式", "error", err)
		}

		// 设置路由
		server.SetupRoutes()

		// 启动服务 — 输出清晰的访问信息
		displayHost := opts.API.Host
		if displayHost == "0.0.0.0" {
			displayHost = "127.0.0.1"
		}
		log.CommandTitle("API 服务已启动")
		log.Info("本地访问", "url", log.Cyan(fmt.Sprintf("http://%s:%d", displayHost, opts.API.Port)))
		log.Info("API 密钥", "key", log.Cyan(opts.API.APIKey))
		log.Info("API 文档", "url", log.Cyan(fmt.Sprintf("http://%s:%d/", displayHost, opts.API.Port)))
		log.Info("健康检查", "url", log.Cyan(fmt.Sprintf("http://%s:%d/health", displayHost, opts.API.Port)))
		log.Info("并发配置", "max_concurrent", log.Cyan(fmt.Sprintf("%d", opts.API.MaxConcurrent)), "queue_size", log.Cyan(fmt.Sprintf("%d", opts.API.QueueSize)))
		return server.Run()
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)

	// 添加API相关选项
	apiCmd.Flags().StringVar(&opts.API.Host, "host", "0.0.0.0", log.Cyan("API服务监听地址"))
	apiCmd.Flags().IntVar(&opts.API.Port, "port", 8080, log.Cyan("API服务监听端口"))
	apiCmd.Flags().StringVar(&opts.API.APIKey, "api-key", "", log.Cyan("API密钥，用于API鉴权，如不指定则自动生成"))

	// 添加黑名单相关选项
	apiCmd.Flags().BoolVar(&opts.Scan.EnableBlacklist, "enable-blacklist", true, log.Cyan("启用URL黑名单检查"))
	apiCmd.Flags().BoolVar(&opts.Scan.DefaultBlacklist, "default-blacklist", true, log.Cyan("使用默认黑名单规则"))
	apiCmd.Flags().StringSliceVar(&opts.Scan.BlacklistPatterns, "blacklist-pattern", []string{}, log.Cyan("添加自定义黑名单规则 (可多次使用)"))
	apiCmd.Flags().StringVar(&opts.Scan.BlacklistFile, "blacklist-file", "", log.Cyan("黑名单规则文件路径"))

	// 添加并发控制相关选项
	apiCmd.Flags().IntVar(&opts.API.MaxConcurrent, "max-concurrent", 10, log.Cyan("最大并发请求数"))
	apiCmd.Flags().IntVar(&opts.API.QueueSize, "queue-size", 100, log.Cyan("请求队列大小"))
	apiCmd.Flags().StringVar(&opts.Chrome.WSS, "wss", "", log.Cyan("远程Chrome WebSocket URL (如 ws://host:9222/devtools/browser/xxx)"))
	apiCmd.Flags().BoolVar(&opts.Chrome.IgnoreCertErrors, "ignore-cert-errors", false, log.Cyan("忽略证书错误"))

	log.Debug(log.Green("已注册api命令"))
}
