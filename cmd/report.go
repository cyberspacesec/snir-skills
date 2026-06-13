package cmd

import (
	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/log"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: log.Yellow("报告相关命令"),
	Long:  log.Yellow("管理和查看扫描报告的相关命令"),
}

func init() {
	// 添加report命令到根命令
	rootCmd.AddCommand(reportCmd)

	// 添加报告相关选项
	reportCmd.PersistentFlags().StringVar(&opts.Report.OutputPath, "output-path", "reports", log.Cyan("报告输出路径"))
	reportCmd.PersistentFlags().StringVar(&opts.Report.Format, "format", "html", log.Cyan("报告格式 (html, json, csv)"))
	reportCmd.PersistentFlags().StringVar(&opts.Report.Host, "host", "127.0.0.1", log.Cyan("Web服务器主机地址"))
	reportCmd.PersistentFlags().IntVar(&opts.Report.Port, "port", 8080, log.Cyan("Web服务器端口"))

	log.Debug(log.Green("已注册report命令"))
}
