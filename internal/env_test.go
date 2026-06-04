package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `# sing-box credentials
CLASH_API_SECRET=mysecret123
MIXED_PROXY_USERNAME=proxyuser
MIXED_PROXY_PASSWORD=proxypass
PROVIDER_NAME_1=my-provider
SUB_URL_1=https://example.com/sub
TAILNET_AUTH_KEY=tskey-auth-xxx
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := ReadEnvFile(envFile)
	if err != nil {
		t.Fatalf("ReadEnvFile failed: %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"CLASH_API_SECRET", "mysecret123"},
		{"MIXED_PROXY_USERNAME", "proxyuser"},
		{"MIXED_PROXY_PASSWORD", "proxypass"},
		{"PROVIDER_NAME_1", "my-provider"},
		{"SUB_URL_1", "https://example.com/sub"},
		{"TAILNET_AUTH_KEY", "tskey-auth-xxx"},
	}
	for _, tc := range tests {
		if got := vars[tc.key]; got != tc.expected {
			t.Errorf("vars[%q] = %q, want %q", tc.key, got, tc.expected)
		}
	}
}

func TestReadEnvFileQuotedValues(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `KEY="quoted value"
EMPTY=""
NUMBER=123
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := ReadEnvFile(envFile)
	if err != nil {
		t.Fatalf("ReadEnvFile failed: %v", err)
	}

	if vars["KEY"] != "quoted value" {
		t.Errorf("KEY = %q, want 'quoted value'", vars["KEY"])
	}
	if vars["EMPTY"] != "" {
		t.Errorf("EMPTY = %q, want ''", vars["EMPTY"])
	}
	if vars["NUMBER"] != "123" {
		t.Errorf("NUMBER = %q, want '123'", vars["NUMBER"])
	}
}

func TestReadEnvFileSingleQuotedValues(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `SINGLE_QUOTED='https://example.com/path?query=value'
EMPTY_SINGLE=''
UNQUOTED=bare-value
MISMATCHED='val'ue'
DOUBLE_INSIDE='he said "hello"'
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := ReadEnvFile(envFile)
	if err != nil {
		t.Fatalf("ReadEnvFile failed: %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"SINGLE_QUOTED", "https://example.com/path?query=value"},
		{"EMPTY_SINGLE", ""},
		{"UNQUOTED", "bare-value"},
		{"MISMATCHED", "val'ue"},    // 首尾配对就剥，内部引号不管
		{"DOUBLE_INSIDE", "he said \"hello\""},
	}
	for _, tc := range tests {
		if got := vars[tc.key]; got != tc.expected {
			t.Errorf("vars[%q] = %q, want %q", tc.key, got, tc.expected)
		}
	}
}

func TestReadEnvFileCommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `# This is a comment

KEY=value
# Another comment

EMPTY_LINE_AFTER=yes
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := ReadEnvFile(envFile)
	if err != nil {
		t.Fatalf("ReadEnvFile failed: %v", err)
	}

	if vars["KEY"] != "value" {
		t.Errorf("KEY = %q, want 'value'", vars["KEY"])
	}
	if vars["EMPTY_LINE_AFTER"] != "yes" {
		t.Errorf("EMPTY_LINE_AFTER = %q, want 'yes'", vars["EMPTY_LINE_AFTER"])
	}
	// Comment-only lines shouldn't produce entries
	if _, ok := vars["This is a comment"]; ok {
		t.Error("comment line produced a key")
	}
}

func TestReadEnvFileMissingFile(t *testing.T) {
	_, err := ReadEnvFile("/nonexistent/path/.env")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadEnvFileEmptyFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := ReadEnvFile(envFile)
	if err != nil {
		t.Fatalf("ReadEnvFile failed: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("expected empty vars, got %d entries", len(vars))
	}
}

func TestReadEnvFileTrailingWhitespace(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := "KEY = value  \n"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := ReadEnvFile(envFile)
	if err != nil {
		t.Fatalf("ReadEnvFile failed: %v", err)
	}
	if vars["KEY"] != "value" {
		t.Errorf("KEY = %q, want 'value'", vars["KEY"])
	}
}

func TestRequireEnvVars(t *testing.T) {
	vars := map[string]string{
		"CLASH_API_SECRET":     "secret",
		"MIXED_PROXY_USERNAME": "user",
		"MIXED_PROXY_PASSWORD": "pass",
		"PROVIDER_NAME_1":      "provider",
		"SUB_URL_1":            "url",
	}

	if missing := RequireEnvVars(vars); len(missing) > 0 {
		t.Errorf("expected no missing vars, got: %v", missing)
	}
}

func TestRequireEnvVarsMissing(t *testing.T) {
	vars := map[string]string{
		"CLASH_API_SECRET": "secret",
		// MIXED_PROXY_USERNAME missing
		"MIXED_PROXY_PASSWORD": "pass",
		// PROVIDER_NAME_1 missing
		"SUB_URL_1": "url",
	}

	missing := RequireEnvVars(vars)
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing vars, got %d: %v", len(missing), missing)
	}
	if missing[0] != "MIXED_PROXY_USERNAME" {
		t.Errorf("expected MIXED_PROXY_USERNAME first, got %s", missing[0])
	}
	if missing[1] != "PROVIDER_NAME_1" {
		t.Errorf("expected PROVIDER_NAME_1 second, got %s", missing[1])
	}
}

func TestRequireEnvVarsEmpty(t *testing.T) {
	vars := map[string]string{}
	missing := RequireEnvVars(vars)
	if len(missing) != 5 {
		t.Errorf("expected 5 missing vars, got %d", len(missing))
	}
}

func TestRequireEnvFile(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// No .env file exists
	err := RequireEnvFile()
	if err == nil {
		t.Fatal("expected error when no .env exists, got nil")
	}
}

func TestLoadEnv(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	confDir := filepath.Join(dir, ".config", "sing-box")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(confDir, ".env")
	if err := os.WriteFile(envFile, []byte("KEY=value\n"), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := LoadEnv()
	if err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}
	if vars["KEY"] != "value" {
		t.Errorf("KEY = %q, want 'value'", vars["KEY"])
	}
}
