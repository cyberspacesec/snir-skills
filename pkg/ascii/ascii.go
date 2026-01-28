package ascii

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
)

// BuildInfo 包含编译时注入的版本信息
var (
	// 版本信息
	version   = "unknown" // 版本号
	commit    = "unknown" // Git提交哈希
	buildDate = "unknown" // 构建日期
	buildTime = "unknown" // 构建时间
)

// ProjectURL 是项目的GitHub URL
const ProjectURL = "https://github.com/cyberspacesec/go-snir"

// Logo 返回 go-snir 的 ASCII 艺术标志
// 使用了颜色修饰以提高可读性
// 返回包含项目标志和基本信息的彩色字符串
func Logo() string {
	// 准备彩色输出函数
	blue := color.New(color.FgBlue).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// ASCII 艺术标志
	logoText := `
  ____           ____       _      
 / ___| ___     / ___|  ___(_)_ __ 
| |  _ / _ \____\___ \ / __| | '__|
| |_| | (_)|_____|__) | (__| | |   
 \____|\___/    |____/ \___|_|_|   
                                   
`
	// 添加颜色
	coloredLogo := blue(logoText)

	// 添加信息文本
	info := fmt.Sprintf("\n%s\n%s: %s\n%s: %s\n",
		yellow("一个强大的网页截图和信息收集工具"),
		cyan("版本"),
		green(version),
		cyan("项目地址"),
		ProjectURL)

	return coloredLogo + info
}

// VersionInfo 返回详细的版本信息
// 包括版本号、Git提交哈希、构建日期和时间
// 返回格式化的版本信息字符串
func VersionInfo() string {
	cyan := color.New(color.FgCyan).SprintFunc()
	return fmt.Sprintf("版本: %s\n提交: %s\n构建时间: %s %s\n项目地址: %s\n",
		version, commit, buildDate, buildTime,
		cyan(ProjectURL))
}

// Markdown 将 Markdown 文本渲染为终端友好的格式
// 使用 glamour 库进行 Markdown 到 ANSI 的转换
// 参数:
//   - markdown: 要渲染的 Markdown 文本
//
// 返回:
//   - 格式化后的终端友好文本
func Markdown(markdown string) string {
	// 创建一个新的渲染器，使用自动样式和适当的文本换行
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	// 渲染 Markdown 文本
	out, err := r.Render(markdown)
	if err != nil {
		return fmt.Sprintf("渲染Markdown时出错: %s\n%s\n\n请到 %s/issues 提交问题反馈",
			err, markdown, ProjectURL)
	}

	// 移除尾部换行符以避免额外空行
	return strings.TrimSuffix(out, "\n")
}
