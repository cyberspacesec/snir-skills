package cmd

import (
	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/report"
)

// webserveCmd 替代原有的serveCmd，避免命令重复问题
var webserveCmd = &cobra.Command{
	Use:     "webserve",
	Aliases: []string{"serve"},
	Short:   log.Yellow("启动Web服务器查看结果"),
	Long:    log.Yellow("启动一个Web服务器，用于查看截图和扫描结果"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建服务器配置
		serverOptions := report.ServerOptions{
			Host:           opts.Report.Host,
			Port:           opts.Report.Port,
			ScreenshotPath: opts.Scan.ScreenshotPath,
			ReportPath:     opts.Report.OutputPath,
		}

		// 创建服务器
		server := report.NewServer(serverOptions)

		// 启动服务器
		log.CommandTitle("启动Web服务器")
		log.Info("服务器地址", "host", log.Cyan(opts.Report.Host), "port", log.Cyan(opts.Report.Port))
		return server.Run()
	},
}

func init() {
	// 添加webserveCmd到根命令
	rootCmd.AddCommand(webserveCmd)

	// 添加服务器选项
	webserveCmd.Flags().StringVar(&opts.Report.Host, "host", "0.0.0.0", log.Cyan("Web服务器监听地址"))
	webserveCmd.Flags().IntVar(&opts.Report.Port, "port", 8080, log.Cyan("Web服务器监听端口"))

	log.Debug(log.Green("已注册serve命令"))
}
