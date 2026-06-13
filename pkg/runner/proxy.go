package runner

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
)

// ProxyProvider 代理提供者接口
// 支持多种代理模式：静态代理、代理列表轮换、代理文件、代理 API
type ProxyProvider interface {
	// GetProxy 获取一个代理地址
	// 每次截图调用时获取，支持轮换
	GetProxy() (string, error)
	// Name 返回提供者名称（用于日志）
	Name() string
}

// StaticProxy 静态代理（不轮换）
// 适合长效代理场景
type StaticProxy struct {
	proxy string
}

// NewStaticProxy 创建静态代理
func NewStaticProxy(proxy string) *StaticProxy {
	return &StaticProxy{proxy: proxy}
}

func (p *StaticProxy) GetProxy() (string, error) {
	return p.proxy, nil
}

func (p *StaticProxy) Name() string {
	return "static"
}

// ProxyList 代理列表轮换
// 从给定的代理列表中按策略轮换：顺序、随机、轮询
type ProxyList struct {
	proxies  []string
	strategy ProxyStrategy
	index    int
	mu       sync.Mutex
	rand     *rand.Rand
}

// ProxyStrategy 代理轮换策略
type ProxyStrategy string

const (
	// ProxyRoundRobin 轮询策略（依次使用每个代理）
	ProxyRoundRobin ProxyStrategy = "round-robin"
	// ProxyRandom 随机策略（每次随机选择）
	ProxyRandom ProxyStrategy = "random"
	// ProxySequential 顺序策略（从第一个开始，失败后切换）
	ProxySequential ProxyStrategy = "sequential"
)

// NewProxyList 创建代理列表轮换
func NewProxyList(proxies []string, strategy ProxyStrategy) *ProxyList {
	if len(proxies) == 0 {
		return nil
	}
	return &ProxyList{
		proxies:  proxies,
		strategy: strategy,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (p *ProxyList) GetProxy() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return "", fmt.Errorf("代理列表为空")
	}

	switch p.strategy {
	case ProxyRoundRobin:
		proxy := p.proxies[p.index%len(p.proxies)]
		p.index++
		return proxy, nil
	case ProxyRandom:
		return p.proxies[p.rand.Intn(len(p.proxies))], nil
	case ProxySequential:
		proxy := p.proxies[p.index]
		if p.index < len(p.proxies)-1 {
			p.index++
		}
		return proxy, nil
	default:
		return p.proxies[0], nil
	}
}

func (p *ProxyList) Name() string {
	return fmt.Sprintf("proxy-list(%d,%s)", len(p.proxies), p.strategy)
}

// ProxyFile 从文件加载代理列表
// 支持实时重载（文件变化时自动更新代理列表）
type ProxyFile struct {
	filePath string
	strategy ProxyStrategy
	list     *ProxyList
	mu       sync.RWMutex
	lastMod  time.Time
}

// NewProxyFile 创建文件代理提供者
// filePath: 代理文件路径（每行一个代理地址，支持 # 注释）
// strategy: 轮换策略
func NewProxyFile(filePath string, strategy ProxyStrategy) (*ProxyFile, error) {
	proxies, err := loadProxyFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("加载代理文件失败: %v", err)
	}

	log.Info("从文件加载代理列表", "file", filePath, "count", len(proxies), "strategy", strategy)

	pf := &ProxyFile{
		filePath: filePath,
		strategy: strategy,
		list:     NewProxyList(proxies, strategy),
	}

	// 记录文件修改时间
	if info, err := os.Stat(filePath); err == nil {
		pf.lastMod = info.ModTime()
	}

	return pf, nil
}

func (pf *ProxyFile) GetProxy() (string, error) {
	pf.mu.RLock()
	list := pf.list
	pf.mu.RUnlock()

	// 检查文件是否需要重载
	pf.reloadIfNeeded()

	if list == nil {
		return "", fmt.Errorf("代理列表为空")
	}
	return list.GetProxy()
}

func (pf *ProxyFile) Name() string {
	return fmt.Sprintf("proxy-file(%s)", pf.filePath)
}

// reloadIfNeeded 检查文件是否更新，需要时重载代理列表
func (pf *ProxyFile) reloadIfNeeded() {
	info, err := os.Stat(pf.filePath)
	if err != nil {
		return
	}

	if info.ModTime().After(pf.lastMod) {
		proxies, err := loadProxyFile(pf.filePath)
		if err != nil {
			log.Warn("重载代理文件失败", "file", pf.filePath, "error", err)
			return
		}

		pf.mu.Lock()
		pf.list = NewProxyList(proxies, pf.strategy)
		pf.lastMod = info.ModTime()
		pf.mu.Unlock()

		log.Info("代理文件已重新加载", "file", pf.filePath, "count", len(proxies))
	}
}

// loadProxyFile 从文件加载代理列表
// 格式：每行一个代理地址，支持 # 注释，空行跳过
func loadProxyFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 支持格式: host:port, http://host:port, socks5://host:port
		if !strings.Contains(line, "://") {
			line = "http://" + line
		}
		proxies = append(proxies, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("代理文件为空: %s", filePath)
	}

	return proxies, nil
}

// NoProxy 无代理（直连）
type NoProxy struct{}

func NewNoProxy() *NoProxy {
	return &NoProxy{}
}

func (p *NoProxy) GetProxy() (string, error) {
	return "", nil
}

func (p *NoProxy) Name() string {
	return "direct"
}

// CreateProxyProvider 根据配置创建代理提供者
// 优先级：ProxyURL（代理 API） > ProxyFile > ProxyList > Proxy（静态）
func CreateProxyProvider(opts *Options) ProxyProvider {
	// 优先级 1: 代理 API URL（动态代理）
	if opts.Chrome.ProxyURL != "" {
		log.Info("使用动态代理 API", "url", opts.Chrome.ProxyURL)
		return NewProxyAPI(opts.Chrome.ProxyURL, opts.Chrome.ProxyStrategy)
	}

	// 优先级 2: 代理文件
	if opts.Chrome.ProxyFile != "" {
		pf, err := NewProxyFile(opts.Chrome.ProxyFile, opts.Chrome.ProxyStrategy)
		if err != nil {
			log.Warn("加载代理文件失败，将使用静态代理", "error", err)
			if opts.Chrome.Proxy != "" {
				return NewStaticProxy(opts.Chrome.Proxy)
			}
			return NewNoProxy()
		}
		return pf
	}

	// 优先级 3: 代理列表
	if len(opts.Chrome.ProxyList) > 0 {
		return NewProxyList(opts.Chrome.ProxyList, opts.Chrome.ProxyStrategy)
	}

	// 优先级 4: 静态代理
	if opts.Chrome.Proxy != "" {
		return NewStaticProxy(opts.Chrome.Proxy)
	}

	// 无代理
	return NewNoProxy()
}

// ProxyAPI 从远程 API 获取代理
// 适合动态代理服务（每次请求返回不同的代理 IP）
type ProxyAPI struct {
	url      string
	strategy ProxyStrategy // 仅用于缓存多个结果时的策略
	client   *simpleHTTPClient
	cache    []string // 缓存最近的代理
	cacheMu  sync.Mutex
	cacheIdx int
}

// NewProxyAPI 创建动态代理 API 提供者
// url: 代理 API 地址，GET 请求返回代理地址文本
func NewProxyAPI(url string, strategy ProxyStrategy) *ProxyAPI {
	return &ProxyAPI{
		url:      url,
		strategy: strategy,
		client:   &simpleHTTPClient{timeout: 5 * time.Second},
	}
}

func (p *ProxyAPI) GetProxy() (string, error) {
	// 每次都从 API 获取最新代理
	body, err := p.client.Get(p.url)
	if err != nil {
		// 如果 API 失败，尝试使用缓存
		p.cacheMu.Lock()
		if len(p.cache) > 0 {
			proxy := p.cache[p.cacheIdx%len(p.cache)]
			p.cacheIdx++
			p.cacheMu.Unlock()
			log.Warn("代理 API 请求失败，使用缓存", "error", err, "proxy", proxy)
			return proxy, nil
		}
		p.cacheMu.Unlock()
		return "", fmt.Errorf("代理 API 请求失败: %v", err)
	}

	proxy := strings.TrimSpace(body)
	if proxy == "" {
		return "", fmt.Errorf("代理 API 返回空结果")
	}

	// 补充协议前缀
	if !strings.Contains(proxy, "://") {
		proxy = "http://" + proxy
	}

	// 缓存代理
	p.cacheMu.Lock()
	p.cache = append(p.cache, proxy)
	if len(p.cache) > 10 {
		p.cache = p.cache[len(p.cache)-10:]
	}
	p.cacheMu.Unlock()

	return proxy, nil
}

func (p *ProxyAPI) Name() string {
	return fmt.Sprintf("proxy-api(%s)", p.url)
}

// simpleHTTPClient 超简单的 HTTP GET 客户端
type simpleHTTPClient struct {
	timeout time.Duration
}

func (c *simpleHTTPClient) Get(url string) (string, error) {
	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
