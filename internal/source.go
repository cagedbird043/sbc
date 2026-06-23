package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// metaInfo represents the structure of a .meta.json file.
type metaInfo struct {
	SHA256 string `json:"sha256"`
}


// DownloadConfigs downloads all config URLs to a temporary directory.
// For each .json URL, it also downloads the corresponding .meta.json and
// verifies the SHA256 checksum.  Failed downloads do not interrupt the
// remaining URLs.
// Returns:
//   - downloaded: map of filename → local temp file path
//   - failed: list of URLs that could not be downloaded/verified
func DownloadConfigs(urls []string) (downloaded map[string]string, failed []string, err error) {
	tmpDir, err := os.MkdirTemp("", "sbc-download.*")
	if err != nil {
		return nil, nil, fmt.Errorf("创建临时下载目录失败: %w", err)
	}

	downloaded = make(map[string]string)
	client := &http.Client{Timeout: 30 * time.Second}

	for _, url := range urls {
		filename := filepath.Base(url)
		if filename == "" || filename == "." || filename == "/" {
			failed = append(failed, url)
			continue
		}

		destPath := filepath.Join(tmpDir, filename)

		// Download the config file itself
		if dlErr := downloadFile(client, url, destPath); dlErr != nil {
			fmt.Fprintf(os.Stderr, "⚠ 下载失败: %v\n", dlErr)
			failed = append(failed, url)
			continue
		}

		// For .json config files, also download the companion .meta.json
		// and verify the SHA256 checksum.
		if strings.HasSuffix(url, ".json") && !strings.HasSuffix(url, ".meta.json") {
			metaURL := url[:len(url)-5] + ".meta.json"
			metaFilename := filename[:len(filename)-5] + ".meta.json"
			metaPath := filepath.Join(tmpDir, metaFilename)

			if vErr := downloadAndVerify(client, destPath, metaURL, metaPath); vErr != nil {
				fmt.Fprintf(os.Stderr, "⚠ 校验失败 [%s]: %v\n", filename, vErr)
				os.Remove(destPath)
				os.Remove(metaPath)
				failed = append(failed, url)
				continue
			}
			// Meta file served its purpose — clean up
			os.Remove(metaPath)
		}

		downloaded[filename] = destPath
	}

	return downloaded, failed, nil
}

// downloadFile fetches url and writes it to dest.
func downloadFile(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载 %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("创建文件 %s: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return fmt.Errorf("写入文件 %s: %w", dest, err)
	}
	return nil
}

// downloadAndVerify downloads the meta file and verifies that the SHA256
// of the already-downloaded config matches the value recorded in the meta file.
func downloadAndVerify(client *http.Client, configPath, metaURL, metaPath string) error {
	if err := downloadFile(client, metaURL, metaPath); err != nil {
		return fmt.Errorf("下载 meta 失败: %w", err)
	}

	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("读取 meta 文件失败: %w", err)
	}

	var meta metaInfo
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return fmt.Errorf("解析 meta 文件失败: %w", err)
	}
	if meta.SHA256 == "" {
		return fmt.Errorf("meta 文件中缺少 sha256 字段")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	hash := sha256.Sum256(configData)
	computed := hex.EncodeToString(hash[:])

	if !strings.EqualFold(computed, meta.SHA256) {
		return fmt.Errorf("SHA256 校验失败: 期望 %s, 计算得 %s", meta.SHA256[:16]+"...", computed[:16]+"...")
	}

	return nil
}
