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

// TestHTTPGetResult_SuccessFalseWithNoError 覆盖 doSingleResult 的
// Success=false 且 Error 为空分支（http_client.go:82-85）。
func TestHTTPGetResult_SuccessFalseWithNoError(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":false}`))
	})
	defer cleanup()
	_, err := c.GetResult(1)
	if err == nil {
		t.Error("Success=false 应返回错误")
	}
}

// TestHTTPGetResult_SuccessFalseWithError 覆盖 doSingleResult 的
// Success=false 且 Error 非空分支（http_client.go:86-87）。
func TestHTTPGetResult_SuccessFalseWithError(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":false,"error":"数据库查询失败"}`))
	})
	defer cleanup()
	_, err := c.GetResult(1)
	if err == nil {
		t.Error("Success=false 应返回错误")
	}
}

// TestHTTPGetResult_InvalidJSON 覆盖 doSingleResult 的 json.Unmarshal 失败分支
// （http_client.go:79-81）。
func TestHTTPGetResult_InvalidJSON(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not-valid-json{`))
	})
	defer cleanup()
	_, err := c.GetResult(1)
	if err == nil {
		t.Error("无效 JSON 应返回错误")
	}
}

// TestHTTPGetResult_500Error 覆盖 doSingleResult 的非 200/503 错误分支
// （http_client.go:71-73）。
func TestHTTPGetResult_500Error(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`内部错误`))
	})
	defer cleanup()
	_, err := c.GetResult(1)
	if err == nil {
		t.Error("500 应返回错误")
	}
}

// TestHTTPListResults_SuccessFalseNoError 覆盖 doListResult 的
// Success=false 无 Error 分支（http_client.go:111-114）。
func TestHTTPListResults_SuccessFalseNoError(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":false}`))
	})
	defer cleanup()
	_, err := c.ListResults(10)
	if err == nil {
		t.Error("Success=false 应返回错误")
	}
}

// TestHTTPListResults_SuccessFalseWithError 覆盖 doListResult 的
// Success=false 有 Error 分支（http_client.go:115-116）。
func TestHTTPListResults_SuccessFalseWithError(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":false,"error":"查询超时"}`))
	})
	defer cleanup()
	_, err := c.ListResults(10)
	if err == nil {
		t.Error("Success=false 应返回错误")
	}
}

// TestHTTPListResults_InvalidJSON 覆盖 doListResult 的 Unmarshal 失败分支
// （http_client.go:108-110）。
func TestHTTPListResults_InvalidJSON(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not-json`))
	})
	defer cleanup()
	_, err := c.ListResults(10)
	if err == nil {
		t.Error("无效 JSON 应返回错误")
	}
}

// TestHTTPListResults_500Error 覆盖 doListResult 的非 200/503 分支
// （http_client.go:100-102）。
func TestHTTPListResults_500Error(t *testing.T) {
	c, cleanup := newMockResultsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`服务器错误`))
	})
	defer cleanup()
	_, err := c.ListResults(10)
	if err == nil {
		t.Error("500 应返回错误")
	}
}

// TestHTTPGetResult_NewRequestError 覆盖 GetResult 的 http.NewRequest 失败分支
// （http_client.go:122-124）。baseURL 含控制字符使 NewRequest 失败。
func TestHTTPGetResult_NewRequestError(t *testing.T) {
	c := NewHTTPClient(HTTPClientOptions{BaseURL: "http://exa\x00mple.com"})
	if _, err := c.GetResult(1); err == nil {
		t.Error("无效 baseURL 应返回 NewRequest 错误")
	}
}

// TestHTTPGetResultByURL_NewRequestError 覆盖 GetResultByURL 的 NewRequest 失败分支
// （http_client.go:134-136）。
func TestHTTPGetResultByURL_NewRequestError(t *testing.T) {
	c := NewHTTPClient(HTTPClientOptions{BaseURL: "http://exa\x00mple.com"})
	if _, err := c.GetResultByURL("https://example.com"); err == nil {
		t.Error("无效 baseURL 应返回 NewRequest 错误")
	}
}

// TestHTTPListResults_NewRequestError 覆盖 ListResults 的 NewRequest 失败分支
// （http_client.go:148-150）。
func TestHTTPListResults_NewRequestError(t *testing.T) {
	c := NewHTTPClient(HTTPClientOptions{BaseURL: "http://exa\x00mple.com"})
	if _, err := c.ListResults(10); err == nil {
		t.Error("无效 baseURL 应返回 NewRequest 错误")
	}
}

// TestHTTPDoRaw_DoError 覆盖 doRaw 的 httpClient.Do 失败分支（http_client.go:53-55）。
// 指向不可达地址让 Do 立即失败。
func TestHTTPDoRaw_DoError(t *testing.T) {
	c := NewHTTPClient(HTTPClientOptions{BaseURL: "http://127.0.0.1:1", Timeout: 1})
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1/results/1", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, _, err := c.doRaw(req); err == nil {
		t.Error("不可达地址应返回 Do 错误")
	}
}
