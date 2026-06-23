package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func stripJSONCComments(s string) string {
	var out strings.Builder
	inStr := false
	esc := false
	runes := []rune(s)
	n := len(runes)
	i := 0
	for i < n {
		r := runes[i]
		if inStr {
			out.WriteRune(r)
			if esc {
				esc = false
			} else if r == '\\' {
				esc = true
			} else if r == '"' {
				inStr = false
			}
			i++
			continue
		}
		if r == '"' {
			inStr = true
			out.WriteRune(r)
			i++
			continue
		}
		if r == '/' && i+1 < n && runes[i+1] == '/' {
			i += 2
			for i < n && runes[i] != '\n' && runes[i] != '\r' {
				i++
			}
			continue
		}
		if r == '/' && i+1 < n && runes[i+1] == '*' {
			i += 2
			for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		out.WriteRune(r)
		i++
	}
	return out.String()
}

func resolvePlaceholders(val interface{}, config map[string]interface{}) interface{} {
	switch v := val.(type) {
	case map[string]interface{}:
		for k, child := range v {
			v[k] = resolvePlaceholders(child, config)
		}
		return v
	case []interface{}:
		for i, child := range v {
			v[i] = resolvePlaceholders(child, config)
		}
		return v
	case string:
		if strings.HasPrefix(v, "sbc:") {
			key := v[4:]
			if resolved, ok := config[key]; ok {
				return resolved
			}
		}
		return v
	default:
		return v
	}
}

func RenderProfile(templatePath, outputPath string, config map[string]interface{}) error {
	tplContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("无法读取模板文件 %s: %w", templatePath, err)
	}

	cleanJSON := stripJSONCComments(string(tplContent))
	var parsed interface{}
	if err := json.Unmarshal([]byte(cleanJSON), &parsed); err != nil {
		return fmt.Errorf("解析模板 JSON 失败: %w", err)
	}

	resolved := resolvePlaceholders(parsed, config)

	renderedBytes, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化渲染配置失败: %w", err)
	}

	if err := os.WriteFile(outputPath, renderedBytes, 0600); err != nil {
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
