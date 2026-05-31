package internal

import (
	"os"
	"path/filepath"
	"testing"
)

// ── TestExpandEnvsubst ────────────────────────────────────────────────────

func TestExpandEnvsubst(t *testing.T) {
	vars := map[string]string{
		"CLASH_API_SECRET":     "mysecret",
		"MIXED_PROXY_USERNAME": "user",
		"MIXED_PROXY_PASSWORD": "pass",
		"SUB_URL_1":            "https://example.com/sub",
		"EMPTY_VAR":            "",
		"HOSTNAME":             "hk-edge",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dollar-var simple",
			input:    "$CLASH_API_SECRET",
			expected: "mysecret",
		},
		{
			name:     "brace-var",
			input:    "${CLASH_API_SECRET}",
			expected: "mysecret",
		},
		{
			name:     "double-dollar becomes literal dollar",
			input:    "$$not_a_var",
			expected: "$",
		},
		{
			name:     "mixed literal and var",
			input:    "http://127.0.0.1:9090?secret=$CLASH_API_SECRET",
			expected: "http://127.0.0.1:9090?secret=mysecret",
		},
		{
			name:     "multiple vars",
			input:    "$MIXED_PROXY_USERNAME:$MIXED_PROXY_PASSWORD",
			expected: "user:pass",
		},
		{
			name:     "missing variable replaced with empty",
			input:    "$NONEXISTENT",
			expected: "",
		},
		{
			name:     "var followed by underscore is part of var name",
			input:    "BEFORE_$NONEXISTENT_AFTER",
			expected: "BEFORE_",
		},
		{
			name:     "brace-var with underscores",
			input:    "${MIXED_PROXY_USERNAME}@${HOSTNAME}",
			expected: "user@hk-edge",
		},
		{
			name:     "empty value variable",
			input:    "value=[${EMPTY_VAR}]",
			expected: "value=[]",
		},
		{
			name:     "url with var",
			input:    "${SUB_URL_1}",
			expected: "https://example.com/sub",
		},
		{
			name:     "no vars",
			input:    "plain text no variables",
			expected: "plain text no variables",
		},
		{
			name:     "dollar at end of string",
			input:    "trailing$",
			expected: "trailing$",
		},
		{
			name:     "dollar followed by space not a var",
			input:    "$ NOT_A_VAR",
			expected: "$ NOT_A_VAR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := expandEnvsubst(tc.input, vars)
			if got != tc.expected {
				t.Errorf("expandEnvsubst(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestExpandEnvsubstNoVars(t *testing.T) {
	// Empty vars map — all variables should be replaced with empty
	got := expandEnvsubst("hello $NONEXISTENT world", nil)
	if got != "hello  world" {
		t.Errorf("expected 'hello  world', got %q", got)
	}
}

// ── TestRenderProfile ─────────────────────────────────────────────────────

func TestRenderProfile(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal config template
	templateContent := `{
  "inbounds": [
    {
      "type": "mixed",
      "tag": "mixed-in",
      "listen": "127.0.0.1",
      "listen_port": "$LISTEN_PORT",
      "users": [
        {
          "username": "${MIXED_PROXY_USERNAME}",
          "password": "${MIXED_PROXY_PASSWORD}"
        }
      ]
    }
  ]
}`

	// Set up profile directory structure: SBC_TEMPLATE_ROOT/profiles/linux/config.template.json
	profileDir := filepath.Join(dir, "profiles", "linux")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(profileDir, "config.template.json")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override TemplateRoot to use our temp dir
	oldSBCProfile := os.Getenv("SBC_PROFILE")
	os.Setenv("SBC_PROFILE", "linux")
	os.Setenv("SBC_TEMPLATE_ROOT", dir)
	defer func() {
		os.Setenv("SBC_PROFILE", oldSBCProfile)
		os.Unsetenv("SBC_TEMPLATE_ROOT")
	}()

	vars := map[string]string{
		"LISTEN_PORT":          "1080",
		"MIXED_PROXY_USERNAME": "testuser",
		"MIXED_PROXY_PASSWORD": "testpass",
	}

	outputPath := filepath.Join(dir, "rendered.json")
	// Template is at templatePath, output to outputPath
	if err := RenderProfile(templatePath, outputPath, vars); err != nil {
		t.Fatalf("RenderProfile failed: %v", err)
	}

	// Read rendered output
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("cannot read rendered output: %v", err)
	}

	rendered := string(data)

	// Verify substitutions
	if !contains(rendered, `"listen_port": "1080"`) &&
		!contains(rendered, "1080") {
		t.Errorf("LISTEN_PORT not found in rendered output:\n%s", rendered)
	}
	if !contains(rendered, "testuser") {
		t.Errorf("MIXED_PROXY_USERNAME not found in rendered output:\n%s", rendered)
	}
	if !contains(rendered, "testpass") {
		t.Errorf("MIXED_PROXY_PASSWORD not found in rendered output:\n%s", rendered)
	}
	if !contains(rendered, "inbounds") {
		t.Errorf("inbounds key not found in rendered output")
	}
}

func TestRenderProfileMissingTemplate(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "rendered.json")
	// Use a non-existent template path
	err := RenderProfile("/nonexistent/template.json", outputPath, map[string]string{"KEY": "VALUE"})
	if err == nil {
		t.Fatal("expected error for missing template file, got nil")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
