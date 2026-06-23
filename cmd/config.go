package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config {status|show|diff|variant|template|env}",
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
			variant = "unknown"
		}
		stateFile, _ := internal.VariantStateFile()
		fmt.Printf("variant=%s\nstate_file=%s\n", variant, stateFile)
		if avail, err := internal.ListAvailableVariants(); err == nil && len(avail) > 0 {
			for _, v := range avail {
				fmt.Printf("  %s    %s\n", v, internal.VariantDescription(v))
			}
		}
	},
}

var configVariantSetCmd = &cobra.Command{
	Use:   "set <变体>",
	Short: "切换配置变体",
	Long:  "可用变体通过 sbc config variant list 查看。变体合法性由文件系统决定。",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		avail, err := internal.ListAvailableVariants()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return avail, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		variant := args[0]
		if err := internal.SetConfigVariant(variant); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		normalized := internal.NormalizeConfigVariant(variant)
		fmt.Printf("✅ 已切换配置变体：%s\n", normalized)
		fmt.Fprintf(os.Stderr, "⚠ 执行 sbc update 后才会部署到 %s。\n", internal.TargetConf())
	},
}

var configVariantListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用变体",
	Run: func(cmd *cobra.Command, args []string) {
		variants, err := internal.ListAvailableVariants()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		if len(variants) == 0 {
			fmt.Println("（暂无可用变体，请先运行 sbc update 下载模板）")
			return
		}
		for _, v := range variants {
			fmt.Printf("%s    %s\n", v, internal.VariantDescription(v))
		}
	},
}

// config template
var configTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "模板路径信息",
	Run: func(cmd *cobra.Command, args []string) {
		profile := internal.Profile()
		templatePath, err := internal.ActiveVariantTemplatePath()
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
	configCmd.AddCommand(configDiffCmd)
	configCmd.AddCommand(configVariantCmd)
	configVariantCmd.AddCommand(configVariantSetCmd)
	configVariantCmd.AddCommand(configVariantListCmd)
	configCmd.AddCommand(configTemplateCmd)
	configCmd.AddCommand(configEnvCmd)
}

func configStatus() {
	variant, _ := internal.ActiveConfigVariant()
	templatePath, _ := internal.ActiveVariantTemplatePath()
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
	fmt.Printf("模板来源:   URL 分发\n")
}

func configShow() {
	templatePath, err := internal.ActiveVariantTemplatePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

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

	if err := internal.RenderProfile(templatePath, tmpFile.Name(), vars); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 渲染失败: %v\n", err)
		os.Exit(1)
	}

	data, _ := os.ReadFile(tmpFile.Name())
	fmt.Print(string(data))
}

func configDiff() {
	templatePath, err := internal.ActiveVariantTemplatePath()
	if err != nil {
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

	if err := internal.RenderProfile(templatePath, tmpFile.Name(), vars); err != nil {
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
	configFile, _ := internal.ConfigFilePath()

	vars, err := internal.LoadEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("config_file=%s\n", configFile)
	printEnvVal(vars, "clash_api_secret")
	printEnvVal(vars, "mixed_proxy_username")
	printEnvVal(vars, "mixed_proxy_password")
	printEnvVal(vars, "provider_name_1")

	rawSub, ok := vars["sub_url_1"]
	if !ok || rawSub == nil {
		rawSub, ok = vars["SUB_URL_1"]
	}
	if ok && rawSub != nil {
		if sub, ok := rawSub.(string); ok && sub != "" {
			fmt.Printf("sub_url_1=已设置（%d 字符）\n", len(sub))
		} else {
			fmt.Println("sub_url_1=未设置")
		}
	} else {
		fmt.Println("sub_url_1=未设置")
	}
	printEnvVal(vars, "tailnet_auth_key")
}

func printEnvVal(vars map[string]interface{}, key string) {
	lowerKey := strings.ToLower(key)
	val, ok := vars[lowerKey]
	if !ok || val == nil {
		val, ok = vars[key]
	}
	if ok && val != nil {
		if str, ok := val.(string); ok && str != "" {
			fmt.Printf("%s=已设置\n", lowerKey)
			return
		} else if _, ok := val.(bool); ok {
			fmt.Printf("%s=已设置\n", lowerKey)
			return
		} else if slice, ok := val.([]interface{}); ok && len(slice) > 0 {
			fmt.Printf("%s=已设置\n", lowerKey)
			return
		}
	}
	fmt.Printf("%s=未设置\n", lowerKey)
}
