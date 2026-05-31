package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// getClashSecret loads the CLASH_API_SECRET from .env.
func getClashSecret() (string, error) {
	vars, err := internal.LoadEnv()
	if err != nil {
		return "", fmt.Errorf("无法加载 .env: %w", err)
	}
	secret := vars["CLASH_API_SECRET"]
	if secret == "" {
		return "", fmt.Errorf("缺少 CLASH_API_SECRET，无法访问 9090 控制接口。")
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
		fmt.Fprintf(w, "%s\t%s\n", s.Name, s.Current)
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
		fmt.Println(s.Name)
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
		fmt.Printf("=== %s ===\n", selector)
		nodes := internal.GetSelectorNodes(resp, selector)
		for _, n := range nodes {
			fmt.Printf("  %s\n", n)
		}
	} else {
		selectors := internal.GetSelectors(resp)
		for _, s := range selectors {
			fmt.Printf("=== %s （当前: %s）===\n", s.Name, s.Current)
			nodes := internal.GetSelectorNodes(resp, s.Name)
			for _, n := range nodes {
				fmt.Printf("  %s\n", n)
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
