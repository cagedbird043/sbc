package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Platform / path constants and detection.
// Mirrors the logic in the original sbc-lib.sh.

const (
	DefaultServiceNameLinux     = "sing-box"
	DefaultServiceLabelMacOS    = "homebrew.mxcl.sing-box-cagedbird"
	DefaultTargetConfLinux      = "/etc/sing-box/config.json"
	DefaultTargetConfMacOS      = "/opt/homebrew/etc/sing-box/config.json"
	DefaultUIBaseDirLinux       = "/var/lib/sing-box"
	DefaultUIBaseDirMacOS       = "/opt/homebrew/var/lib/sing-box"
	DefaultUIDownloadURL        = "https://github.com/cagedbird043/zashboard/archive/refs/heads/gh-pages.zip"
	DefaultUIOwnerLinux         = "sing-box:sing-box"
	DefaultConfigVariant        = "default"
	DefaultConfigVariantRealIP  = "realip-v4-only"
)

// Platform returns the current OS platform string: "linux" or "macos".
func Platform() string {
	switch runtime.GOOS {
	case "linux":
		return "linux"
	case "darwin":
		return "macos"
	default:
		// fallback: return raw GOOS, caller should handle
		return runtime.GOOS
	}
}

// ConfDir returns $HOME/.config/sing-box.
func ConfDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户 home 目录: %w", err)
	}
	return filepath.Join(home, ".config", "sing-box"), nil
}

// EnvFilePath returns $HOME/.config/sing-box/.env.
func EnvFilePath() (string, error) {
	confDir, err := ConfDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(confDir, ".env"), nil
}

// ConfigFilePath returns $HOME/.config/sing-box/sbc.toml.
func ConfigFilePath() (string, error) {
	confDir, err := ConfDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(confDir, "sbc.toml"), nil
}

// VariantStateFile returns $HOME/.config/sing-box/config-variant.
func VariantStateFile() (string, error) {
	confDir, err := ConfDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(confDir, "config-variant"), nil
}

// Profile returns the config profile: $SBC_PROFILE or defaults to Platform().
func Profile() string {
	if p := os.Getenv("SBC_PROFILE"); p != "" {
		return p
	}
	return Platform()
}

// TargetConf returns the deployed config.json path for the current platform.
// Can be overridden by the TARGET_CONF environment variable.
func TargetConf() string {
	if tc := os.Getenv("TARGET_CONF"); tc != "" {
		return tc
	}
	profile := Profile()
	switch profile {
	case "macos":
		return DefaultTargetConfMacOS
	default:
		return DefaultTargetConfLinux
	}
}

// UIBaseDir returns the base directory for UI files.
func UIBaseDir() string {
	profile := Profile()
	switch profile {
	case "macos":
		return DefaultUIBaseDirMacOS
	default:
		return DefaultUIBaseDirLinux
	}
}

// UIOwner returns the owner for UI files (Linux only).
func UIOwner() string {
	profile := Profile()
	switch profile {
	case "macos":
		return ""
	default:
		return DefaultUIOwnerLinux
	}
}

// ServiceLabelMacOS returns the launchctl service label for macOS.
func ServiceLabelMacOS() string {
	return DefaultServiceLabelMacOS
}

// ServiceNameLinux returns the systemd service name for Linux.
func ServiceNameLinux() string {
	return DefaultServiceNameLinux
}
