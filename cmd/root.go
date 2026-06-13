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
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	// 注册根级别持久化标志
	rootCmd.PersistentFlags().BoolVarP(&opts.Logging.Debug, "debug-log", "D", false, log.Cyan("启用调试日志"))
	rootCmd.PersistentFlags().BoolVarP(&opts.Logging.Silence, "quiet", "q", false, log.Cyan("静默（几乎所有）日志"))

	// 设置自定义帮助模板 — 自动从 cobra 命令树生成，不再硬编码
	rootCmd.SetHelpFunc(showCobraHelp)

	// 修改默认 help 命令的描述
	if helpCmd, _, err := rootCmd.Find([]string{"help"}); err == nil {
		helpCmd.Short = log.Yellow("帮助信息")
		helpCmd.Long = log.Yellow("显示命令的帮助信息")
	}
}

// showCobraHelp 基于 Cobra 命令树自动生成帮助信息
// 不再硬编码命令列表，新命令自动出现在帮助中
func showCobraHelp(cmd *cobra.Command, args []string) {
	// 根命令显示 Logo
	if cmd.Name() == rootCmd.Name() {
		fmt.Print(ascii.Logo())
	}

	fmt.Println("Usage:")
	fmt.Printf("  %s [command]\n", cmd.CommandPath())
	fmt.Println()

	// 可用子命令（从 cobra 自动获取）
	if cmd.HasAvailableSubCommands() {
		fmt.Println("Available Commands:")
		for _, sub := range cmd.Commands() {
			if sub.IsAvailableCommand() && !sub.Hidden {
				fmt.Printf("  %-12s %s\n", sub.Name(), log.Yellow(sub.Short))
			}
		}
		fmt.Println()
	}

	// 本地标志
	if cmd.HasAvailableFlags() {
		fmt.Println("Flags:")
		fmt.Print(cmd.Flags().FlagUsages())
		fmt.Println()
	}

	// 继承的持久化标志
	if cmd.HasAvailableInheritedFlags() {
		fmt.Println("Global Flags:")
		fmt.Print(cmd.InheritedFlags().FlagUsages())
		fmt.Println()
	}

	fmt.Printf("Use \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
}

// Execute 执行根命令
func Execute() {
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
