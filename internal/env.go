package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// requiredEnvVars lists the env vars that must be present in .env.
var requiredEnvVars = []string{
	"CLASH_API_SECRET",
	"MIXED_PROXY_USERNAME",
	"MIXED_PROXY_PASSWORD",
	"PROVIDER_NAME_1",
	"SUB_URL_1",
}

// ReadEnvFile reads a key=value .env file and returns a map.
// Lines starting with # are ignored as comments; empty lines skipped.
// Values may be quoted with double quotes.
func ReadEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取 .env 文件 %s: %w", path, err)
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split on first = only
		idx := strings.IndexByte(line, '=')
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding double quotes if present
		val = strings.Trim(val, "\"")
		vars[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 .env 文件出错: %w", err)
	}
	return vars, nil
}

// RequireEnvFile checks that the .env file exists.
func RequireEnvFile() error {
	envPath, err := EnvFilePath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf("未发现凭证！请将 .env 放置在 %s", envPath)
	}
	return nil
}

// LoadEnv reads .env and returns the map. Fatal if missing.
func LoadEnv() (map[string]string, error) {
	if err := RequireEnvFile(); err != nil {
		return nil, err
	}
	envPath, _ := EnvFilePath()
	return ReadEnvFile(envPath)
}

// RequireEnvVars checks that required env variables are present.
// Returns a list of missing keys.
func RequireEnvVars(vars map[string]string) []string {
	var missing []string
	for _, key := range requiredEnvVars {
		if vars[key] == "" {
			missing = append(missing, key)
		}
	}
	return missing
}
