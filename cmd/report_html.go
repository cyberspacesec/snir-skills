package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/report"
)

var htmlCmd = &cobra.Command{
	Use:   "html",
	Short: "生成HTML报告",
	Long:  "根据扫描结果生成HTML格式的报告",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查输入文件
		if opts.Report.InputFile == "" {
			return fmt.Errorf("请使用 --input 参数指定JSONL结果文件")
		}

		// 创建HTML选项
		htmlOptions := report.HTMLOptions{
			InputFile:  opts.Report.InputFile,
			OutputPath: opts.Report.OutputPath,
		}

		// 生成HTML报告
		err := report.GenerateHTML(htmlOptions)
		if err != nil {
			return fmt.Errorf("生成HTML报告失败: %v", err)
		}

		log.Info("HTML报告生成成功", "path", opts.Report.OutputPath)
		return nil
	},
}

func init() {
	reportCmd.AddCommand(htmlCmd)

	// 添加HTML报告相关选项
	htmlCmd.Flags().StringVar(&opts.Report.InputFile, "input", "", "JSONL格式的结果文件路径")
	htmlCmd.Flags().StringVar(&opts.Report.OutputPath, "output", "report.html", "HTML报告输出路径")
	htmlCmd.MarkFlagRequired("input")

	log.Debug("已注册html报告命令")
}
