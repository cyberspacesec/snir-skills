package runner

import "testing"

func TestURLMatchesTarget(t *testing.T) {
	cases := []struct {
		resp, target string
		want         bool
	}{
		{"https://example.com/", "https://example.com", true},     // 尾斜杠差异
		{"https://example.com", "https://example.com/", true},     // 反向尾斜杠
		{"https://example.com/", "https://example.com/", true},    // 完全一致
		{"https://example.com", "https://example.com", true},      // 都无尾斜杠
		{"http://example.com/", "https://example.com", false},     // 协议不同
		{"https://example.com/foo", "https://example.com", false}, // 路径不同
		{"https://example.com/foo", "https://example.com/foo", true},
		{"https://sub.example.com/", "https://example.com", false}, // host 不同
		{"https://example.com:443/", "https://example.com", false}, // 端口差异（host 不同）
	}
	for _, c := range cases {
		got := urlMatchesTarget(c.resp, c.target)
		if got != c.want {
			t.Errorf("urlMatchesTarget(%q,%q)=%v want %v", c.resp, c.target, got, c.want)
		}
	}
}

// TestURLMatchesTarget_ParseFallback 覆盖 respURL/target 均无法被 url.Parse 解析时
// 走后缀匹配兜底的分支（chromedp.go:96）。
func TestURLMatchesTarget_ParseFallback(t *testing.T) {
	// ":foo" 这类字符串 url.Parse 会返回错误，触发后缀匹配兜底
	if !urlMatchesTarget(":foo", ":foo") {
		t.Error("相同非法串后缀匹配应返回 true")
	}
	// 两个互不为后缀的非法串应返回 false
	if urlMatchesTarget(":foo", ":bar") {
		t.Error("互不后缀的非法串应返回 false")
	}
	// target 是 resp 的后缀（反向匹配）
	if !urlMatchesTarget("prefix:foo", ":foo") {
		t.Error("target 为 resp 后缀时应返回 true")
	}
}
