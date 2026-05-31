package internal

import (
	"fmt"
	"os"
	"strings"
)

// NormalizeConfigVariant lowercases and trims the input.
// Validation is deferred — the filesystem (ListAvailableVariants) is the
// source of truth for which variants actually exist.
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
		return DefaultConfigVariant, nil
	}
	data, err := os.ReadFile(stateFile)
	if err == nil {
		return NormalizeConfigVariant(strings.TrimSpace(string(data))), nil
	}

	// 3. Default
	return DefaultConfigVariant, nil
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
