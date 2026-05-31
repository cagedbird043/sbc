package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config {status|show|edit|diff|variant|template|env}",
	Short: "配置管理",
	Long:  "管理 sing-box 配置模板、变体、渲染和部署。",
}

// config status
var configStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "当前配置状态",
	Run: func(cmd *cobra.Command, args []string) {
		configStatus()
	},
}

// config show
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "查看渲染后的配置内容",
	Run: func(cmd *cobra.Command, args []string) {
		configShow()
	},
}

// config edit
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "用编辑器打开模板",
	Run: func(cmd *cobra.Command, args []string) {
		configEdit()
	},
}

// config diff
var configDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "比较模板 vs 已部署配置",
	Run: func(cmd *cobra.Command, args []string) {
		configDiff()
	},
}

// config variant
var configVariantCmd = &cobra.Command{
	Use:   "variant {set|list}",
	Short: "管理配置变体",
	Run: func(cmd *cobra.Command, args []string) {
		variant, err := internal.ActiveConfigVariant()
		if err != nil {
			variant = "default"
		}
		stateFile, _ := internal.VariantStateFile()
		fmt.Printf("variant=%s\nstate_file=%s\n", variant, stateFile)
		fmt.Println("default=fakeip+prefer_ipv4 · realip-v4-only=real IP + ipv4_only fallback")
	},
}

var configVariantSetCmd = &cobra.Command{
	Use:   "set <变体>",
	Short: "切换配置变体 (default|realip-v4-only)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		variant := args[0]
		if err := internal.SetConfigVariant(variant); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		normalized, _ := internal.NormalizeConfigVariant(variant)
		fmt.Printf("✅ 已切换配置变体：%s\n", normalized)
		fmt.Fprintf(os.Stderr, "⚠ 执行 sbc update 后才会部署到 %s。\n", internal.TargetConf())
	},
}

var configVariantListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用变体",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("default        FakeIP + prefer_ipv4（主流）")
		fmt.Println("realip-v4-only Real IP + IPv4-only（保守备用）")
	},
}

// config template
var configTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "模板路径信息",
	Run: func(cmd *cobra.Command, args []string) {
		profile := internal.Profile()
		templatePath, err := internal.TemplatePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("profile=%s\n", profile)
		fmt.Printf("template=%s\n", templatePath)
	},
}

// config env
var configEnvCmd = &cobra.Command{
	Use:   "env",
	Short: ".env 文件信息",
	Run: func(cmd *cobra.Command, args []string) {
		configEnv()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configStatusCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configDiffCmd)
	configCmd.AddCommand(configVariantCmd)
	configVariantCmd.AddCommand(configVariantSetCmd)
	configVariantCmd.AddCommand(configVariantListCmd)
	configCmd.AddCommand(configTemplateCmd)
	configCmd.AddCommand(configEnvCmd)
}

func configStatus() {
	variant, _ := internal.ActiveConfigVariant()
	templatePath, _ := internal.TemplatePath()
	envFile, _ := internal.EnvFilePath()
	platform := internal.Platform()
	profile := internal.Profile()
	target := internal.TargetConf()

	fmt.Printf("变体:       %s （%s）\n", variant, internal.VariantDescription(variant))
	fmt.Printf("模板:       %s\n", templatePath)
	fmt.Printf("目标:       %s\n", target)
	fmt.Printf("环境变量:   %s\n", envFile)
	fmt.Printf("平台:       %s\n", platform)
	fmt.Printf("配置轨道:   %s\n", profile)
}

func configShow() {
	tmpFile, err := os.CreateTemp("", "sbc-show.*.json")
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

	data, _ := os.ReadFile(tmpFile.Name())
	fmt.Print(string(data))
}

func configEdit() {
	templatePath, err := internal.TemplatePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	if err := internal.RequirePrivateRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		for _, e := range []string{"vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		fmt.Fprintf(os.Stderr, "❌ 未找到编辑器。请设置 EDITOR 环境变量。\n")
		os.Exit(1)
	}

	fmt.Printf("📝 打开模板: %s\n", templatePath)
	editCmd := exec.Command(editor, templatePath)
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = os.Stdout
	editCmd.Stderr = os.Stderr
	if err := editCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 编辑器退出异常: %v\n", err)
		os.Exit(1)
	}
}

func configDiff() {
	if err := internal.RequirePrivateRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	target := internal.TargetConf()
	if _, err := os.Stat(target); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ 已部署配置不存在: %s\n", target)
		fmt.Fprintf(os.Stderr, "⚠ 先执行 sbc update 部署后再 diff。\n")
		os.Exit(1)
	}

	tmpFile, err := os.CreateTemp("", "sbc-diff.*.json")
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

	if err := internal.RenderProfile(tmpFile.Name(), vars); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 渲染失败: %v\n", err)
		os.Exit(1)
	}

	// Run diff
	diffCmd := exec.Command("diff", "-u", target, tmpFile.Name())
	diffCmd.Stdout = os.Stdout
	diffCmd.Stderr = os.Stderr
	if err := diffCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// Diff found, that's expected behavior - not an error
				return
			}
		}
		// diff exit code >1 means actual error
		fmt.Fprintf(os.Stderr, "⚠ 差异如上（- 已部署 / + 模板渲染）\n")
		return
	}
	fmt.Println("✅ 模板与已部署配置一致。")
}

func configEnv() {
	envFile, _ := internal.EnvFilePath()

	vars, err := internal.LoadEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("env_file=%s\n", envFile)
	printEnvVal(vars, "CLASH_API_SECRET")
	printEnvVal(vars, "MIXED_PROXY_USERNAME")
	printEnvVal(vars, "MIXED_PROXY_PASSWORD")
	printEnvVal(vars, "PROVIDER_NAME_1")
	if sub := vars["SUB_URL_1"]; sub != "" {
		fmt.Printf("SUB_URL_1=已设置（%d 字符）\n", len(sub))
	} else {
		fmt.Println("SUB_URL_1=未设置")
	}
	printEnvVal(vars, "TAILNET_AUTH_KEY")
}

func printEnvVal(vars map[string]string, key string) {
	if val, ok := vars[key]; ok && val != "" {
		fmt.Printf("%s=已设置\n", key)
	} else {
		fmt.Printf("%s=未设置\n", key)
	}
}

// requireControllerEnv loads env and ensures CLASH_API_SECRET is present.
func requireControllerEnv() (map[string]string, error) {
	vars, err := internal.LoadEnv()
	if err != nil {
		return nil, err
	}
	if vars["CLASH_API_SECRET"] == "" {
		return nil, fmt.Errorf("缺少 CLASH_API_SECRET，无法访问 9090 控制接口。")
	}
	return vars, nil
}

// selectorPath returns URL-encoded path for a selector.
func selectorPath(name string) string {
	// URL path encoding for proxy names
	encoded := strings.ReplaceAll(name, " ", "%20")
	encoded = strings.ReplaceAll(encoded, "#", "%23")
	encoded = strings.ReplaceAll(encoded, "&", "%26")
	return "/proxies/" + encoded
}

// renderProfileAndSave renders template and saves to a temp file.
// Returns the file handle; caller must close and remove.
func renderProfileAndSave() (*os.File, error) {
	vars, err := internal.LoadEnv()
	if err != nil {
		return nil, err
	}

	if missing := internal.RequireEnvVars(vars); len(missing) > 0 {
		return nil, fmt.Errorf("缺少必需环境变量: %s", strings.Join(missing, ", "))
	}

	tmpFile, err := os.CreateTemp("", "sbc-render.*.json")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}

	if err := internal.RenderProfile(tmpFile.Name(), vars); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("渲染失败: %w", err)
	}

	return tmpFile, nil
}

// Ensure bytes import is used (for consistency)
var _ io.Reader = bytes.NewReader(nil)
