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
		wantErr  bool
	}{
		{"default", "default", false},
		{"main", "default", false},
		{"fakeip", "default", false},
		{"fakeip-prefer-ipv4", "default", false},
		{"fakeip_prefer_ipv4", "default", false},
		{"realip", "realip-v4-only", false},
		{"realip-v4", "realip-v4-only", false},
		{"realip-v4-only", "realip-v4-only", false},
		{"realip_v4_only", "realip-v4-only", false},
		{"v4-only", "realip-v4-only", false},
		{"ipv4-only", "realip-v4-only", false},
		// Case insensitivity
		{"Default", "default", false},
		{"REALIP", "realip-v4-only", false},
		{"FakeIP-Prefer-IPv4", "default", false},
		// Invalid
		{"unknown", "", true},
		{"", "", true},
		// Whitespace
		{"  default  ", "default", false},
	}
	for _, tc := range tests {
		result, err := NormalizeConfigVariant(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("NormalizeConfigVariant(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeConfigVariant(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if result != tc.expected {
			t.Errorf("NormalizeConfigVariant(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestActiveConfigVariantDefault(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)
	os.Unsetenv("SBC_CONFIG_VARIANT")

	variant, err := ActiveConfigVariant()
	if err != nil {
		t.Fatalf("ActiveConfigVariant failed: %v", err)
	}
	if variant != "default" {
		t.Errorf("expected 'default', got %q", variant)
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
	if variant != "realip-v4-only" {
		t.Errorf("expected 'realip-v4-only', got %q", variant)
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

func TestSetConfigVariantInvalid(t *testing.T) {
	err := SetConfigVariant("unknown")
	if err == nil {
		t.Fatal("expected error for invalid variant, got nil")
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
	if string(data) != "realip-v4-only\n" {
		t.Errorf("state file content = %q, want 'realip-v4-only\\n'", string(data))
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
