package internal

import (
	"fmt"
	"os"
	"strings"
)

// NormalizeConfigVariant normalizes user input to canonical variant names.
func NormalizeConfigVariant(input string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "default", "main", "fakeip", "fakeip-prefer-ipv4", "fakeip_prefer_ipv4":
		return DefaultConfigVariant, nil
	case "realip", "realip-v4", "realip-v4-only", "realip_v4_only", "v4-only", "ipv4-only":
		return DefaultConfigVariantRealIP, nil
	default:
		return "", fmt.Errorf("未知配置变体: %s （可选: default, realip-v4-only）", input)
	}
}

// ActiveConfigVariant reads the current variant from state file, env, or default.
func ActiveConfigVariant() (string, error) {
	// 1. Environment variable override
	if envVar := os.Getenv("SBC_CONFIG_VARIANT"); envVar != "" {
		return NormalizeConfigVariant(envVar)
	}

	// 2. State file
	stateFile, err := VariantStateFile()
	if err != nil {
		return DefaultConfigVariant, nil
	}
	data, err := os.ReadFile(stateFile)
	if err == nil {
		return NormalizeConfigVariant(strings.TrimSpace(string(data)))
	}

	// 3. Default
	return DefaultConfigVariant, nil
}

// SetConfigVariant writes the variant to the state file.
func SetConfigVariant(variant string) error {
	normalized, err := NormalizeConfigVariant(variant)
	if err != nil {
		return err
	}

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
	case DefaultConfigVariant:
		return "fakeip+prefer_ipv4"
	case DefaultConfigVariantRealIP:
		return "real IP + ipv4_only fallback"
	default:
		return variant
	}
}
