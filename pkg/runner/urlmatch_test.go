package runner

import "testing"

func TestURLMatchesTarget(t *testing.T) {
	cases := []struct{ resp, target string; want bool }{
		{"https://example.com/", "https://example.com", true},      // 尾斜杠差异
		{"https://example.com", "https://example.com/", true},      // 反向尾斜杠
		{"https://example.com/", "https://example.com/", true},     // 完全一致
		{"https://example.com", "https://example.com", true},       // 都无尾斜杠
		{"http://example.com/", "https://example.com", false},      // 协议不同
		{"https://example.com/foo", "https://example.com", false},  // 路径不同
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
