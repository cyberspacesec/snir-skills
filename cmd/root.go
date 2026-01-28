package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/go-snir/pkg/ascii"
	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/runner"
)

var (
	opts = &runner.Options{}
)

// init 在 rootCmd 构建后执行
func init() {
	log.Debug(log.Green("初始化根命令"))
	// 移除重复添加的serveCmd，因为在各自的文件中已经添加了

	// 设置自定义帮助模板
	helpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd.Name() == rootCmd.Name() {
			// 没有子命令，显示自定义的帮助信息
			showCustomHelp()
			return
		}
		helpFunc(cmd, args)
	})

	// 修改默认help命令的描述
	if helpCmd, _, err := rootCmd.Find([]string{"help"}); err == nil {
		helpCmd.Short = log.Yellow("帮助信息")
		helpCmd.Long = log.Yellow("显示命令的帮助信息")
	}
}

// rootCmd 代表没有调用子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   "snir",
	Short: log.Bold(log.Cyan("一个网页截图和信息收集工具")),
	Long:  ascii.Logo(),
	// 禁用自动添加帮助命令
	DisableSuggestions: true,
	SilenceErrors:      true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if opts.Logging.Silence {
			log.EnableSilence()
		}

		if opts.Logging.Debug && !opts.Logging.Silence {
			log.EnableDebug()
			log.Debug(log.Green("调试日志已启用"))
		}

		return nil
	},
	// 添加自定义帮助命令
	RunE: func(cmd *cobra.Command, args []string) error {
		// 在命令执行前调用，设置自定义帮助命令
		return cmd.Help()
	},
}

// 定义一个显示自定义帮助信息的函数
func showCustomHelp() {
	// 输出 Logo 和基本信息
	fmt.Print(ascii.Logo())

	fmt.Println("Usage:")
	fmt.Println("  snir [command]")
	fmt.Println("")

	fmt.Println("Available Commands:")

	// 直接定义所有可用命令，确保不会重复
	fmt.Printf("  %-12s %s\n", "api", log.Yellow("启动API服务"))
	fmt.Printf("  %-12s %s\n", "help", log.Yellow("帮助信息"))
	fmt.Printf("  %-12s %s\n", "report", log.Yellow("报告相关命令"))
	fmt.Printf("  %-12s %s\n", "scan", log.Yellow("扫描并截图网站"))
	fmt.Printf("  %-12s %s\n", "serve", log.Yellow("启动Web服务器查看结果"))
	fmt.Printf("  %-12s %s\n", "version", log.Yellow("显示版本信息"))

	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -D, --debug-log   启用调试日志")
	fmt.Println("  -h, --help        help for snir")
	fmt.Println("  -q, --quiet       静默（几乎所有）日志")

	fmt.Println("")
	fmt.Println("Use \"snir [command] --help\" for more information about a command.")
}

func Execute() {
	// 设置帮助命令的描述
	if helpCmd, _, err := rootCmd.Find([]string{"help"}); err == nil {
		helpCmd.Short = log.Yellow("帮助信息")
		helpCmd.Long = log.Yellow("显示命令的帮助信息")
	}

	// 检查是否有传入参数
	if len(os.Args) == 1 {
		// 没有参数，显示自定义的帮助信息
		fmt.Print(ascii.Logo())

		fmt.Println("Usage:")
		fmt.Println("  snir [command]")
		fmt.Println("")

		fmt.Println("Available Commands:")
		fmt.Printf("  %-12s %s\n", "api", log.Yellow("启动API服务"))
		fmt.Printf("  %-12s %s\n", "help", log.Yellow("帮助信息"))
		fmt.Printf("  %-12s %s\n", "report", log.Yellow("报告相关命令"))
		fmt.Printf("  %-12s %s\n", "scan", log.Yellow("扫描并截图网站"))
		fmt.Printf("  %-12s %s\n", "serve", log.Yellow("启动Web服务器查看结果"))
		fmt.Printf("  %-12s %s\n", "version", log.Yellow("显示版本信息"))

		fmt.Println("")
		fmt.Println("Flags:")
		fmt.Println("  -D, --debug-log   启用调试日志")
		fmt.Println("  -h, --help        help for snir")
		fmt.Println("  -q, --quiet       静默（几乎所有）日志")

		fmt.Println("")
		fmt.Println("Use \"snir [command] --help\" for more information about a command.")
		return
	}

	// 有传入参数，使用Cobra的标准执行流程
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	err := rootCmd.Execute()
	if err != nil {
		var cmd string
		c, _, cerr := rootCmd.Find(os.Args[1:])
		if cerr == nil {
			cmd = c.Name()
		}

		v := "\n"

		if cmd != "" {
			v += fmt.Sprintf(log.Red("运行 `%s` 命令时发生错误\n"), cmd)
		} else {
			v += log.Red("发生了一个错误。 ")
		}

		v += log.Red("错误信息为:\n\n") + fmt.Sprintf("```%s```", log.Red(err.Error()))
		fmt.Println(ascii.Markdown(v))

		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&opts.Logging.Debug, "debug-log", "D", false, log.Cyan("启用调试日志"))
	rootCmd.PersistentFlags().BoolVarP(&opts.Logging.Silence, "quiet", "q", false, log.Cyan("静默（几乎所有）日志"))
}
