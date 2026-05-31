package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VerifyDownloads checks that the downloaded config set is usable.
//
// Rules (in priority order):
//  1. If the active variant's template was not downloaded, that is a fatal
//     error.  A helpful hint is printed if a local cache copy exists.
//  2. If every URL failed and a local cache exists, the error message
//     suggests using the cache.
//  3. If every URL failed and no cache exists at all, return a fatal error.
//  4. Partial failures are printed as warnings; the caller may proceed with
//     the successfully downloaded templates.
func VerifyDownloads(urls []string, downloaded map[string]string, failed []string) error {
	configDir, _ := ConfDir()

	// --- Rule 1: active variant's template must be present ----------
	variant, _ := ActiveConfigVariant()
	expectedFile := "config-" + variant + ".json"

	if _, ok := downloaded[expectedFile]; !ok {
		// Check for local cache fallback
		cacheFile := filepath.Join(configDir, expectedFile)
		if _, err := os.Stat(cacheFile); err == nil {
			return fmt.Errorf("当前变体 '%s' 的模板下载失败，但本地缓存可用。请检查网络连接。\n失败 URL:\n  %s", variant, strings.Join(failed, "\n  "))
		}
		return fmt.Errorf("当前变体 '%s' 的模板下载失败且本地无缓存。\n请确认 SBC_CONFIG_URLS 包含该变体的 JSON 文件。", variant)
	}

	// --- Rule 2 & 3: all-URLs-failed cases --------------------------
	if len(downloaded) == 0 {
		if configDir != "" {
			entries, _ := os.ReadDir(configDir)
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, "config-") && strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".meta.json") {
					return fmt.Errorf("所有配置下载失败，但本地缓存 (%s) 仍可用。请检查网络连接。\n失败列表:\n  %s", configDir, strings.Join(failed, "\n  "))
				}
			}
		}
		return fmt.Errorf("所有配置下载失败且本地无缓存。\n失败列表:\n  %s", strings.Join(failed, "\n  "))
	}

	// --- Rule 4: partial failure → warn, but not fatal ---------------
	if len(failed) > 0 {
		fmt.Fprintf(os.Stderr, "⚠ 以下配置下载失败（将继续使用已下载的配置）:\n")
		for _, f := range failed {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
	}

	return nil
}
