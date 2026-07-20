package models

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestHeaderMap(t *testing.T) {
	tests := []struct {
		name    string
		headers []Header
		want    map[string][]string
	}{
		{
			name: "单个头信息测试",
			headers: []Header{
				{Name: "Content-Type", Value: "application/json"},
			},
			want: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "多个不同头信息测试",
			headers: []Header{
				{Name: "Content-Type", Value: "application/json"},
				{Name: "Authorization", Value: "Bearer token123"},
			},
			want: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer token123"},
			},
		},
		{
			name: "相同名称多值头信息测试",
			headers: []Header{
				{Name: "Set-Cookie", Value: "sessionId=123"},
				{Name: "Set-Cookie", Value: "userId=456"},
			},
			want: map[string][]string{
				"Set-Cookie": {"sessionId=123", "userId=456"},
			},
		},
		{
			name:    "空头信息测试",
			headers: []Header{},
			want:    map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				Headers: tt.headers,
			}
			got := r.HeaderMap()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HeaderMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResultEnrichEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		scheme   string
		host     string
		port     int
		endpoint string
	}{
		{
			name:     "https default port",
			result:   Result{URL: "https://example.com/path"},
			scheme:   "https",
			host:     "example.com",
			port:     443,
			endpoint: "https://example.com:443",
		},
		{
			name:     "http explicit port",
			result:   Result{URL: "http://example.com:8080/admin"},
			scheme:   "http",
			host:     "example.com",
			port:     8080,
			endpoint: "http://example.com:8080",
		},
		{
			name:     "final url fallback",
			result:   Result{FinalURL: "https://www.example.org/"},
			scheme:   "https",
			host:     "www.example.org",
			port:     443,
			endpoint: "https://www.example.org:443",
		},
		{
			name:     "bare host fallback",
			result:   Result{URL: "example.net"},
			scheme:   "https",
			host:     "example.net",
			port:     443,
			endpoint: "https://example.net:443",
		},
		{
			name:     "ipv6 endpoint",
			result:   Result{URL: "https://[2001:db8::1]:8443/"},
			scheme:   "https",
			host:     "2001:db8::1",
			port:     8443,
			endpoint: "https://[2001:db8::1]:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.result.EnrichEndpoint()
			if tt.result.SchemaVersion != ResultSchemaVersion {
				t.Fatalf("SchemaVersion = %q, want %q", tt.result.SchemaVersion, ResultSchemaVersion)
			}
			if tt.result.Scheme != tt.scheme || tt.result.Host != tt.host || tt.result.Port != tt.port || tt.result.Endpoint != tt.endpoint {
				t.Fatalf("endpoint fields = %s %s %d %s, want %s %s %d %s",
					tt.result.Scheme, tt.result.Host, tt.result.Port, tt.result.Endpoint,
					tt.scheme, tt.host, tt.port, tt.endpoint)
			}
		})
	}
}

func TestResultJSONOmitsScreenshotBytes(t *testing.T) {
	result := Result{
		URL:             "https://example.com",
		Screenshot:      "/tmp/shot.png",
		ScreenshotBytes: []byte{0x89, 'P', 'N', 'G'},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonText := string(data)
	if strings.Contains(jsonText, "ScreenshotBytes") || strings.Contains(jsonText, "screenshot_bytes") {
		t.Fatalf("ScreenshotBytes should not be exposed in JSON: %s", jsonText)
	}
	if !strings.Contains(jsonText, `"screenshot":"/tmp/shot.png"`) {
		t.Fatalf("screenshot path should still be exposed in JSON: %s", jsonText)
	}
}

func TestEnrichEndpoint_PackageLevel_NilSafe(t *testing.T) {
	EnrichEndpoint(nil) // 不应 panic

	r := &Result{URL: "http://example.com:8080/path"}
	EnrichEndpoint(r)
	if r.Host == "" {
		t.Fatal("EnrichEndpoint 后 Host 不应为空")
	}
	if r.Scheme != "http" {
		t.Fatalf("Scheme = %q, want http", r.Scheme)
	}
	if r.Port != 8080 {
		t.Fatalf("Port = %d, want 8080", r.Port)
	}
	if r.Endpoint != "http://example.com:8080" {
		t.Fatalf("Endpoint = %q, want http://example.com:8080", r.Endpoint)
	}
	if r.SchemaVersion != ResultSchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", r.SchemaVersion, ResultSchemaVersion)
	}
}

func TestEnrichEndpoint_EmptyURLNoPanic(t *testing.T) {
	r := &Result{} // URL 与 FinalURL 均为空
	EnrichEndpoint(r)
	// 不应 panic，且 SchemaVersion 仍被设置
	if r.SchemaVersion != ResultSchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", r.SchemaVersion, ResultSchemaVersion)
	}
	if r.Host != "" || r.Scheme != "" || r.Port != 0 || r.Endpoint != "" {
		t.Fatalf("空 URL 不应填充端点字段: %+v", r)
	}
}

func TestDefaultPortForScheme(t *testing.T) {
	tests := []struct {
		scheme string
		want   int
	}{
		{"http", 80},
		{"https", 443},
		{"HTTP", 80},
		{"HTTPS", 443},
		{"ftp", 0},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			if got := DefaultPortForScheme(tt.scheme); got != tt.want {
				t.Fatalf("DefaultPortForScheme(%q) = %d, want %d", tt.scheme, got, tt.want)
			}
		})
	}
}

func TestResult_HeaderMap(t *testing.T) {
	r := &Result{
		Headers: []Header{{Name: "X-A", Value: "1"}, {Name: "X-A", Value: "2"}, {Name: "X-B", Value: "3"}},
	}
	m := r.HeaderMap()
	if len(m["X-A"]) != 2 || m["X-A"][0] != "1" || m["X-A"][1] != "2" {
		t.Fatalf("HeaderMap X-A = %v", m["X-A"])
	}
	if len(m["X-B"]) != 1 || m["X-B"][0] != "3" {
		t.Fatalf("HeaderMap X-B = %v", m["X-B"])
	}
}

func TestEnrichEndpoint_UnparseableURL(t *testing.T) {
	// 无 scheme 且兜底 https:// 后 Host 仍为空 → 提前 return，不填充端点
	r := &Result{URL: "://bad"}
	EnrichEndpoint(r)
	if r.Host != "" || r.Endpoint != "" {
		t.Fatalf("不可解析 URL 不应填充端点: %+v", r)
	}
}

func TestEnrichEndpoint_OnlyFinalURL(t *testing.T) {
	// URL 为空、FinalURL 非空 → 用 FinalURL 兜底（已部分覆盖，强化断言）
	r := &Result{URL: "   ", FinalURL: "https://final.example.com/x"}
	EnrichEndpoint(r)
	if r.Host != "final.example.com" {
		t.Fatalf("Host = %q, want final.example.com", r.Host)
	}
	if r.Port != 443 {
		t.Fatalf("Port = %d, want 443", r.Port)
	}
	if r.Endpoint != "https://final.example.com:443" {
		t.Fatalf("Endpoint = %q", r.Endpoint)
	}
}

func TestEnrichEndpoint_NilReceiver(t *testing.T) {
	var r *Result
	r.EnrichEndpoint() // 不应 panic
	if r != nil {
		t.Fatalf("nil receiver 应保持 nil")
	}
}
