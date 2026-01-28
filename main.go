package main

import (
	"fmt"
	"os"

	"github.com/cyberspacesec/go-snir/cmd"
	"github.com/cyberspacesec/go-snir/pkg/ascii"
	"github.com/cyberspacesec/go-snir/pkg/log"
)

func main() {
	// 如果没有参数，直接输出自定义帮助信息
	if len(os.Args) == 1 {
		// 输出自定义帮助信息
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
		fmt.Printf("  %-12s %s\n", "webserve", log.Yellow("启动Web服务器查看结果（别名）"))
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

	// 有参数则执行命令
	cmd.Execute()
}
