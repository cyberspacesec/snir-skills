package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/report"
)

var convertCmdFlags = struct {
	fromFile string
	toFile   string
}{}

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "转换报告格式",
	Long:  "将报告从一种格式转换为另一种格式",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查必要的参数
		if convertCmdFlags.fromFile == "" {
			return fmt.Errorf("请使用 --from 参数指定源文件")
		}
		if convertCmdFlags.toFile == "" {
			return fmt.Errorf("请使用 --to 参数指定目标文件")
		}

		// 创建转换选项
		options := report.ConvertOptions{
			FromFile: convertCmdFlags.fromFile,
			ToFile:   convertCmdFlags.toFile,
		}

		// 执行转换
		log.Info("开始转换报告", "from", options.FromFile, "to", options.ToFile)
		if err := report.Convert(options); err != nil {
			return fmt.Errorf("转换报告失败: %v", err)
		}

		log.Info("报告转换完成", "to", options.ToFile)
		return nil
	},
}

func init() {
	reportCmd.AddCommand(convertCmd)

	// 添加转换相关选项
	convertCmd.Flags().StringVar(&convertCmdFlags.fromFile, "from", "", "源文件路径")
	convertCmd.Flags().StringVar(&convertCmdFlags.toFile, "to", "", "目标文件路径")
	convertCmd.MarkFlagRequired("from")
	convertCmd.MarkFlagRequired("to")

	log.Debug("已注册convert命令")
}
