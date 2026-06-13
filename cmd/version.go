package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cyberspacesec/snir-skills/pkg/ascii"
	"github.com/cyberspacesec/snir-skills/pkg/log"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: log.Yellow("显示版本信息"),
	Long:  log.Yellow("显示详细的版本和构建信息"),
	Run: func(cmd *cobra.Command, args []string) {
		log.CommandTitle("版本信息")
		fmt.Println(ascii.VersionInfo())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	log.Debug(log.Green("已注册version命令"))
}
