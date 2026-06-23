package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// requiredEnvVars lists the env vars that must be present.
var requiredEnvVars = []string{
	"CLASH_API_SECRET",
	"MIXED_PROXY_USERNAME",
	"MIXED_PROXY_PASSWORD",
	"PROVIDER_NAME_1",
	"SUB_URL_1",
}

// ReadEnvFile reads a key=value .env file or sbc.toml file and returns a map.
func ReadEnvFile(path string) (map[string]string, error) {
	// If path is .env and doesn't exist, try to fall back to sbc.toml
	if strings.HasSuffix(path, ".env") {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if tomlPath, err := ConfigFilePath(); err == nil {
				if _, err := os.Stat(tomlPath); err == nil {
					path = tomlPath
				}
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取环境配置文件 %s: %w", path, err)
	}

	if strings.HasSuffix(path, ".toml") {
		var config map[string]interface{}
		if err := toml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析 TOML 失败: %w", err)
		}
		vars := make(map[string]string)
		for k, v := range config {
			if slice, ok := v.([]interface{}); ok {
				var parts []string
				for _, item := range slice {
					parts = append(parts, fmt.Sprintf("%v", item))
				}
				vars[strings.ToUpper(k)] = `"` + strings.Join(parts, `","`) + `"`
			} else {
				vars[strings.ToUpper(k)] = fmt.Sprintf("%v", v)
			}
		}
		return vars, nil
	}

	vars := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, "\"")
		if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") && len(val) >= 2 {
			val = val[1 : len(val)-1]
		}
		vars[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 env 文件出错: %w", err)
	}
	return vars, nil
}

// RequireEnvFile checks that the sbc.toml file exists.
func RequireEnvFile() error {
	configPath, err := ConfigFilePath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("未发现配置！请将 sbc.toml 放置在 %s", configPath)
	}
	return nil
}

// LoadEnv reads sbc.toml and returns a map.
func LoadEnv() (map[string]interface{}, error) {
	configPath, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("未发现配置！请将 sbc.toml 放置在 %s", configPath)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取 sbc.toml 文件: %w", err)
	}
	var config map[string]interface{}
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析 sbc.toml 失败: %w", err)
	}
	return config, nil
}

// ReadEnvURLs reads sbc_config_urls from sbc.toml.
func ReadEnvURLs() ([]string, error) {
	config, err := LoadEnv()
	if err != nil {
		return nil, err
	}
	raw, ok := config["sbc_config_urls"]
	if !ok {
		return nil, fmt.Errorf("sbc.toml 中缺少 sbc_config_urls")
	}
	
	switch v := raw.(type) {
	case []interface{}:
		var urls []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				urls = append(urls, str)
			}
		}
		return urls, nil
	case string:
		return []string{v}, nil
	default:
		return nil, fmt.Errorf("sbc_config_urls 格式不正确")
	}
}

// RequireEnvVars checks that required env variables are present.
// Returns a list of missing keys.
func RequireEnvVars(vars map[string]interface{}) []string {
	var missing []string
	for _, key := range requiredEnvVars {
		lowerKey := strings.ToLower(key)
		val, ok := vars[lowerKey]
		if !ok || val == nil {
			val, ok = vars[key]
		}
		if !ok || val == nil {
			missing = append(missing, lowerKey)
			continue
		}
		if str, ok := val.(string); ok && str == "" {
			missing = append(missing, lowerKey)
		}
	}
	return missing
}
