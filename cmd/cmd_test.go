package cmd

import (
	"net"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestProxyStrategyFlag_Set(t *testing.T) {
	tests := []struct {
		input string
		want  runner.ProxyStrategy
	}{
		{"round-robin", runner.ProxyRoundRobin},
		{"random", runner.ProxyRandom},
		{"sequential", runner.ProxySequential},
		{"", runner.ProxyRoundRobin},
		{"unknown", runner.ProxyRoundRobin},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s runner.ProxyStrategy
			f := &proxyStrategyFlag{value: &s}
			if err := f.Set(tt.input); err != nil {
				t.Fatalf("Set(%q) 错误: %v", tt.input, err)
			}
			if s != tt.want {
				t.Fatalf("Set(%q) => %s, want %s", tt.input, s, tt.want)
			}
		})
	}
}

func TestProxyStrategyFlag_StringAndType(t *testing.T) {
	s := runner.ProxyRandom
	f := &proxyStrategyFlag{value: &s}
	if f.String() != "random" {
		t.Fatalf("String() = %q, want random", f.String())
	}
	if f.Type() != "string" {
		t.Fatalf("Type() = %q, want string", f.Type())
	}
	var nilF proxyStrategyFlag
	if nilF.String() != "" {
		t.Fatalf("nil value String() 应为空串, got %q", nilF.String())
	}
}

func TestInc_IPIncrement(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"末字节+1", "192.168.1.1", "192.168.1.2"},
		{"末字节进位", "192.168.1.255", "192.168.2.0"},
		{"全进位", "10.0.0.255", "10.0.1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.in).To4()
			if ip == nil {
				t.Fatalf("非法 IP: %s", tt.in)
			}
			inc(ip)
			if got := ip.String(); got != tt.want {
				t.Fatalf("inc(%s) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestGenerateRandomAPIKey(t *testing.T) {
	key := generateRandomAPIKey(16)
	if len(key) != 16 {
		t.Fatalf("长度 16 请求应返回 16 字符 hex, got %d", len(key))
	}
	key2 := generateRandomAPIKey(16)
	if key == key2 {
		t.Fatal("两次随机密钥不应相同")
	}
}

func TestPrintResult_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("printResult panic: %v", r)
		}
	}()
	printResult("any")
}
