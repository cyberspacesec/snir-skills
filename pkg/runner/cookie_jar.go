package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/log"
)

// CookieJar Cookie 持久化存储
// 支持按域名保存/加载 Cookie 到 JSON 文件
// 支持一次性 Cookie 和持久化 Cookie
type CookieJar struct {
	filePath string
	cookies  map[string][]PersistentCookie // key = domain
	mu       sync.RWMutex
}

// PersistentCookie 持久化 Cookie（扩展 CustomCookie 增加 TTL 和持久化标记）
type PersistentCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain,omitempty"`
	Path     string `json:"path,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HttpOnly bool   `json:"http_only,omitempty"`

	// 持久化控制
	Persistent bool   `json:"persistent"`             // true=持久化保存，false=一次性使用
	ExpiresAt  int64  `json:"expires_at,omitempty"`   // Unix 时间戳，0=永不过期
	Source     string `json:"source,omitempty"`        // 来源标记：manual/api/file/session
}

// IsExpired 检查 Cookie 是否已过期
func (c *PersistentCookie) IsExpired() bool {
	if c.ExpiresAt == 0 {
		return false // 永不过期
	}
	return time.Now().Unix() > c.ExpiresAt
}

// ToCustomCookie 转换为 runner.CustomCookie
func (c *PersistentCookie) ToCustomCookie() CustomCookie {
	return CustomCookie{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   c.Domain,
		Path:     c.Path,
		Secure:   c.Secure,
		HttpOnly: c.HttpOnly,
	}
}

// NewCookieJar 创建 Cookie 持久化存储
// filePath: Cookie 存储文件路径（JSON 格式）
// 如果文件不存在，创建空存储
func NewCookieJar(filePath string) (*CookieJar, error) {
	jar := &CookieJar{
		filePath: filePath,
		cookies:  make(map[string][]PersistentCookie),
	}

	// 尝试加载已有文件
	if _, err := os.Stat(filePath); err == nil {
		if err := jar.load(); err != nil {
			log.Warn("加载 Cookie 文件失败，创建新存储", "file", filePath, "error", err)
		} else {
			count := jar.count()
			log.Info("Cookie 存储已加载", "file", filePath, "cookies", count)
		}
	}

	return jar, nil
}

// AddCookie 添加一个 Cookie
// persistent=true 会保存到文件，false 只在内存中（一次性）
func (jar *CookieJar) AddCookie(cookie PersistentCookie) error {
	jar.mu.Lock()
	defer jar.mu.Unlock()

	domain := cookie.Domain
	if domain == "" {
		domain = "_global"
	}

	// 检查是否已存在同名 Cookie（更新而非追加）
	cookies := jar.cookies[domain]
	replaced := false
	for i, c := range cookies {
		if c.Name == cookie.Name {
			cookies[i] = cookie
			replaced = true
			break
		}
	}
	if !replaced {
		cookies = append(cookies, cookie)
	}
	jar.cookies[domain] = cookies

	// 持久化 Cookie 立即保存
	if cookie.Persistent {
		return jar.save()
	}
	return nil
}

// AddCookies 批量添加 Cookie
func (jar *CookieJar) AddCookies(cookies []PersistentCookie) error {
	jar.mu.Lock()
	defer jar.mu.Unlock()

	for _, cookie := range cookies {
		domain := cookie.Domain
		if domain == "" {
			domain = "_global"
		}

		existing := jar.cookies[domain]
		replaced := false
		for i, c := range existing {
			if c.Name == cookie.Name {
				existing[i] = cookie
				replaced = true
				break
			}
		}
		if !replaced {
			existing = append(existing, cookie)
		}
		jar.cookies[domain] = existing
	}

	// 保存持久化 Cookie
	return jar.save()
}

// GetCookies 获取指定域名的 Cookie（转换为 CustomCookie 列表）
// 一次性 Cookie 在获取后自动移除
func (jar *CookieJar) GetCookies(domain string) []CustomCookie {
	jar.mu.Lock()
	defer jar.mu.Unlock()

	var result []CustomCookie
	needsSave := false

	// 获取全局 Cookie
	if global, ok := jar.cookies["_global"]; ok {
		var remaining []PersistentCookie
		for _, c := range global {
			if c.IsExpired() {
				needsSave = true
				continue
			}
			result = append(result, c.ToCustomCookieWithExpires())
			if c.Persistent {
				remaining = append(remaining, c)
			} else {
				needsSave = true // 一次性 Cookie 被消费
			}
		}
		jar.cookies["_global"] = remaining
	}

	// 获取域名 Cookie（支持子域名匹配）
	// 遍历所有域名，检查请求域名是否匹配 Cookie 域名
	// .example.com 的 Cookie 匹配 sub.example.com 和 example.com
	for cookieDomain, domainCookies := range jar.cookies {
		if cookieDomain == "_global" {
			continue // 全局 Cookie 已处理
		}
		if !domainMatches(domain, cookieDomain) {
			continue
		}
		var remaining []PersistentCookie
		for _, c := range domainCookies {
			if c.IsExpired() {
				needsSave = true
				continue
			}
			result = append(result, c.ToCustomCookieWithExpires())
			if c.Persistent {
				remaining = append(remaining, c)
			} else {
				needsSave = true // 一次性 Cookie 被消费
			}
		}
		jar.cookies[cookieDomain] = remaining
	}

	if needsSave {
		jar.save()
	}

	return result
}

// GetAllCookies 获取所有域名的 Cookie
func (jar *CookieJar) GetAllCookies() []CustomCookie {
	jar.mu.RLock()
	defer jar.mu.RUnlock()

	var result []CustomCookie
	for _, cookies := range jar.cookies {
		for _, c := range cookies {
			if !c.IsExpired() {
				result = append(result, c.ToCustomCookieWithExpires())
			}
		}
	}
	return result
}

// RemoveCookie 删除指定域名的指定 Cookie
func (jar *CookieJar) RemoveCookie(domain, name string) error {
	jar.mu.Lock()
	defer jar.mu.Unlock()

	if domain == "" {
		domain = "_global"
	}

	cookies, ok := jar.cookies[domain]
	if !ok {
		return nil
	}

	var remaining []PersistentCookie
	for _, c := range cookies {
		if c.Name != name {
			remaining = append(remaining, c)
		}
	}
	jar.cookies[domain] = remaining

	return jar.save()
}

// Clear 清空所有 Cookie
func (jar *CookieJar) Clear() error {
	jar.mu.Lock()
	defer jar.mu.Unlock()

	jar.cookies = make(map[string][]PersistentCookie)
	return jar.save()
}

// Count 返回 Cookie 数量
func (jar *CookieJar) count() int {
	count := 0
	for _, cookies := range jar.cookies {
		for _, c := range cookies {
			if !c.IsExpired() {
				count++
			}
		}
	}
	return count
}

// Count 返回 Cookie 数量（线程安全）
func (jar *CookieJar) Count() int {
	jar.mu.RLock()
	defer jar.mu.RUnlock()
	return jar.count()
}

// Domains 返回所有域名
func (jar *CookieJar) Domains() []string {
	jar.mu.RLock()
	defer jar.mu.RUnlock()

	var domains []string
	for d := range jar.cookies {
		if d != "_global" {
			domains = append(domains, d)
		}
	}
	return domains
}

// load 从文件加载 Cookie
func (jar *CookieJar) load() error {
	data, err := os.ReadFile(jar.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &jar.cookies)
}

// save 保存 Cookie 到文件
func (jar *CookieJar) save() error {
	// 确保目录存在
	dir := filepath.Dir(jar.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建 Cookie 目录失败: %v", err)
		}
	}

	// 清理过期 Cookie
	jar.cleanExpired()

	data, err := json.MarshalIndent(jar.cookies, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 Cookie 失败: %v", err)
	}

	return os.WriteFile(jar.filePath, data, 0644)
}

// cleanExpired 清理过期 Cookie
func (jar *CookieJar) cleanExpired() {
	for domain, cookies := range jar.cookies {
		var valid []PersistentCookie
		for _, c := range cookies {
			if !c.IsExpired() {
				valid = append(valid, c)
			}
		}
		if len(valid) == 0 {
			delete(jar.cookies, domain)
		} else {
			jar.cookies[domain] = valid
		}
	}
}

// CookieJarToCustomCookies 将 CookieJar 中的 Cookie 转换为 runner 选项
// 如果指定了 domain，返回该域名的 Cookie + 全局 Cookie
// 如果 domain 为空，返回所有 Cookie
func CookieJarToCustomCookies(jar *CookieJar, domain string) []CustomCookie {
	if jar == nil {
		return nil
	}
	return jar.GetCookies(domain)
}


// domainMatches 检查请求域名是否匹配 Cookie 域名
// 遵循 RFC 6265 的域名匹配规则：
//   - .example.com 匹配 sub.example.com 和 example.com
//   - example.com 匹配 example.com 和 sub.example.com
//   - .example.com 不匹配 notexample.com
func domainMatches(requestDomain, cookieDomain string) bool {
	// 精确匹配
	if requestDomain == cookieDomain {
		return true
	}

	// Cookie 域名以 . 开头（表示包含子域名）
	if strings.HasPrefix(cookieDomain, ".") {
		suffix := cookieDomain[1:] // 去掉前导 .
		// 请求域名等于后缀 或 以 .后缀 结尾
		if requestDomain == suffix || strings.HasSuffix(requestDomain, "."+suffix) {
			return true
		}
	}

	// Cookie 域名不以 . 开头但请求域名是子域名
	// example.com 匹配 sub.example.com
	if strings.HasSuffix(requestDomain, "."+cookieDomain) {
		return true
	}

	return false
}
