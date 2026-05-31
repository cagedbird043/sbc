package internal

import (
	"encoding/json"
	"testing"
)

var mockProxiesJSON = `{
  "proxies": {
    "🔰 节点选择": {
      "type": "Selector",
      "now": "🇭🇰 香港",
      "all": ["🇭🇰 香港", "🇯🇵 日本", "🇺🇸 美国"]
    },
    "🎛 手动选择 1": {
      "type": "Selector",
      "now": "🛫 老村长/香港 HK1",
      "all": ["🛫 老村长/香港 HK1", "🛫 老村长/日本 JP1", "🛫 老村长/新加坡 SG1"]
    },
    "📨 FCM": {
      "type": "Selector",
      "now": "🇨🇳 国内直连",
      "all": ["🇨🇳 国内直连", "🔰 节点选择", "🇯🇵 JP s4 Fast"]
    },
    "🇭🇰 香港": {
      "type": "Selector",
      "now": "🛫 老村长/香港 HK1",
      "all": ["🛫 老村长/香港 HK1", "🛫 老村长/香港 HK2"]
    },
    "DIRECT": {
      "type": "Direct",
      "now": "DIRECT",
      "all": null
    },
    "REJECT": {
      "type": "Reject",
      "now": "REJECT",
      "all": null
    }
  }
}`

func parseMockResponse(t *testing.T) *ClashProxiesResponse {
	t.Helper()
	resp, err := ParseProxiesResponseFromJSON(mockProxiesJSON)
	if err != nil {
		t.Fatalf("ParseProxiesResponseFromJSON failed: %v", err)
	}
	return resp
}

func TestGetSelectors(t *testing.T) {
	resp := parseMockResponse(t)
	selectors := GetSelectors(resp)

	if len(selectors) != 4 {
		t.Fatalf("expected 4 selectors, got %d", len(selectors))
	}

	// Verify each selector
	names := make(map[string]string)
	for _, s := range selectors {
		names[s.Name] = s.Current
	}

	tests := []struct {
		name    string
		current string
	}{
		{"🔰 节点选择", "🇭🇰 香港"},
		{"🎛 手动选择 1", "🛫 老村长/香港 HK1"},
		{"📨 FCM", "🇨🇳 国内直连"},
		{"🇭🇰 香港", "🛫 老村长/香港 HK1"},
	}
	for _, tc := range tests {
		cur, ok := names[tc.name]
		if !ok {
			t.Errorf("selector %q not found", tc.name)
			continue
		}
		if cur != tc.current {
			t.Errorf("selector %q current = %q, want %q", tc.name, cur, tc.current)
		}
	}

	// Ensure non-Selector types are excluded
	for name, info := range resp.Proxies {
		if info.Type != "Selector" {
			for _, s := range selectors {
				if s.Name == name {
					t.Errorf("non-Selector %q (%s) was included in GetSelectors", name, info.Type)
				}
			}
		}
	}
}

func TestGetSelectorsEmpty(t *testing.T) {
	resp := &ClashProxiesResponse{Proxies: map[string]ClashProxyInfo{}}
	selectors := GetSelectors(resp)
	if len(selectors) != 0 {
		t.Errorf("expected 0 selectors, got %d", len(selectors))
	}
}

func TestGetSelectorNodes(t *testing.T) {
	resp := parseMockResponse(t)

	tests := []struct {
		selector string
		expected []string
	}{
		{"🔰 节点选择", []string{"🇭🇰 香港", "🇯🇵 日本", "🇺🇸 美国"}},
		{"🎛 手动选择 1", []string{"🛫 老村长/香港 HK1", "🛫 老村长/日本 JP1", "🛫 老村长/新加坡 SG1"}},
		{"不存在", nil},
	}
	for _, tc := range tests {
		nodes := GetSelectorNodes(resp, tc.selector)
		if !stringSliceEqual(nodes, tc.expected) {
			t.Errorf("GetSelectorNodes(%q) = %v, want %v", tc.selector, nodes, tc.expected)
		}
	}
}

func TestGetSelectorNodesNonSelector(t *testing.T) {
	resp := parseMockResponse(t)
	// DIRECT is a Direct type, not Selector, so All should be nil
	nodes := GetSelectorNodes(resp, "DIRECT")
	if nodes != nil {
		t.Errorf("expected nil for non-Selector, got %v", nodes)
	}
}

func TestResolveSelectorExact(t *testing.T) {
	resp := parseMockResponse(t)

	result, err := ResolveSelector(resp, "🔰 节点选择")
	if err != nil {
		t.Fatalf("ResolveSelector failed: %v", err)
	}
	if result != "🔰 节点选择" {
		t.Errorf("expected '🔰 节点选择', got %q", result)
	}
}

func TestResolveSelectorSubstring(t *testing.T) {
	resp := parseMockResponse(t)

	tests := []struct {
		input    string
		expected string
	}{
		{"节点", "🔰 节点选择"},
		{"FCM", "📨 FCM"},
		{"香港", "🇭🇰 香港"},
		{"选择 1", "🎛 手动选择 1"},
	}
	for _, tc := range tests {
		result, err := ResolveSelector(resp, tc.input)
		if err != nil {
			t.Errorf("ResolveSelector(%q) failed: %v", tc.input, err)
			continue
		}
		if result != tc.expected {
			t.Errorf("ResolveSelector(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestResolveSelectorNotFound(t *testing.T) {
	resp := parseMockResponse(t)

	_, err := ResolveSelector(resp, "不存在的组")
	if err == nil {
		t.Fatal("expected error for non-existent selector, got nil")
	}
}

func TestResolveSelectorAmbiguous(t *testing.T) {
	resp := parseMockResponse(t)

	// "香港" is an exact match for "🇭🇰 香港" and also a substring of
	// nodes in other selectors, but it matches exactly one Selector name
	_, err := ResolveSelector(resp, "香港")
	if err != nil {
		t.Errorf("香港 should match exactly one selector: %v", err)
	}

	// "手动" matches "🎛 手动选择 1" only
	result, err := ResolveSelector(resp, "手动")
	if err != nil {
		t.Errorf("手动 should match: %v", err)
	}
	if result != "🎛 手动选择 1" {
		t.Errorf("expected '🎛 手动选择 1', got %q", result)
	}
}

func TestResolveSelectorCaseInsensitive(t *testing.T) {
	resp := parseMockResponse(t)

	result, err := ResolveSelector(resp, "fcm")
	if err != nil {
		t.Fatalf("ResolveSelector('fcm') failed: %v", err)
	}
	if result != "📨 FCM" {
		t.Errorf("expected '📨 FCM', got %q", result)
	}
}

func TestResolveNodeExact(t *testing.T) {
	resp := parseMockResponse(t)

	result, err := ResolveNode(resp, "🎛 手动选择 1", "🛫 老村长/香港 HK1")
	if err != nil {
		t.Fatalf("ResolveNode failed: %v", err)
	}
	if result != "🛫 老村长/香港 HK1" {
		t.Errorf("expected '🛫 老村长/香港 HK1', got %q", result)
	}
}

func TestResolveNodeSubstring(t *testing.T) {
	resp := parseMockResponse(t)

	tests := []struct {
		selector string
		input    string
		expected string
	}{
		{"🎛 手动选择 1", "香港 HK1", "🛫 老村长/香港 HK1"},
		{"🎛 手动选择 1", "JP1", "🛫 老村长/日本 JP1"},
		{"🔰 节点选择", "日本", "🇯🇵 日本"},
		{"🔰 节点选择", "美国", "🇺🇸 美国"},
	}
	for _, tc := range tests {
		result, err := ResolveNode(resp, tc.selector, tc.input)
		if err != nil {
			t.Errorf("ResolveNode(%q, %q) failed: %v", tc.selector, tc.input, err)
			continue
		}
		if result != tc.expected {
			t.Errorf("ResolveNode(%q, %q) = %q, want %q", tc.selector, tc.input, result, tc.expected)
		}
	}
}

func TestResolveNodeNotFound(t *testing.T) {
	resp := parseMockResponse(t)

	_, err := ResolveNode(resp, "🔰 节点选择", "不存在的节点")
	if err == nil {
		t.Fatal("expected error for non-existent node, got nil")
	}
}

func TestResolveNodeAmbiguous(t *testing.T) {
	// Create a response where a substring matches multiple nodes
	json := `{
	  "proxies": {
	    "test": {
	      "type": "Selector",
	      "now": "🇯🇵 日本",
	      "all": ["🇯🇵 日本/JP1", "🇯🇵 日本/JP2 住宅", "🇭🇰 香港"]
	    }
	  }
	}`
	resp, err := ParseProxiesResponseFromJSON(json)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ResolveNode(resp, "test", "日本")
	if err == nil {
		t.Fatal("expected error for ambiguous match, got nil")
	}
}

func TestResolveNodeCaseInsensitive(t *testing.T) {
	resp := parseMockResponse(t)

	result, err := ResolveNode(resp, "🎛 手动选择 1", "hk1")
	if err != nil {
		t.Fatalf("ResolveNode case-insensitive failed: %v", err)
	}
	if result != "🛫 老村长/香港 HK1" {
		t.Errorf("expected '🛫 老村长/香港 HK1', got %q", result)
	}
}

func TestSwitchProxyPayloadMarshal(t *testing.T) {
	payload := SwitchProxyPayload{Name: "🇭🇰 香港"}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	expected := `{"name":"🇭🇰 香港"}`
	if string(data) != expected {
		t.Errorf("json.Marshal = %q, want %q", string(data), expected)
	}
}

// Helper functions

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
