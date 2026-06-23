package internal

import (
	"os"
	"path/filepath"
	"testing"
)

// ── TestStripJSONCComments & TestResolvePlaceholders ──────────────────────────────────

func TestStripJSONCComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line comment",
			input:    `{"a": 1} // comment`,
			expected: `{"a": 1} `,
		},
		{
			name:     "multi line comment",
			input:    `{"a": /* comment */ 1}`,
			expected: `{"a":  1}`,
		},
		{
			name:     "comment in string is preserved",
			input:    `{"url": "https://example.com"}`,
			expected: `{"url": "https://example.com"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripJSONCComments(tc.input)
			if got != tc.expected {
				t.Errorf("stripJSONCComments(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestResolvePlaceholders(t *testing.T) {
	config := map[string]interface{}{
		"port":      1080,
		"username":  "testuser",
		"addresses": []interface{}{"1.1.1.1", "8.8.8.8"},
	}

	inputMap := map[string]interface{}{
		"port":      "sbc:port",
		"user":      "sbc:username",
		"addresses": "sbc:addresses",
		"plain":     "just string",
	}

	resolved := resolvePlaceholders(inputMap, config).(map[string]interface{})

	if resolved["port"] != 1080 {
		t.Errorf("expected port to be 1080, got %v", resolved["port"])
	}
	if resolved["user"] != "testuser" {
		t.Errorf("expected user to be 'testuser', got %v", resolved["user"])
	}
	if resolved["plain"] != "just string" {
		t.Errorf("expected plain to be 'just string', got %v", resolved["plain"])
	}
}

// ── TestRenderProfile ─────────────────────────────────────────────────────

func TestRenderProfile(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal config template
	templateContent := `{
  // comment
  "inbounds": [
    {
      "type": "mixed",
      "tag": "mixed-in",
      "listen": "127.0.0.1",
      "listen_port": "sbc:listen_port",
      "users": [
        {
          "username": "sbc:mixed_proxy_username",
          "password": "sbc:mixed_proxy_password"
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

	vars := map[string]interface{}{
		"listen_port":          1080,
		"mixed_proxy_username": "testuser",
		"mixed_proxy_password": "testpass",
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
	if !contains(rendered, `"listen_port": 1080`) {
		t.Errorf("listen_port not found in rendered output:\n%s", rendered)
	}
	if !contains(rendered, "testuser") {
		t.Errorf("mixed_proxy_username not found in rendered output:\n%s", rendered)
	}
	if !contains(rendered, "testpass") {
		t.Errorf("mixed_proxy_password not found in rendered output:\n%s", rendered)
	}
	if !contains(rendered, "inbounds") {
		t.Errorf("inbounds key not found in rendered output")
	}
}

func TestRenderProfileMissingTemplate(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "rendered.json")
	// Use a non-existent template path
	err := RenderProfile("/nonexistent/template.json", outputPath, map[string]interface{}{"KEY": "VALUE"})
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
