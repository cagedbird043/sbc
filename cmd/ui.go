package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui {status|update}",
	Short: "面板管理",
	Long:  "管理 sing-box 面板（zashboard）的状态与更新。",
}

var uiStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "面板状态",
	Run: func(cmd *cobra.Command, args []string) {
		uiStatus()
	},
}

var uiUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新面板",
	Run: func(cmd *cobra.Command, args []string) {
		uiUpdate()
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
	uiCmd.AddCommand(uiStatusCmd)
	uiCmd.AddCommand(uiUpdateCmd)
}

func uiStatus() {
	uiName, err := internal.ConfigUIName()
	if err != nil {
		uiName = "zashboard"
	}
	url, err := internal.ConfigUIDownloadURL()
	if err != nil {
		url = internal.DefaultUIDownloadURL
	}
	dest := internal.UIDestDir(uiName)

	fmt.Printf("ui=%s\nurl=%s\ndest=%s\n", uiName, url, dest)

	// List asset files
	assetsDir := filepath.Join(dest, "assets")
	if entries, err := os.ReadDir(assetsDir); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if (strings.HasPrefix(name, "index-") && strings.HasSuffix(name, ".js")) ||
				(strings.HasPrefix(name, "index-") && strings.HasSuffix(name, ".css")) {
				fmt.Println(name)
			}
		}
	} else {
		fmt.Println("missing")
	}
}

func uiUpdate() {
	uiName, err := internal.ConfigUIName()
	if err != nil {
		uiName = "zashboard"
	}
	url, err := internal.ConfigUIDownloadURL()
	if err != nil {
		url = internal.DefaultUIDownloadURL
	}
	dest := internal.UIDestDir(uiName)

	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "sbc-ui.*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "ui.zip")
	extractedDir := filepath.Join(tmpDir, "extracted")

	fmt.Printf("📦 正在下载 zashboard UI: %s\n", url)

	// Download
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 下载失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "❌ 下载失败，HTTP 状态码: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	archiveFile, err := os.Create(archivePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 创建临时文件失败: %v\n", err)
		os.Exit(1)
	}
	if _, err := io.Copy(archiveFile, resp.Body); err != nil {
		archiveFile.Close()
		fmt.Fprintf(os.Stderr, "❌ 下载写入失败: %v\n", err)
		os.Exit(1)
	}
	archiveFile.Close()

	// Extract
	if err := internal.ExtractUIZip(archivePath, extractedDir); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 解压失败: %v\n", err)
		os.Exit(1)
	}

	// Validate
	if _, err := os.Stat(filepath.Join(extractedDir, "index.html")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ UI 压缩包内容异常：缺少 index.html\n")
		os.Exit(1)
	}

	// Install
	if err := internal.InstallUIDir(extractedDir, dest); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 安装 UI 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ UI 已更新：%s\n", dest)
	uiStatus()
}
