package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion {zsh|bash|fish|powershell}",
	Short: "生成 shell 补全脚本",
	Long: `生成指定 shell 的补全脚本，输出到 stdout。

Zsh:
  source <(sbc completion zsh)
  或保存到文件:
  sbc completion zsh > ~/.zsh/completion/_sbc

Bash:
  source <(sbc completion bash)
  或保存到文件:
  sbc completion bash > /etc/bash_completion.d/sbc`,
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "生成 Zsh 补全脚本",
	Run: func(cmd *cobra.Command, args []string) {
		if err := rootCmd.GenZshCompletion(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 生成 Zsh 补全失败: %v\n", err)
			os.Exit(1)
		}
	},
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "生成 Bash 补全脚本",
	Run: func(cmd *cobra.Command, args []string) {
		if err := rootCmd.GenBashCompletionV2(os.Stdout, true); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 生成 Bash 补全失败: %v\n", err)
			os.Exit(1)
		}
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "生成 Fish 补全脚本",
	Run: func(cmd *cobra.Command, args []string) {
		if err := rootCmd.GenFishCompletion(os.Stdout, true); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 生成 Fish 补全失败: %v\n", err)
			os.Exit(1)
		}
	},
}

var completionPowershellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "生成 PowerShell 补全脚本",
	Run: func(cmd *cobra.Command, args []string) {
		if err := rootCmd.GenPowerShellCompletionWithDesc(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 生成 PowerShell 补全失败: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionFishCmd)
	completionCmd.AddCommand(completionPowershellCmd)
}
