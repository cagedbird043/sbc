package internal

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ClashProxyInfo represents the proxy information returned by /proxies.
type ClashProxyInfo struct {
	Type string   `json:"type"`
	Now  string   `json:"now"`
	All  []string `json:"all"`
}

// ClashProxiesResponse represents the top-level response from GET /proxies.
type ClashProxiesResponse struct {
	Proxies map[string]ClashProxyInfo `json:"proxies"`
}

// ProxySelector represents a Selector-type proxy with its current node.
type ProxySelector struct {
	Name    string
	Current string
}

// ParseProxiesResponseFromJSON parses a JSON string into a ClashProxiesResponse.
// This is used by tests.
func ParseProxiesResponseFromJSON(jsonStr string) (*ClashProxiesResponse, error) {
	var resp ClashProxiesResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("解析 Clash API 响应失败: %w", err)
	}
	return &resp, nil
}

// GetSelectors extracts all ProxySelector entries from the response.
func GetSelectors(resp *ClashProxiesResponse) []ProxySelector {
	var selectors []ProxySelector
	for name, info := range resp.Proxies {
		if info.Type == "Selector" {
			selectors = append(selectors, ProxySelector{
				Name:    name,
				Current: info.Now,
			})
		}
	}
	return selectors
}

// GetSelectorNodes returns the available nodes for a given selector.
func GetSelectorNodes(resp *ClashProxiesResponse, selectorName string) []string {
	if info, ok := resp.Proxies[selectorName]; ok {
		return info.All
	}
	return nil
}

// SwitchProxyPayload is the JSON payload for PUT /proxies/{name}.
type SwitchProxyPayload struct {
	Name string `json:"name"`
}

// ResolveSelector resolves a selector name using exact match first, then substring (case-insensitive).
// Returns the resolved name, or an error with suggestions if ambiguous.
func ResolveSelector(resp *ClashProxiesResponse, input string) (string, error) {
	// 1. Exact match
	for name := range resp.Proxies {
		if resp.Proxies[name].Type == "Selector" && name == input {
			return name, nil
		}
	}

	// 2. Case-insensitive substring match
	var matches []string
	lowerInput := strings.ToLower(input)
	for name := range resp.Proxies {
		if resp.Proxies[name].Type == "Selector" && strings.Contains(strings.ToLower(name), lowerInput) {
			matches = append(matches, name)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("未找到组: %s", input)
	case 1:
		return matches[0], nil
	default:
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("组 \"%s\" 匹配到多个 Selector：", input))
		for _, m := range matches {
			sb.WriteString("\n  - " + m)
		}
		return "", fmt.Errorf("%s", sb.String())
	}
}

// ResolveNode resolves a node name within a selector using exact match first, then substring (case-insensitive).
func ResolveNode(resp *ClashProxiesResponse, selector, input string) (string, error) {
	nodes := GetSelectorNodes(resp, selector)

	// 1. Exact match
	for _, node := range nodes {
		if node == input {
			return node, nil
		}
	}

	// 2. Case-insensitive substring match
	var matches []string
	lowerInput := strings.ToLower(input)
	for _, node := range nodes {
		if strings.Contains(strings.ToLower(node), lowerInput) {
			matches = append(matches, node)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("在 [%s] 中未找到节点: %s", selector, input)
	case 1:
		return matches[0], nil
	default:
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("节点 \"%s\" 在 [%s] 中匹配到多个：", input, selector))
		for _, m := range matches {
			sb.WriteString("\n  - " + m)
		}
		return "", fmt.Errorf("%s", sb.String())
	}
}
