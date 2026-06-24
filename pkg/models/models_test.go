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
