package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigDir returns ~/.config/sing-box/.  Returns empty string on error.
func ConfigDir() string {
	dir, err := ConfDir()
	if err != nil {
		return ""
	}
	return dir
}

// ListAvailableVariants scans ConfigDir for config-*.json files and
// extracts variant names.  Files ending with .meta.json are skipped.
//
// Example: "config-realip-v4-only.json" → "realip-v4-only"
func ListAvailableVariants() ([]string, error) {
	dir := ConfigDir()
	if dir == "" {
		return nil, fmt.Errorf("无法确定配置目录")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置目录 %s: %w", dir, err)
	}

	var variants []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		after, ok := strings.CutPrefix(name, "config-")
		if !ok {
			continue
		}
		// Skip companion .meta.json files
		if strings.HasSuffix(after, ".meta.json") {
			continue
		}
		// Extract variant name: strip trailing ".json"
		if variant, ok2 := strings.CutSuffix(after, ".json"); ok2 && variant != "" {
			variants = append(variants, variant)
		}
	}
	return variants, nil
}

// ActiveVariantTemplatePath reads the config-variant state file and returns
// the full path to the active variant's JSON template inside ConfigDir.
func ActiveVariantTemplatePath() (string, error) {
	variant, err := ActiveConfigVariant()
	if err != nil {
		return "", err
	}

	dir := ConfigDir()
	if dir == "" {
		return "", fmt.Errorf("无法确定配置目录")
	}

	templatePath := filepath.Join(dir, "config-"+variant+".json")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return "", fmt.Errorf("未找到变体 '%s' 的模板文件: %s\n请先运行 sbc update 下载模板，或 sbc config variant set 选择其他变体。", variant, templatePath)
	}
	return templatePath, nil
}
