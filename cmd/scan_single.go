package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/scan"
)

var singleCmd = &cobra.Command{
	Use:   "single [url]",
	Short: log.Yellow("扫描单个URL"),
	Long:  log.Yellow("扫描单个URL并进行截图"),
	Example: `  # 基本用法
  ./snir scan single example.com
  
  # 增加超时和延迟（对于加载慢的网站）
  ./snir scan single example.com --timeout 60 --delay 3
  
  # 保存HTML和HTTP头信息
  ./snir scan single example.com --save-html --save-headers
  
  # 使用代理
  ./snir scan single example.com --proxy http://127.0.0.1:8080
  
  # 执行JavaScript脚本（例如关闭弹窗）
  ./snir scan single example.com --js "document.querySelectorAll('.popup').forEach(el => el.remove());"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		// 创建扫描配置（使用连接池复用 Chrome 进程）
		config := &scan.Config{
			Target:  target,
			Options: opts,
			UsePool: true,
		}

		// 创建扫描器
		scanner, err := scan.NewScanner(config)
		if err != nil {
			return fmt.Errorf("创建扫描器失败: %v", err)
		}
		defer scanner.Close()

		// 执行扫描
		log.CommandTitle("扫描URL")
		log.Info("开始扫描", "url", log.Cyan(target))
		result, err := scanner.ScanSingle(target)
		if err != nil {
			// 美化错误消息
			errMsg := err.Error()

			// 处理常见的ChromeDP错误
			if strings.Contains(errMsg, "Could not find node with given id") {
				return fmt.Errorf("扫描过程中发生错误: 无法找到页面上的某个元素。这可能是因为:\n" +
					"1. 网站加载较慢，请尝试增加超时时间 (--timeout 选项)\n" +
					"2. 网站可能有反爬虫措施\n" +
					"3. 网站结构与预期不符\n" +
					"建议尝试增加延迟: --delay 3")
			} else if strings.Contains(errMsg, "timeout") {
				return fmt.Errorf("扫描超时: 无法在指定时间内完成页面加载。请尝试:\n" +
					"1. 增加超时时间: --timeout 60\n" +
					"2. 检查网络连接\n" +
					"3. 检查目标站点是否可访问")
			} else if strings.Contains(errMsg, "net::ERR_") {
				return fmt.Errorf("网络错误: 无法连接到目标网站。请检查:\n" +
					"1. 目标URL是否正确\n" +
					"2. 您的网络连接\n" +
					"3. 目标站点是否在线")
			}

			return fmt.Errorf("扫描失败: %v", err)
		}

		// 打印结果
		printResult(result)

		return nil
	},
}

// printResult 打印扫描结果
func printResult(result interface{}) {
	log.Success("扫描完成")
}

func init() {
	scanCmd.AddCommand(singleCmd)
	log.Debug(log.Green("已注册single命令"))

	// 自定义帮助输出，为示例部分添加颜色
	defaultHelpFunc := singleCmd.HelpFunc()
	singleCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
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
}
