package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeConfigVariant(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"default", "default"},
		{"DEFAULT", "default"},
		{"REALIP", "realip"},
		{"realip-v4-only", "realip-v4-only"},
		{"  fakeip-prefer-ipv4  ", "fakeip-prefer-ipv4"},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		result := NormalizeConfigVariant(tc.input)
		if result != tc.expected {
			t.Errorf("NormalizeConfigVariant(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestActiveConfigVariantUnset(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)
	os.Unsetenv("SBC_CONFIG_VARIANT")

	_, err := ActiveConfigVariant()
	if err == nil {
		t.Fatal("expected error when no variant is set, got nil")
	}
}

func TestActiveConfigVariantFromEnv(t *testing.T) {
	os.Setenv("SBC_CONFIG_VARIANT", "realip-v4-only")
	defer os.Unsetenv("SBC_CONFIG_VARIANT")

	variant, err := ActiveConfigVariant()
	if err != nil {
		t.Fatalf("ActiveConfigVariant failed: %v", err)
	}
	if variant != "realip-v4-only" {
		t.Errorf("expected 'realip-v4-only', got %q", variant)
	}
}

func TestActiveConfigVariantFromEnvNormalized(t *testing.T) {
	os.Setenv("SBC_CONFIG_VARIANT", "REALIP")
	defer os.Unsetenv("SBC_CONFIG_VARIANT")

	variant, err := ActiveConfigVariant()
	if err != nil {
		t.Fatalf("ActiveConfigVariant failed: %v", err)
	}
	if variant != "realip" {
		t.Errorf("expected 'realip', got %q", variant)
	}
}

func TestActiveConfigVariantFromStateFile(t *testing.T) {
	os.Unsetenv("SBC_CONFIG_VARIANT")

	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	stateDir := filepath.Join(dir, ".config", "sing-box")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	stateFile := filepath.Join(stateDir, "config-variant")
	if err := os.WriteFile(stateFile, []byte("realip-v4-only\n"), 0644); err != nil {
		t.Fatal(err)
	}

	variant, err := ActiveConfigVariant()
	if err != nil {
		t.Fatalf("ActiveConfigVariant failed: %v", err)
	}
	if variant != "realip-v4-only" {
		t.Errorf("expected 'realip-v4-only', got %q", variant)
	}

	// Cleanup
	os.RemoveAll(stateDir)
}

func TestActiveConfigVariantStateFilePriority(t *testing.T) {
	// Env var should override state file
	os.Setenv("SBC_CONFIG_VARIANT", "realip-v4-only")
	defer os.Unsetenv("SBC_CONFIG_VARIANT")

	variant, err := ActiveConfigVariant()
	if err != nil {
		t.Fatalf("ActiveConfigVariant failed: %v", err)
	}
	if variant != "realip-v4-only" {
		t.Errorf("expected 'realip-v4-only', got %q", variant)
	}
}

func TestSetConfigVariant(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	if err := SetConfigVariant("realip-v4-only"); err != nil {
		t.Fatalf("SetConfigVariant failed: %v", err)
	}

	stateFile, _ := VariantStateFile()
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("cannot read state file: %v", err)
	}
	if string(data) != "realip-v4-only\n" {
		t.Errorf("state file content = %q, want 'realip-v4-only\\n'", string(data))
	}
}

func TestSetConfigVariantAnyName(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	if err := SetConfigVariant("custom-variant"); err != nil {
		t.Fatalf("SetConfigVariant('custom-variant') failed: %v", err)
	}

	stateFile, _ := VariantStateFile()
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("cannot read state file: %v", err)
	}
	if string(data) != "custom-variant\n" {
		t.Errorf("state file content = %q, want 'custom-variant\\n'", string(data))
	}
}

func TestSetConfigVariantNormalized(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	if err := SetConfigVariant("REALIP"); err != nil {
		t.Fatalf("SetConfigVariant('REALIP') failed: %v", err)
	}

	stateFile, _ := VariantStateFile()
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("cannot read state file: %v", err)
	}
	if string(data) != "realip\n" {
		t.Errorf("state file content = %q, want 'realip\\n'", string(data))
	}
}

func TestVariantDescription(t *testing.T) {
	tests := []struct {
		variant  string
		expected string
	}{
		{"default", "fakeip+prefer_ipv4"},
		{"realip-v4-only", "real IP + ipv4_only fallback"},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		if desc := VariantDescription(tc.variant); desc != tc.expected {
			t.Errorf("VariantDescription(%q) = %q, want %q", tc.variant, desc, tc.expected)
		}
	}
}

func TestVariantStateFile(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	stateFile, err := VariantStateFile()
	if err != nil {
		t.Fatalf("VariantStateFile failed: %v", err)
	}
	expected := filepath.Join(dir, ".config", "sing-box", "config-variant")
	if stateFile != expected {
		t.Errorf("VariantStateFile = %q, want %q", stateFile, expected)
	}
}
