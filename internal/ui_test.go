package internal

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigUIName(t *testing.T) {
	dir := t.TempDir()
	origTarget := os.Getenv("TARGET_CONF")
	os.Setenv("SBC_PROFILE", "linux")
	defer func() {
		if origTarget != "" {
			os.Setenv("TARGET_CONF", origTarget)
		} else {
			os.Unsetenv("TARGET_CONF")
		}
		os.Unsetenv("SBC_PROFILE")
	}()

	// Override TargetConf via env for testing
	target := filepath.Join(dir, "config.json")
	os.Setenv("TARGET_CONF", target)

	configContent := `{
	  "experimental": {
	    "clash_api": {
	      "external_ui": "test-ui",
	      "external_ui_download_url": "https://example.com/ui.zip"
	    }
	  }
	}`
	if err := os.WriteFile(target, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	name, err := ConfigUIName()
	if err != nil {
		t.Fatalf("ConfigUIName failed: %v", err)
	}
	if name != "test-ui" {
		t.Errorf("ConfigUIName = %q, want 'test-ui'", name)
	}

	url, err := ConfigUIDownloadURL()
	if err != nil {
		t.Fatalf("ConfigUIDownloadURL failed: %v", err)
	}
	if url != "https://example.com/ui.zip" {
		t.Errorf("ConfigUIDownloadURL = %q, want 'https://example.com/ui.zip'", url)
	}
}

func TestConfigUINameDefaults(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("SBC_PROFILE", "linux")
	target := filepath.Join(dir, "config.json")
	os.Setenv("TARGET_CONF", target)
	defer func() {
		os.Unsetenv("SBC_PROFILE")
		os.Unsetenv("TARGET_CONF")
	}()

	// No config file exists
	name, err := ConfigUIName()
	if err != nil {
		t.Fatalf("ConfigUIName failed: %v", err)
	}
	if name != "zashboard" {
		t.Errorf("ConfigUIName = %q, want 'zashboard'", name)
	}

	url, err := ConfigUIDownloadURL()
	if err != nil {
		t.Fatalf("ConfigUIDownloadURL failed: %v", err)
	}
	if url != DefaultUIDownloadURL {
		t.Errorf("ConfigUIDownloadURL = %q, want %q", url, DefaultUIDownloadURL)
	}
}

func TestUIDestDir(t *testing.T) {
	tests := []struct {
		uiName   string
		expected string
	}{
		{"zashboard", filepath.Join(UIBaseDir(), "zashboard")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", filepath.Join(UIBaseDir(), "relative/path")},
	}
	for _, tc := range tests {
		dest := UIDestDir(tc.uiName)
		if dest != tc.expected {
			t.Errorf("UIDestDir(%q) = %q, want %q", tc.uiName, dest, tc.expected)
		}
	}
}

func TestExtractUIZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "ui.zip")
	outputDir := filepath.Join(dir, "extracted")

	// Create a test zip file with the GitHub archive structure (top-level directory)
	createTestZip(t, zipPath, map[string]string{
		"zashboard-gh-pages/index.html":      "<html></html>",
		"zashboard-gh-pages/assets/index.js": "console.log('test')",
		"zashboard-gh-pages/assets/style.css": "body { margin: 0 }",
	})

	if err := ExtractUIZip(zipPath, outputDir); err != nil {
		t.Fatalf("ExtractUIZip failed: %v", err)
	}

	// Verify extracted files (top-level dir stripped)
	expectedFiles := []string{
		"index.html",
		"assets/index.js",
		"assets/style.css",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("extracted file %q not found", path)
		}
	}
}

func TestExtractUIZipTopLevelDirStripped(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "ui.zip")
	outputDir := filepath.Join(dir, "extracted")

	// GitHub archive format: top-level-dir/path/to/file
	createTestZip(t, zipPath, map[string]string{
		"zashboard-gh-pages/index.html":       "<html></html>",
		"zashboard-gh-pages/sub/index.js":     "console.log('test')",
	})

	if err := ExtractUIZip(zipPath, outputDir); err != nil {
		t.Fatalf("ExtractUIZip failed: %v", err)
	}

	// Top-level dir should be stripped
	if _, err := os.Stat(filepath.Join(outputDir, "index.html")); os.IsNotExist(err) {
		t.Errorf("index.html not found in output (top-level dir should be stripped)")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "sub", "index.js")); os.IsNotExist(err) {
		t.Errorf("sub/index.js not found in output")
	}
	// Top-level directory itself should not be present
	if _, err := os.Stat(filepath.Join(outputDir, "zashboard-gh-pages")); !os.IsNotExist(err) {
		t.Errorf("top-level directory zashboard-gh-pages should have been stripped")
	}
}

func TestExtractUIZipInvalidFile(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "extracted")

	err := ExtractUIZip("/nonexistent/file.zip", outputDir)
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

func TestInstallUIDir(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	destDir := filepath.Join(dir, "dest")

	// Create source files
	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "index.html"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sub", "file.js"), []byte("js"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := InstallUIDir(srcDir, destDir); err != nil {
		t.Fatalf("InstallUIDir failed: %v", err)
	}

	// Verify files installed
	if _, err := os.Stat(filepath.Join(destDir, "index.html")); os.IsNotExist(err) {
		t.Errorf("index.html not installed")
	}
	if _, err := os.Stat(filepath.Join(destDir, "sub", "file.js")); os.IsNotExist(err) {
		t.Errorf("sub/file.js not installed")
	}
}

func TestInstallUIDirAtomicSwap(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	destDir := filepath.Join(dir, "dest")

	// Create old dest content
	oldContent := filepath.Join(destDir, "old.txt")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldContent, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create new source
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := InstallUIDir(srcDir, destDir); err != nil {
		t.Fatalf("InstallUIDir failed: %v", err)
	}

	// New content should be there (atomic swap)
	if _, err := os.Stat(filepath.Join(destDir, "new.txt")); os.IsNotExist(err) {
		t.Errorf("new.txt should exist after install")
	}
	// Old content should be gone
	if _, err := os.Stat(filepath.Join(destDir, "old.txt")); !os.IsNotExist(err) {
		t.Errorf("old.txt should NOT exist after install")
	}
}

func TestCopyDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	if err := os.MkdirAll(filepath.Join(src, "a", "b"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a", "file.txt"), []byte("file"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a", "b", "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify all files exist in dst
	for _, rel := range []string{"root.txt", "a/file.txt", "a/b/deep.txt"} {
		path := filepath.Join(dst, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("copied file %q not found at %s", rel, path)
		}
	}
}

func TestCopyDirSrcNotExist(t *testing.T) {
	err := copyDir("/nonexistent", "/tmp/dst")
	if err == nil {
		t.Fatal("expected error for nonexistent source, got nil")
	}
}

// Helper: createTestZip creates a zip file with the given content map.
// The map keys are file paths within the zip, values are file contents.
func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("cannot create zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatalf("cannot create zip entry %q: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("cannot write zip entry %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("cannot close zip writer: %v", err)
	}
}
