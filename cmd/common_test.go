package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ── Command registration tests ────────────────────────────────────────────

func TestRootCommandExists(t *testing.T) {
	if rootCmd.Name() != "sbc" {
		t.Fatalf("root command Name = %q, want %q", rootCmd.Name(), "sbc")
	}
}

func TestSubCommandsRegistered(t *testing.T) {
	expected := []string{
		"start", "stop", "restart", "status", "log", "config", "proxy", "ui", "update", "validate",
		"check", "completion",
	}

	seen := make(map[string]bool)
	for _, c := range rootCmd.Commands() {
		seen[c.Name()] = true
	}

	for _, name := range expected {
		if !seen[name] {
			t.Errorf("subcommand %q not registered on root", name)
		}
	}
}

func TestServiceCommandsRegistered(t *testing.T) {
	expected := []string{"start", "stop", "restart", "status", "log"}

	seen := make(map[string]bool)
	for _, c := range rootCmd.Commands() {
		seen[c.Name()] = true
	}

	for _, name := range expected {
		if !seen[name] {
			t.Errorf("service command %q not registered on root", name)
		}
	}
}

func TestConfigSubCommandsRegistered(t *testing.T) {
	expected := []string{"status", "show", "diff", "variant", "template", "env"}

	var configSub *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "config" {
			configSub = c
			break
		}
	}
	if configSub == nil {
		t.Fatal("config command not found")
	}

	seen := make(map[string]bool)
	for _, c := range configSub.Commands() {
		seen[c.Name()] = true
	}

	for _, name := range expected {
		if !seen[name] {
			t.Errorf("config subcommand %q not registered", name)
		}
	}
}

func TestConfigVariantSubCommandsRegistered(t *testing.T) {
	var configSub *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "config" {
			configSub = c
			break
		}
	}
	if configSub == nil {
		t.Fatal("config command not found")
	}

	var variantSub *cobra.Command
	for _, c := range configSub.Commands() {
		if c.Name() == "variant" {
			variantSub = c
			break
		}
	}
	if variantSub == nil {
		t.Fatal("variant command not found")
	}

	seen := make(map[string]bool)
	for _, c := range variantSub.Commands() {
		seen[c.Name()] = true
	}

	if !seen["set"] {
		t.Error("variant set subcommand not found")
	}
	if !seen["list"] {
		t.Error("variant list subcommand not found")
	}
}

func TestProxySubCommandsRegistered(t *testing.T) {
	expected := []string{"list", "groups", "nodes", "use"}

	var proxySub *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "proxy" {
			proxySub = c
			break
		}
	}
	if proxySub == nil {
		t.Fatal("proxy command not found")
	}

	seen := make(map[string]bool)
	for _, c := range proxySub.Commands() {
		seen[c.Name()] = true
	}

	for _, name := range expected {
		if !seen[name] {
			t.Errorf("proxy subcommand %q not registered", name)
		}
	}
}

func TestUISubCommandsRegistered(t *testing.T) {
	expected := []string{"status", "update"}

	var uiSub *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "ui" {
			uiSub = c
			break
		}
	}
	if uiSub == nil {
		t.Fatal("ui command not found")
	}

	seen := make(map[string]bool)
	for _, c := range uiSub.Commands() {
		seen[c.Name()] = true
	}

	for _, name := range expected {
		if !seen[name] {
			t.Errorf("ui subcommand %q not registered", name)
		}
	}
}

func TestCompletionSubCommandsRegistered(t *testing.T) {
	expected := []string{"zsh", "bash", "fish", "powershell"}

	var completionSub *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "completion" {
			completionSub = c
			break
		}
	}
	if completionSub == nil {
		t.Fatal("completion command not found")
	}

	seen := make(map[string]bool)
	for _, c := range completionSub.Commands() {
		seen[c.Name()] = true
	}

	for _, name := range expected {
		if !seen[name] {
			t.Errorf("completion subcommand %q not registered", name)
		}
	}
}

// ── Help output tests ─────────────────────────────────────────────────────

func TestRootHelpDoesNotPanic(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("root --help panicked: %v", r)
		}
	}()
	_ = rootCmd.Execute()
	output := buf.String()
	if !strings.Contains(output, "sbc") {
		t.Error("root --help output does not contain 'sbc'")
	}
}

func TestSubCommandHelpDoesNotPanic(t *testing.T) {
	subCommands := []string{
		"start", "stop", "restart", "status", "log", "config", "proxy", "ui", "update", "validate",
		"check", "completion",
	}

	for _, name := range subCommands {
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{name, "--help"})

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s --help panicked: %v", name, r)
				}
			}()
			_ = rootCmd.Execute()
			output := buf.String()
			if !strings.Contains(output, name) {
				t.Errorf("%s --help output does not contain %q", name, name)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--version"})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("--version panicked: %v", r)
		}
	}()
	_ = rootCmd.Execute()
}

func TestVersionVarSet(t *testing.T) {
	// Version should be set via ldflags at build time, defaults to "dev"
	if Version == "" {
		t.Error("Version variable is empty, should be 'dev' or ldflags value")
	}
}

func TestNoArgsDoesNotPanic(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("sbc with no args panicked: %v", r)
		}
	}()
	_ = rootCmd.Execute()
}

func TestCompletionZshGeneratesOutput(t *testing.T) {
	// GenZshCompletion writes to os.Stdout directly; swap stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		ioCopy(&buf, r)
		done <- buf.String()
	}()

	rootCmd.SetArgs([]string{"completion", "zsh"})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("completion zsh panicked: %v", r)
		}
	}()
	_ = rootCmd.Execute()

	w.Close()
	os.Stdout = old
	output := <-done

	if len(output) == 0 {
		t.Error("completion zsh produced no output")
	}
}

// ioCopy wraps io.Copy to avoid import conflict with net/http.
func ioCopy(dst *bytes.Buffer, src *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}
