package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "下载 + 渲染 + 部署 + 重启",
	Long:  "从 HTTPS URL 下载模板，渲染配置，部署到系统，重启服务。",
	Run: func(cmd *cobra.Command, args []string) {
		cmdUpdate()
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "下载 + 渲染 + 语法检查（不部署）",
	Long:  "从 HTTPS URL 下载模板、渲染、语法检查，但不部署。",
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
	fmt.Println("📡 正在从 HTTPS 模板分发拉取配置...")

	// 1. Read template URLs from .env
	urls, err := internal.ReadEnvURLs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 2. Download all configs
	downloaded, failed, err := internal.DownloadConfigs(urls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 下载失败: %v\n", err)
		os.Exit(1)
	}

	// 3. Verify downloads (checks active variant availability, etc.)
	if err := internal.VerifyDownloads(urls, downloaded, failed); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 4. Copy downloaded files to ConfigDir
	configDir := internal.ConfigDir()
	if configDir == "" {
		fmt.Fprintf(os.Stderr, "❌ 无法确定配置目录\n")
		os.Exit(1)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建配置目录失败: %v\n", err)
		os.Exit(1)
	}
	for filename, srcPath := range downloaded {
		dstPath := filepath.Join(configDir, filename)
		if err := copyFile(srcPath, dstPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 复制文件失败 (%s): %v\n", filename, err)
			os.Exit(1)
		}
	}
	fmt.Printf("✅ 已下载 %d 个模板到 %s\n", len(downloaded), configDir)

	// 5. Find active variant template
	templatePath, err := internal.ActiveVariantTemplatePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 6. Load .env variables
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

	// Create temp workspace for rendering
	tmpDir, err := os.MkdirTemp("", "sbc-runtime.*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	renderedConf := filepath.Join(tmpDir, "config.json.tmp")

	// 7. Render
	if err := internal.RenderProfile(templatePath, renderedConf, vars); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 渲染失败 (%s): %v\n", templatePath, err)
		os.Exit(1)
	}

	// Basic validation — must contain "inbounds"
	data, err := os.ReadFile(renderedConf)
	if err != nil || !strings.Contains(string(data), "inbounds") {
		fmt.Fprintf(os.Stderr, "❌ 模板渲染结果校验失败。\n")
		os.Exit(1)
	}
	fmt.Println("✅ 模板已渲染。准备执行预检与部署...")

	// 8. sing-box syntax check
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

	// 9. Deploy
	if err := internal.InstallConfig(renderedConf); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 部署配置失败: %v\n", err)
		os.Exit(1)
	}

	// 10. Restart service
	serviceRestart()

	variant, _ := internal.ActiveConfigVariant()
	if variant == "" {
		variant = "未设置"
	}
	fmt.Printf("✨ 同步完成！轨道：%s，变体：%s，模板来源：URL 分发\n", profile, variant)
}

func cmdValidate() {
	// 1. Read template URLs
	urls, err := internal.ReadEnvURLs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 2. Download
	downloaded, failed, err := internal.DownloadConfigs(urls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 下载失败: %v\n", err)
		os.Exit(1)
	}

	// 3. Verify
	if err := internal.VerifyDownloads(urls, downloaded, failed); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 4. Copy to ConfigDir
	configDir := internal.ConfigDir()
	if configDir == "" {
		fmt.Fprintf(os.Stderr, "❌ 无法确定配置目录\n")
		os.Exit(1)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建配置目录失败: %v\n", err)
		os.Exit(1)
	}
	for filename, srcPath := range downloaded {
		dstPath := filepath.Join(configDir, filename)
		if err := copyFile(srcPath, dstPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 复制文件失败 (%s): %v\n", filename, err)
			os.Exit(1)
		}
	}

	// 5. Find active variant template
	templatePath, err := internal.ActiveVariantTemplatePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 6. Load env
	vars, err := internal.LoadEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
	if missing := internal.RequireEnvVars(vars); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "❌ 缺少必需环境变量: %s\n", strings.Join(missing, ", "))
		os.Exit(1)
	}

	// Create temp file for rendering
	tmpFile, err := os.CreateTemp("", "sbc-validate.*.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建临时文件失败: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())

	// 7. Render
	if err := internal.RenderProfile(templatePath, tmpFile.Name(), vars); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 渲染失败 (%s): %v\n", templatePath, err)
		os.Exit(1)
	}

	// 8. Syntax check (no deploy)
	singBoxBin, err := internal.SingBoxBin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	checkCmd := exec.Command(singBoxBin, "check", "-c", tmpFile.Name())
	checkCmd.Stdout = os.Stdout
	checkCmd.Stderr = os.Stderr
	if err := checkCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 模板渲染语法失败：%s\n", templatePath)
		os.Exit(1)
	}

	fmt.Printf("✅ 模板渲染语法通过：%s\n", templatePath)
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

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}
