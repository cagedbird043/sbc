package internal

import (
	"fmt"
	"os"
	"strings"
)

// NormalizeConfigVariant normalizes user input by lowercasing and trimming.
// No hardcoded names — validation is deferred to ListAvailableVariants
// which scans the actual files on disk.
func NormalizeConfigVariant(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

// ActiveConfigVariant reads the current variant from state file, env, or default.
func ActiveConfigVariant() (string, error) {
	// 1. Environment variable override
	if envVar := os.Getenv("SBC_CONFIG_VARIANT"); envVar != "" {
		return NormalizeConfigVariant(envVar), nil
	}

	// 2. State file
	stateFile, err := VariantStateFile()
	if err != nil {
		return "", fmt.Errorf("无法确定配置目录: %w", err)
	}
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "", fmt.Errorf("未设置配置变体。请先执行 'sbc config variant set <变体>'")
	}

	return NormalizeConfigVariant(strings.TrimSpace(string(data))), nil
}

// SetConfigVariant writes the normalized variant name to the state file.
func SetConfigVariant(variant string) error {
	normalized := NormalizeConfigVariant(variant)

	stateFile, err := VariantStateFile()
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	confDir, _ := ConfDir()
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("无法创建配置目录 %s: %w", confDir, err)
	}

	return os.WriteFile(stateFile, []byte(normalized+"\n"), 0644)
}

// VariantDescription returns a human-readable description for a variant.
func VariantDescription(variant string) string {
	switch variant {
	case DefaultConfigVariant, "fakeip-prefer-ipv4":
		return "fakeip+prefer_ipv4"
	case DefaultConfigVariantRealIP:
		return "real IP + ipv4_only fallback"
	default:
		return variant
	}
}
