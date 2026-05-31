package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via ldflags.
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "sbc",
	Short: "sing-box commander",
	Long:  "sbc — sing-box 命令行控制器，管理 sing-box 服务、配置、代理、面板。",
	Run: func(cmd *cobra.Command, args []string) {
		cmdOverview()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "sbc: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version
}

// cmdOverview prints a status summary (like the shell version's empty-arg behavior).
func cmdOverview() {
	platform := internal.Platform()
	variant := activeVariant()
	variantDesc := variantDescription(variant)

	envFile, _ := internal.EnvFilePath()
	targetConf := internal.TargetConf()

	// Template path: show local cache if available, else "URL 分发"
	templateInfo := "URL 分发"
	if tp, err := internal.ActiveVariantTemplatePath(); err == nil {
		templateInfo = tp
	}

	// Service status
	serviceStatus := checkServiceStatus(platform)

	// API status
	apiStatus := checkAPIStatus()

	fmt.Println("────────── sing-box 状态概览 ──────────")
	fmt.Printf("  服务:     %s\n", serviceStatus)
	fmt.Printf("  API:      %s\n", apiStatus)
	fmt.Printf("  变体:     %s（%s）\n", variant, variantDesc)
	fmt.Printf("  平台:     %s\n", platform)
	fmt.Printf("  模板:     %s\n", templateInfo)
	fmt.Printf("  目标:     %s\n", targetConf)
	fmt.Printf("  环境:     %s\n", envFile)
}

// checkServiceStatus checks if the sing-box service is running.
func checkServiceStatus(platform string) string {
	switch platform {
	case "linux":
		cmd := exec.Command("systemctl", "is-active", "--quiet", internal.ServiceNameLinux())
		if err := cmd.Run(); err == nil {
			return "运行中"
		}
		return "未运行"
	case "macos":
		cmd := exec.Command("launchctl", "print", "system/"+internal.ServiceLabelMacOS())
		out, err := cmd.Output()
		if err == nil {
			if contains(string(out), "state = running") {
				return "运行中"
			}
		}
		return "未运行"
	default:
		return "未知"
	}
}

// checkAPIStatus checks if the Clash API is accessible.
func checkAPIStatus() string {
	envFile, err := internal.EnvFilePath()
	if err != nil {
		return "无 .env"
	}
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return "无 .env"
	}

	vars, err := readEnvFile(envFile)
	if err != nil {
		return "无 .env"
	}
	secret := vars["CLASH_API_SECRET"]
	if secret == "" {
		return "无密钥"
	}

	status, err := apiGet("/")
	if err != nil || status == "" {
		return "未连接"
	}
	return "已连接"
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
