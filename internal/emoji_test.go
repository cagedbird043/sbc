package internal

import (
	"testing"
)

func TestStripEmoji_Common(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Selector names
		{"节点选择", "🔰 节点选择", "节点选择"},
		{"香港", "🇭🇰 香港", "香港"},
		{"日本", "🇯🇵 日本", "日本"},
		{"新加坡", "🇸🇬 新加坡", "新加坡"},
		{"FCM", "📨 FCM", "FCM"},
		{"Apple", "🍎 Apple", "Apple"},
		{"Telegram", "📲 Telegram", "Telegram"},
		{"Google", "🔎 Google", "Google"},
		{"AI", "🤖 AI", "AI"},
		{"Steam", "🎮 Steam", "Steam"},
		{"dlsite", "🎮 dlsite", "dlsite"},
		{"18comic", "🔞 18comic", "18comic"},
		{"手动选择", "🎛 手动选择 1", "手动选择 1"},

		// Node names
		{"老村长节点", "🛫 老村长/香港 HK1", "老村长/香港 HK1"},
		{"老村长流量", "🛫 老村长/剩余流量：381.16 GB", "老村长/剩余流量：381.16 GB"},
		{"JMS节点", "🛫 JustMySocks/c62s1.com:11819", "JustMySocks/c62s1.com:11819"},

		// No emoji — untouched
		{"国内直连", "国内直连", "国内直连"},
		{"plain", "hello world", "hello world"},
		{"Bulk", "Bulk s801", "Bulk s801"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripEmoji(tc.input); got != tc.want {
				t.Errorf("StripEmoji(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestStripEmoji_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"空字符串", "", ""},
		{"只有表情", "🔰🇭🇰", ""},
		{"两端空格后清", "  🔰 节点选择  ", "节点选择"},
		{"表情在中间", "香港🔰节点", "香港节点"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripEmoji(tc.input); got != tc.want {
				t.Errorf("StripEmoji(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
