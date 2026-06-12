package runner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/log"
)

// ChromeInstance 表示一个已发现的 Chrome 实例
type ChromeInstance struct {
	WsURL      string // WebSocket 调试 URL
	BrowserURL string // HTTP 调试地址
	Port       int    // 远程调试端口
	Version    string // Chrome 版本
}

// DiscoverChrome 尝试发现本地已运行的 Chrome 实例
// 扫描指定端口列表，查询 /json/version 获取 WebSocket URL
// 如果不传端口列表，默认扫描 9222, 9223, 9224
func DiscoverChrome(host string, ports []int) (*ChromeInstance, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	if len(ports) == 0 {
		ports = []int{9222, 9223, 9224}
	}

	for _, port := range ports {
		instance, err := probeChromePort(host, port)
		if err == nil {
			log.Info("发现本地 Chrome 实例", "port", port, "ws_url", instance.WsURL)
			return instance, nil
		}
	}

	return nil, fmt.Errorf("未发现本地 Chrome 实例 (扫描 %s:%v)", host, ports)
}

// DiscoverChromeWithTimeout 带超时的 Chrome 发现
func DiscoverChromeWithTimeout(host string, ports []int, timeout time.Duration) (*ChromeInstance, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	if len(ports) == 0 {
		ports = []int{9222, 9223, 9224}
	}

	client := &http.Client{Timeout: timeout}

	for _, port := range ports {
		instance, err := probeChromePortWithClient(host, port, client)
		if err == nil {
			log.Info("发现本地 Chrome 实例", "port", port, "ws_url", instance.WsURL)
			return instance, nil
		}
	}

	return nil, fmt.Errorf("未发现本地 Chrome 实例 (扫描 %s:%v, 超时 %v)", host, ports, timeout)
}

// probeChromePort 探测指定端口的 Chrome 实例
func probeChromePort(host string, port int) (*ChromeInstance, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	return probeChromePortWithClient(host, port, client)
}

// probeChromePortWithClient 使用指定 HTTP 客户端探测
func probeChromePortWithClient(host string, port int, client *http.Client) (*ChromeInstance, error) {
	url := fmt.Sprintf("http://%s:%d/json/version", host, port)

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 Chrome 版本信息失败: %v", err)
	}

	wsURL, ok := result["webSocketDebuggerUrl"].(string)
	if !ok || wsURL == "" {
		return nil, fmt.Errorf("Chrome 实例未返回 WebSocket URL")
	}

	version, _ := result["Browser"].(string)

	return &ChromeInstance{
		WsURL:      wsURL,
		BrowserURL: fmt.Sprintf("http://%s:%d", host, port),
		Port:       port,
		Version:    version,
	}, nil
}

// AutoConnect 自动连接模式
// 优先级：
// 1. 如果指定了 WSS URL，直接连接远程 Chrome
// 2. 尝试发现本地已运行的 Chrome 实例
// 3. 如果都没有，启动新的 Chrome 进程
// 返回 DriverPool 和使用的连接模式
func AutoConnect(opts *Options, maxConcurrent int) (*DriverPool, string, error) {
	// 优先级 1: 指定了 WSS URL
	if opts.Chrome.WSS != "" {
		pool, err := NewDriverPool(opts, maxConcurrent)
		if err != nil {
			return nil, "", fmt.Errorf("连接远程 Chrome 失败: %v", err)
		}
		return pool, "remote", nil
	}

	// 优先级 2: 尝试发现本地 Chrome
	instance, err := DiscoverChrome("127.0.0.1", nil)
	if err == nil {
		remoteOpts := *opts
		remoteOpts.Chrome.WSS = instance.WsURL
		pool, err := NewDriverPool(&remoteOpts, maxConcurrent)
		if err == nil {
			return pool, "discovered", nil
		}
		// 发现了但连接失败，回退到启动新实例
		log.Warn("发现本地 Chrome 但连接失败，启动新实例", "error", err)
	}

	// 优先级 3: 启动新的 Chrome 进程
	pool, err := NewDriverPool(opts, maxConcurrent)
	if err != nil {
		return nil, "", fmt.Errorf("启动 Chrome 实例失败: %v", err)
	}
	return pool, "local", nil
}
