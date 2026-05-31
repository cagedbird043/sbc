package internal

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

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

// RenderProfile reads the template at templatePath, substitutes variables
// with envsubst semantics, and writes the result to outputPath.
// The caller is responsible for choosing the correct template file (variant).
func RenderProfile(templatePath, outputPath string, vars map[string]string) error {
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
