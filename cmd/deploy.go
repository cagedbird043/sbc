package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "同步 + 渲染 + 部署 + 重启",
	Long:  "从私有 production truth 同步模板，渲染配置，部署到系统，重启服务。",
	Run: func(cmd *cobra.Command, args []string) {
		cmdUpdate()
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "检查模板渲染语法（不部署）",
	Run: func(cmd *cobra.Command, args []string) {
		cmdValidate()
	},
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "检查已部署配置语法",
	Run: func(cmd *cobra.Command, args []string) {
		cmdCheck()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(checkCmd)
}

func cmdUpdate() {
	fmt.Println("📡 正在从私有 production truth 下发指令...")

	// Load env to get any SBC_TEMPLATE_ROOT override
	vars, err := internal.LoadEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	if missing := internal.RequireEnvVars(vars); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "❌ 缺少必需环境变量: %s\n", strings.Join(missing, ", "))
		os.Exit(1)
	}

	profile := internal.Profile()

	// macOS: cache sudo credentials
	if profile == "macos" {
		exec.Command("sudo", "-v").Run()
	}

	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "sbc-runtime.*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	renderedConf := tmpDir + "/config.json.tmp"

	// Sync private repo
	fmt.Println("📦 正在同步私有 production truth...")
	if err := internal.SyncPrivateRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// Render
	templatePath, _ := internal.TemplatePath()
	if err := internal.RenderProfile(renderedConf, vars); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 渲染失败 (%s): %v\n", templatePath, err)
		os.Exit(1)
	}

	// Basic validation
	data, err := os.ReadFile(renderedConf)
	if err != nil || !strings.Contains(string(data), "inbounds") {
		fmt.Fprintf(os.Stderr, "❌ 模板渲染结果校验失败。\n")
		os.Exit(1)
	}
	fmt.Println("✅ 私有模板已同步。准备执行预检与部署...")

	// Syntax check
	singBoxBin, err := internal.SingBoxBin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	checkCmd := exec.Command(singBoxBin, "check", "-c", renderedConf)
	checkCmd.Stdout = os.Stdout
	checkCmd.Stderr = os.Stderr
	if err := checkCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 渲染配置语法有误，拒绝执行！\n")
		os.Exit(1)
	}

	fmt.Println("✅ 语法预检通过。执行原子化部署...")

	// Install config
	if err := internal.InstallConfig(renderedConf); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 部署配置失败: %v\n", err)
		os.Exit(1)
	}

	// Restart service
	serviceRestart()

	templateRoot, _ := internal.TemplateRoot()
	fmt.Printf("✨ 同步完成！轨道：%s @ %s\n", profile, templateRoot)
}

func cmdValidate() {
	tmpFile, err := os.CreateTemp("", "sbc-validate.*.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建临时文件失败: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())

	vars, err := internal.LoadEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	if missing := internal.RequireEnvVars(vars); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "❌ 缺少必需环境变量: %s\n", strings.Join(missing, ", "))
		os.Exit(1)
	}

	if err := internal.RenderProfile(tmpFile.Name(), vars); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 渲染失败: %v\n", err)
		os.Exit(1)
	}

	singBoxBin, err := internal.SingBoxBin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	checkCmd := exec.Command(singBoxBin, "check", "-c", tmpFile.Name())
	checkCmd.Stdout = os.Stdout
	checkCmd.Stderr = os.Stderr
	if err := checkCmd.Run(); err != nil {
		templatePath, _ := internal.TemplatePath()
		fmt.Fprintf(os.Stderr, "❌ 私有模板渲染语法失败：%s\n", templatePath)
		os.Exit(1)
	}

	templatePath, _ := internal.TemplatePath()
	fmt.Printf("✅ 私有模板渲染通过：%s\n", templatePath)
}

func cmdCheck() {
	singBoxBin, err := internal.SingBoxBin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	target := internal.TargetConf()
	checkCmd := exec.Command(singBoxBin, "check", "-c", target)
	checkCmd.Stdout = os.Stdout
	checkCmd.Stderr = os.Stderr
	if err := checkCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 语法检查失败！\n")
		os.Exit(1)
	}

	fmt.Println("✅ 语法检查通过 (Current Config)")
}
