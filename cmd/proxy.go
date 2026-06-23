package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cagedbird043/sbc/internal"

	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy {list|groups|nodes|use}",
	Short: "代理控制",
	Long:  "通过 sing-box Clash API (localhost:9090) 动态管理代理节点。",
}

var proxyListCmd = &cobra.Command{
	Use:   "list",
	Short: "当前 Selector 状态",
	Run: func(cmd *cobra.Command, args []string) {
		proxyList()
	},
}

var proxyGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "组代号列表",
	Run: func(cmd *cobra.Command, args []string) {
		proxyGroups()
	},
}

var proxyNodesCmd = &cobra.Command{
	Use:   "nodes [组]",
	Short: "节点代号列表（可选指定组）",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		names, err := completeSelectorNames()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		filter := ""
		if len(args) > 0 {
			filter = args[0]
		}
		proxyNodes(filter)
	},
}

var proxyUseCmd = &cobra.Command{
	Use:   "use <组> <节点>",
	Short: "切换节点（支持精确名和子串匹配）",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete group name
			names, err := completeSelectorNames()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		// len(args) == 1: complete node name within the group
		nodes, err := completeSelectorNodes(args[0])
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return nodes, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		proxyUse(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(proxyCmd)
	proxyCmd.AddCommand(proxyListCmd)
	proxyCmd.AddCommand(proxyGroupsCmd)
	proxyCmd.AddCommand(proxyNodesCmd)
	proxyCmd.AddCommand(proxyUseCmd)
}

// clashAPIRequest performs an HTTP request to the Clash API.
func clashAPIRequest(method, path, secret string, body io.Reader) (*http.Response, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(method, "http://127.0.0.1:9090"+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return client.Do(req)
}

// getClashSecret loads the clash_api_secret from sbc.toml.
func getClashSecret() (string, error) {
	vars, err := internal.LoadEnv()
	if err != nil {
		return "", fmt.Errorf("无法加载配置: %w", err)
	}
	rawSecret, ok := vars["clash_api_secret"]
	if !ok || rawSecret == nil {
		rawSecret, ok = vars["CLASH_API_SECRET"]
	}
	if !ok || rawSecret == nil {
		return "", fmt.Errorf("缺少 clash_api_secret，无法访问 9090 控制接口。")
	}
	secret, ok := rawSecret.(string)
	if !ok || secret == "" {
		return "", fmt.Errorf("clash_api_secret 不是有效的字符串")
	}
	return secret, nil
}

// fetchProxies fetches and parses the /proxies endpoint.
func fetchProxies() (*internal.ClashProxiesResponse, error) {
	secret, err := getClashSecret()
	if err != nil {
		return nil, err
	}

	resp, err := clashAPIRequest("GET", "/proxies", secret, nil)
	if err != nil {
		return nil, fmt.Errorf("无法连接 Clash API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Clash API 返回状态码 %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 API 响应失败: %w", err)
	}

	var proxiesResp internal.ClashProxiesResponse
	if err := json.Unmarshal(data, &proxiesResp); err != nil {
		return nil, fmt.Errorf("解析 Clash API 响应失败: %w", err)
	}

	return &proxiesResp, nil
}

func proxyList() {
	resp, err := fetchProxies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	selectors := internal.GetSelectors(resp)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, s := range selectors {
		fmt.Fprintf(w, "%s\t%s\n", internal.StripEmoji(s.Name), internal.StripEmoji(s.Current))
	}
	w.Flush()
}

func proxyGroups() {
	resp, err := fetchProxies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	selectors := internal.GetSelectors(resp)
	for _, s := range selectors {
		fmt.Println(internal.StripEmoji(s.Name))
	}
}

func proxyNodes(filter string) {
	resp, err := fetchProxies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	if filter != "" {
		// Resolve selector
		selector, err := internal.ResolveSelector(resp, filter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("=== %s ===\n", internal.StripEmoji(selector))
		nodes := internal.GetSelectorNodes(resp, selector)
		for _, n := range nodes {
			fmt.Printf("  %s\n", internal.StripEmoji(n))
		}
	} else {
		selectors := internal.GetSelectors(resp)
		for _, s := range selectors {
			fmt.Printf("=== %s （当前: %s）===\n", internal.StripEmoji(s.Name), internal.StripEmoji(s.Current))
			nodes := internal.GetSelectorNodes(resp, s.Name)
			for _, n := range nodes {
				fmt.Printf("  %s\n", internal.StripEmoji(n))
			}
			fmt.Println()
		}
	}
}

func proxyUse(selectorInput, nodeInput string) {
	if selectorInput == "" || nodeInput == "" {
		fmt.Fprintf(os.Stderr, "❌ 用法: sbc proxy use <组> <节点>\n")
		fmt.Fprintf(os.Stderr, "  组和节点都支持精确名或子串匹配。\n")
		os.Exit(1)
	}

	resp, err := fetchProxies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// Resolve selector
	selector, err := internal.ResolveSelector(resp, selectorInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		fmt.Fprintf(os.Stderr, "可用组:\n")
		for _, s := range internal.GetSelectors(resp) {
			fmt.Fprintf(os.Stderr, "  - %s\n", s.Name)
		}
		os.Exit(1)
	}

	// Get current node
	current := ""
	if info, ok := resp.Proxies[selector]; ok {
		current = info.Now
	}

	// Resolve target node
	target, err := internal.ResolveNode(resp, selector, nodeInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		fmt.Fprintf(os.Stderr, "可用节点:\n")
		nodes := internal.GetSelectorNodes(resp, selector)
		for _, n := range nodes {
			fmt.Fprintf(os.Stderr, "  - %s\n", n)
		}
		os.Exit(1)
	}

	// Execute switch
	secret, err := getClashSecret()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	payload, _ := json.Marshal(internal.SwitchProxyPayload{Name: target})
	path := selectorPath(selector)
	resp2, err := clashAPIRequest("PUT", path, secret, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 切换节点失败: %v\n", err)
		os.Exit(1)
	}
	defer resp2.Body.Close()

	fmt.Printf("✅ [%s]：%s → %s\n", selector, current, target)
}

// selectorPath returns URL-encoded path for a proxy selector.
func selectorPath(name string) string {
	return "/proxies/" + url.PathEscape(name)
}

// completeSelectorNames returns all Selector names for shell completion
// with emoji stripped for cleaner display. The selected value still
// matches the original name via substring matching in proxyUse.
func completeSelectorNames() ([]string, error) {
	resp, err := fetchProxies()
	if err != nil {
		return nil, err
	}
	selectors := internal.GetSelectors(resp)
	names := make([]string, len(selectors))
	for i, s := range selectors {
		names[i] = internal.StripEmoji(s.Name)
	}
	return names, nil
}

// completeSelectorNodes returns all node names for a given selector (for shell completion)
// with emoji stripped for cleaner display.
func completeSelectorNodes(selectorInput string) ([]string, error) {
	resp, err := fetchProxies()
	if err != nil {
		return nil, err
	}
	selector, err := internal.ResolveSelector(resp, selectorInput)
	if err != nil {
		return nil, err
	}
	nodes := internal.GetSelectorNodes(resp, selector)
	cleaned := make([]string, len(nodes))
	for i, n := range nodes {
		cleaned[i] = internal.StripEmoji(n)
	}
	return cleaned, nil
}
