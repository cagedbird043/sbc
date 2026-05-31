package internal

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// TemplatePath returns the full path to the config template for the current profile.
func TemplatePath() (string, error) {
	templateRoot, err := TemplateRoot()
	if err != nil {
		return "", err
	}
	profile := Profile()
	return templateRoot + "/profiles/" + profile + "/config.template.json", nil
}

// RequirePrivateRepo checks that the template root is a valid git repo with a template file.
func RequirePrivateRepo() error {
	templateRoot, err := TemplateRoot()
	if err != nil {
		return err
	}
	gitDir := templateRoot + "/.git"
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		return fmt.Errorf("未发现私有模板仓：%s\n请先克隆 git@github.com:cagedbird043/sing-box-private-prod.git，或设置 SBC_TEMPLATE_ROOT。", templateRoot)
	}

	templatePath, err := TemplatePath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("未发现模板文件：%s", templatePath)
	}
	return nil
}

// expandEnvsubst replaces $VAR and ${VAR} with values from vars.
// If a variable is not found, it's replaced with empty string (envsubst behavior).
func expandEnvsubst(input string, vars map[string]string) string {
	// Match ${VAR} first, then $VAR (alphanumeric + underscore only)
	result := regexp.MustCompile(`\$\{([^}]+)\}`).ReplaceAllStringFunc(input, func(match string) string {
		key := match[2 : len(match)-1]
		if val, ok := vars[key]; ok {
			return val
		}
		return ""
	})
	result = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`).ReplaceAllStringFunc(result, func(match string) string {
		key := match[1:]
		if val, ok := vars[key]; ok {
			return val
		}
		return ""
	})
	return result
}

// RenderProfile renders the template with envsubst semantics and writes to outputPath.
func RenderProfile(outputPath string, vars map[string]string) error {
	templatePath, err := TemplatePath()
	if err != nil {
		return err
	}

	// Check for variant-specific template (realip-v4-only)
	variant, _ := ActiveConfigVariant()
	switch variant {
	case DefaultConfigVariantRealIP:
		templateRoot, _ := TemplateRoot()
		profile := Profile()
		realipPath := templateRoot + "/profiles/" + profile + "-realip/config.template.json"
		if _, err := os.Stat(realipPath); err == nil {
			templatePath = realipPath
		}
	}

	tplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("无法读取模板文件 %s: %w", templatePath, err)
	}

	rendered := expandEnvsubst(string(tplContent), vars)

	if err := os.WriteFile(outputPath, []byte(rendered), 0600); err != nil {
		return fmt.Errorf("写入渲染结果失败: %w", err)
	}

	return nil
}

// SyncPrivateRepo runs git pull --ff-only on the template root.
func SyncPrivateRepo() error {
	if err := RequirePrivateRepo(); err != nil {
		return err
	}
	templateRoot, _ := TemplateRoot()

	cmd := exec.Command("git", "-C", templateRoot, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull --ff-only 失败: %w", err)
	}
	return nil
}

// SingBoxBin finds the sing-box binary. Returns path or error.
func SingBoxBin() (string, error) {
	// Try command -v
	path, err := exec.LookPath("sing-box")
	if err == nil {
		return path, nil
	}
	// macOS homebrew path
	if _, err := os.Stat("/opt/homebrew/bin/sing-box"); err == nil {
		return "/opt/homebrew/bin/sing-box", nil
	}
	// Linux default path
	if _, err := os.Stat("/usr/bin/sing-box"); err == nil {
		return "/usr/bin/sing-box", nil
	}
	return "", fmt.Errorf("未找到 sing-box 可执行文件。")
}

// InstallConfig copies the rendered config to the target path.
func InstallConfig(renderedConf string) error {
	profile := Profile()
	target := TargetConf()

	switch profile {
	case "macos":
		// Ensure parent dir exists with current user as owner
		parentDir := target[:len(target)-len("/config.json")] // dirname
		_ = os.MkdirAll(parentDir, 0755)
		data, err := os.ReadFile(renderedConf)
		if err != nil {
			return fmt.Errorf("读取渲染配置失败: %w", err)
		}
		return os.WriteFile(target, data, 0600)
	default:
		// Linux: use sudo cp
		cmd := exec.Command("sudo", "cp", renderedConf, target)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}
