package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// HTTPClient 是 snir HTTP API 的轻量客户端，用于结果检索等只读端点。
// 与 Client（进程内 driver）不同，HTTPClient 不启动 Chrome，仅发 HTTP 请求。
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// HTTPClientOptions 配置 HTTPClient
type HTTPClientOptions struct {
	BaseURL string // snir api 地址，如 "http://127.0.0.1:8080"
	APIKey  string // X-API-Key 鉴权密钥
	Timeout time.Duration
}

// NewHTTPClient 创建 HTTP API 客户端
func NewHTTPClient(opts HTTPClientOptions) *HTTPClient {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPClient{
		baseURL:    strings.TrimRight(opts.BaseURL, "/"),
		apiKey:     opts.APIKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// addAuth 注入鉴权头
func (c *HTTPClient) addAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}

// doRaw 执行请求返回 body + status
func (c *HTTPClient) doRaw(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// doSingleResult 发请求并解析 APIResponse 信封到单个 Result
func (c *HTTPClient) doSingleResult(req *http.Request) (*models.Result, error) {
	body, status, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}
	if status == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("服务端未启用数据库：请用 --db-path 启动 snir api")
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(body))
	}
	var apiResp struct {
		Success bool           `json:"success"`
		Error   string         `json:"error,omitempty"`
		Data    *models.Result `json:"data,omitempty"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if !apiResp.Success {
		if apiResp.Error == "" {
			return nil, fmt.Errorf("API 返回失败（未提供错误信息）")
		}
		return nil, fmt.Errorf("%s", apiResp.Error)
	}
	return apiResp.Data, nil
}

// doListResult 发请求并解析 APIResponse 信封到 Result 切片
func (c *HTTPClient) doListResult(req *http.Request) ([]*models.Result, error) {
	body, status, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}
	if status == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("服务端未启用数据库：请用 --db-path 启动 snir api")
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(body))
	}
	var apiResp struct {
		Success bool             `json:"success"`
		Error   string           `json:"error,omitempty"`
		Data    []*models.Result `json:"data,omitempty"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if !apiResp.Success {
		if apiResp.Error == "" {
			return nil, fmt.Errorf("API 返回失败（未提供错误信息）")
		}
		return nil, fmt.Errorf("%s", apiResp.Error)
	}
	return apiResp.Data, nil
}

// GetResult 按主键 id 检索单个历史扫描结果。
func (c *HTTPClient) GetResult(id uint64) (*models.Result, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/results/%d", c.baseURL, id), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	return c.doSingleResult(req)
}

// GetResultByURL 按精确 URL 查询该 URL 的所有历史扫描记录（按 probed_at 倒序）。
func (c *HTTPClient) GetResultByURL(rawURL string) ([]*models.Result, error) {
	q := url.Values{}
	q.Set("url", rawURL)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/results/by-url?%s", c.baseURL, q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	return c.doListResult(req)
}

// ListResults 列出所有历史扫描结果（按 probed_at 倒序）。
// limit<=0 用服务端默认 100，>1000 截断为 1000。
func (c *HTTPClient) ListResults(limit int) ([]*models.Result, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/results?%s", c.baseURL, q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)
	return c.doListResult(req)
}
