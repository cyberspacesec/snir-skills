package sdk

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newMockResultsServer(t *testing.T, handler http.HandlerFunc) (*HTTPClient, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewHTTPClient(HTTPClientOptions{BaseURL: srv.URL, APIKey: "test"})
	return c, srv.Close
}

func TestHTTPGetResult_HappyPath(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/results/1" {
			t.Errorf("path = %s, want /results/1", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test" {
			t.Errorf("api key header = %q", r.Header.Get("X-API-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":{"id":1,"url":"https://example.com/","response_code":200}}`))
	})
	defer cleanup()
	res, err := c.GetResult(1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res == nil || res.ResponseCode != 200 {
		t.Errorf("got %+v, want response_code=200", res)
	}
}

func TestHTTPGetResult_DBNotConfigured(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"success":false,"error":"未启用数据库"}`))
	})
	defer cleanup()
	if _, err := c.GetResult(1); err == nil {
		t.Error("expected error for 503, got nil")
	}
}

func TestHTTPGetResult_NotFound(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":"未找到 id=9999 的结果"}`))
	})
	defer cleanup()
	if _, err := c.GetResult(9999); err == nil {
		t.Error("expected error for 404, got nil")
	}
}

func TestHTTPGetResultByURL_HappyPath(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/results/by-url" {
			t.Errorf("path = %s, want /results/by-url", r.URL.Path)
		}
		if r.URL.Query().Get("url") != "https://example.com/" {
			t.Errorf("url param = %q", r.URL.Query().Get("url"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":[{"id":1,"url":"https://example.com/","response_code":200}]}`))
	})
	defer cleanup()
	results, err := c.GetResultByURL("https://example.com/")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ResponseCode != 200 {
		t.Errorf("response_code = %d, want 200", results[0].ResponseCode)
	}
}

func TestHTTPListResults_HappyPath(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/results" {
			t.Errorf("path = %s, want /results", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":[{"id":1},{"id":2}]}`))
	})
	defer cleanup()
	results, err := c.ListResults(0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestHTTPNewHTTPClient_TrimsTrailingSlash(t *testing.T) {
	c := NewHTTPClient(HTTPClientOptions{BaseURL: "http://127.0.0.1:8080///", APIKey: "k"})
	if c.baseURL != "http://127.0.0.1:8080" {
		t.Errorf("baseURL = %q, want http://127.0.0.1:8080", c.baseURL)
	}
	// 无尾斜杠的 baseURL 应原样保留
	c2 := NewHTTPClient(HTTPClientOptions{BaseURL: "http://127.0.0.1:8080"})
	if c2.baseURL != "http://127.0.0.1:8080" {
		t.Errorf("baseURL = %q, want http://127.0.0.1:8080", c2.baseURL)
	}
}

func TestHTTPListResults_WithLimit(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "50" {
			t.Errorf("limit param = %q, want 50", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":[{"id":1}]}`))
	})
	defer cleanup()
	results, err := c.ListResults(50)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1", len(results))
	}
}

func TestHTTPGetResultByURL_SpecialChars(t *testing.T) {
	// 含 query 字符的 URL，验证 url.Values 正确编码与还原
	target := "https://example.com/path?a=1&b=2"
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("url")
		if got != target {
			t.Errorf("url param = %q, want %q", got, target)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":[]}`))
	})
	defer cleanup()
	results, err := c.GetResultByURL(target)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
