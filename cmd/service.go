package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)


var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 sing-box 服务",
	Run: func(cmd *cobra.Command, args []string) {
		serviceStart()
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 sing-box 服务",
	Run: func(cmd *cobra.Command, args []string) {
		serviceStop()
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "重启 sing-box 服务",
	Run: func(cmd *cobra.Command, args []string) {
		serviceRestart()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看 sing-box 服务状态",
	Run: func(cmd *cobra.Command, args []string) {
		serviceStatus()
	},
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "查看 sing-box 服务日志",
	Long:  "Linux: journalctl -u sing-box -f -n 50\nmacOS: tail -f /opt/homebrew/var/log/sing-box.log",
	Run: func(cmd *cobra.Command, args []string) {
		serviceLog()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logCmd)
}

func serviceStart() {
	profile := internal.Profile()
	switch profile {
	case "macos":
		label := internal.ServiceLabelMacOS()
		plist := "/Library/LaunchDaemons/" + label + ".plist"
		// bootstrap fails when the daemon is already loaded; kickstart proves it can run.
		runCmdIgnoreError("sudo", "launchctl", "bootstrap", "system", plist)
		runCmdRequired("sudo", "launchctl", "kickstart", "-k", "system/"+label)
	default:
		runCmdRequired("sudo", "systemctl", "start", internal.ServiceNameLinux())
	}
	fmt.Println("✅ 服务已启动。")
}

func serviceStop() {
	profile := internal.Profile()
	switch profile {
	case "macos":
		label := internal.ServiceLabelMacOS()
		plist := "/Library/LaunchDaemons/" + label + ".plist"
		runCmdRequired("sudo", "launchctl", "bootout", "system", plist)
	default:
		runCmdRequired("sudo", "systemctl", "stop", internal.ServiceNameLinux())
	}
	fmt.Println("⚠ 服务已停止。")
}

func serviceRestart() {
	profile := internal.Profile()
	switch profile {
	case "macos":
		label := internal.ServiceLabelMacOS()
		plist := "/Library/LaunchDaemons/" + label + ".plist"
		runCmdIgnoreError("sudo", "launchctl", "bootout", "system", plist)
		runCmdRequired("sudo", "launchctl", "bootstrap", "system", plist)
		runCmdRequired("sudo", "launchctl", "kickstart", "-k", "system/"+label)
	default:
		runCmdRequired("sudo", "systemctl", "restart", internal.ServiceNameLinux())
	}
	fmt.Println("✅ 服务已重启。")
}

func serviceStatus() {
	profile := internal.Profile()
	switch profile {
	case "macos":
		label := internal.ServiceLabelMacOS()
		cmd := exec.Command("launchctl", "print", "system/"+label)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	default:
		cmd := exec.Command("systemctl", "status", internal.ServiceNameLinux(), "--no-pager")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}

func serviceLog() {
	profile := internal.Profile()
	switch profile {
	case "macos":
		cmd := exec.Command("tail", "-f", "-n", "80", "/opt/homebrew/var/log/sing-box.log")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	default:
		cmd := exec.Command("journalctl", "-u", internal.ServiceNameLinux(), "-f", "-n", "50")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}

// runCmd executes a command and prints its output. Used for service management.
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCmdRequired(name string, args ...string) {
	if err := runCmd(name, args...); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 命令执行失败: %s %v: %v\n", name, args, err)
		os.Exit(1)
	}
}

func runCmdIgnoreError(name string, args ...string) {
	_ = runCmd(name, args...)
}
