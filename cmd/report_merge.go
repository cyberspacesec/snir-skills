package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/report"
)

var mergeCmdFlags = struct {
	sourceFiles []string
	sourcePath  string
	outputFile  string
}{}

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "合并多个报告",
	Long:  "将多个报告合并为一个报告",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查必要的参数
		if len(mergeCmdFlags.sourceFiles) == 0 && mergeCmdFlags.sourcePath == "" {
			return fmt.Errorf("请使用 --files 或 --path 参数指定源文件")
		}
		if mergeCmdFlags.outputFile == "" {
			return fmt.Errorf("请使用 --output 参数指定输出文件")
		}

		// 创建合并选项
		options := report.MergeOptions{
			SourceFiles: mergeCmdFlags.sourceFiles,
			SourcePath:  mergeCmdFlags.sourcePath,
			OutputFile:  mergeCmdFlags.outputFile,
		}

		// 执行合并
		log.Info("开始合并报告", "output", options.OutputFile)
		if err := report.Merge(options); err != nil {
			return fmt.Errorf("合并报告失败: %v", err)
		}

		log.Info("报告合并完成", "output", options.OutputFile)
		return nil
	},
}

func init() {
	reportCmd.AddCommand(mergeCmd)

	// 添加合并相关选项
	mergeCmd.Flags().StringSliceVar(&mergeCmdFlags.sourceFiles, "files", []string{}, "源文件路径列表")
	mergeCmd.Flags().StringVar(&mergeCmdFlags.sourcePath, "path", "", "包含源文件的目录路径")
	mergeCmd.Flags().StringVar(&mergeCmdFlags.outputFile, "output", "", "输出文件路径")
	mergeCmd.MarkFlagRequired("output")

	log.Debug("已注册merge命令")
}
