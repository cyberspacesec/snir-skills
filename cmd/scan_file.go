package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/scan"
)

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: log.Yellow("从文件批量扫描URL"),
	Long:  log.Yellow("从文件中读取URL列表进行批量扫描和截图"),
	Example: `  # 基本用法
  ./snir scan file -f urls.txt
  
  # 调整并发数进行扫描
  ./snir scan file -f urls.txt --threads 10
  
  # 保存结果为CSV格式
  ./snir scan file -f urls.txt --write-csv
  
  # 修改截图保存目录
  ./snir scan file -f urls.txt --screenshot-path custom_screenshots
  
  # 增加超时和延迟处理慢网站
  ./snir scan file -f urls.txt --timeout 60 --delay 3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查文件路径是否提供
		if opts.Scan.FilePath == "" {
			return fmt.Errorf("请使用 -f 或 --file 参数指定URL文件路径")
		}

		// 打开文件
		file, err := os.Open(opts.Scan.FilePath)
		if err != nil {
			return fmt.Errorf("无法打开文件: %v", err)
		}
		defer file.Close()

		// 读取URL列表
		var urls []string
		fileScanner := bufio.NewScanner(file)
		for fileScanner.Scan() {
			url := strings.TrimSpace(fileScanner.Text())
			if url != "" && !strings.HasPrefix(url, "#") {
				urls = append(urls, url)
			}
		}

		if err := fileScanner.Err(); err != nil {
			return fmt.Errorf("读取文件时出错: %v", err)
		}

		if len(urls) == 0 {
			return fmt.Errorf("文件中没有有效的URL")
		}

		log.Info("从文件中读取URL", "count", log.Cyan(fmt.Sprintf("%d", len(urls))), "file", log.Cyan(opts.Scan.FilePath))

		// 创建扫描配置（使用连接池复用 Chrome 进程）
		config := &scan.Config{
			Targets: urls,
			Options: opts,
			UsePool: true,
		}

		// 创建扫描器
		urlScanner, err := scan.NewScanner(config)
		if err != nil {
			return fmt.Errorf("创建扫描器失败: %v", err)
		}
		defer urlScanner.Close()

		// 执行扫描
		log.CommandTitle("批量扫描")
		log.Info("开始批量扫描", "url_count", log.Cyan(fmt.Sprintf("%d", len(urls))))
		err = urlScanner.ScanMulti(urls)
		if err != nil {
			return fmt.Errorf("批量扫描失败: %v", err)
		}

		log.Success("批量扫描完成", "file", log.Cyan(opts.Scan.FilePath))
		return nil
	},
}

func init() {
	scanCmd.AddCommand(fileCmd)

	// 添加文件扫描相关选项
	fileCmd.Flags().StringVarP(&opts.Scan.FilePath, "file", "f", "", log.Cyan("包含URL列表的文件路径"))
	fileCmd.MarkFlagRequired("file")

	// 自定义帮助输出，为示例部分添加颜色
	defaultHelpFunc := fileCmd.HelpFunc()
	fileCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// 保存原始示例
		originalExample := cmd.Example

		// 为示例添加颜色
		coloredExample := ""
		lines := strings.Split(originalExample, "\n")
		for _, line := range lines {
			// 为示例添加颜色
			if strings.HasPrefix(line, "  #") {
				coloredExample += log.Cyan(line) + "\n"
			} else if strings.HasPrefix(line, "  ./snir") {
				coloredExample += log.Yellow(line) + "\n"
			} else {
				coloredExample += line + "\n"
			}
		}
		cmd.Example = coloredExample

		// 调用默认帮助函数
		defaultHelpFunc(cmd, args)

		// 恢复原始示例
		cmd.Example = originalExample
	})

	log.Debug(log.Green("已注册file命令"))
}
