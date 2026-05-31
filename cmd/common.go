package cmd

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cagedbird043/sbc/internal"
)

// activeVariant returns the current config variant (for overview display).
func activeVariant() string {
	v, err := internal.ActiveConfigVariant()
	if err != nil {
		return "default"
	}
	return v
}

// variantDescription returns a description string for the given variant.
func variantDescription(v string) string {
	return internal.VariantDescription(v)
}

// readEnvFile reads the .env file and returns the key-value map.
func readEnvFile(path string) (map[string]string, error) {
	return internal.ReadEnvFile(path)
}

// apiGet performs a GET request to the sing-box Clash API.
func apiGet(path string) (string, error) {
	envFile, err := internal.EnvFilePath()
	if err != nil {
		return "", err
	}
	vars, err := internal.ReadEnvFile(envFile)
	if err != nil {
		return "", err
	}
	secret := vars["CLASH_API_SECRET"]
	if secret == "" {
		return "", fmt.Errorf("缺少 CLASH_API_SECRET")
	}

	return apiGetWithSecret(path, secret)
}

// apiGetWithSecret performs a GET request to the Clash API with the given secret.
func apiGetWithSecret(path, secret string) (string, error) {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequest("GET", "http://127.0.0.1:9090"+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// apiPut performs a PUT request to the sing-box Clash API.
func apiPut(path, secret string, body io.Reader) error {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequest("PUT", "http://127.0.0.1:9090"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
