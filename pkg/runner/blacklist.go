package runner

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/cyberspacesec/snir-skills/pkg/log"
)

// DefaultBlacklist 包含默认的黑名单规则
var DefaultBlacklist = []string{
	// 本地和内网地址
	"localhost",
	"127.0.0.0/8",    // 本地环回
	"10.0.0.0/8",     // RFC1918私有地址
	"172.16.0.0/12",  // RFC1918私有地址
	"192.168.0.0/16", // RFC1918私有地址
	"169.254.0.0/16", // 链路本地地址 (包含AWS元数据服务)
	"fc00::/7",       // 唯一本地地址 (IPv6)

	// 云服务元数据地址
	"169.254.169.254",          // AWS/GCP/DigitalOcean 元数据
	"metadata.google.internal", // GCP
	"metadata.internal",        // DigitalOcean
	"metadata.service",         // Azure

	// 敏感内部服务默认地址
	"consul.service.consul",
	"vault.service.consul",

	// 本地服务常用端口
	".*:1433",  // MSSQL
	".*:3306",  // MySQL
	".*:5432",  // PostgreSQL
	".*:6379",  // Redis
	".*:27017", // MongoDB
	".*:9200",  // Elasticsearch
	".*:11211", // Memcached

	// 危险协议
	"file://.*",
	"ftp://.*",
}

// URLBlacklist 表示URL黑名单
type URLBlacklist struct {
	enabled        bool
	patterns       []string
	ipNetworks     []*net.IPNet
	regexPatterns  []*regexp.Regexp
	domainPatterns []string
}

// NewURLBlacklist 创建一个新的URL黑名单
func NewURLBlacklist(opts *Options) (*URLBlacklist, error) {
	bl := &URLBlacklist{
		enabled:        opts.Scan.EnableBlacklist,
		patterns:       []string{},
		ipNetworks:     []*net.IPNet{},
		regexPatterns:  []*regexp.Regexp{},
		domainPatterns: []string{},
	}

	// 如果黑名单未启用，直接返回
	if !bl.enabled {
		return bl, nil
	}

	// 添加默认黑名单
	if opts.Scan.DefaultBlacklist {
		bl.patterns = append(bl.patterns, DefaultBlacklist...)
	}

	// 添加自定义黑名单
	if len(opts.Scan.BlacklistPatterns) > 0 {
		bl.patterns = append(bl.patterns, opts.Scan.BlacklistPatterns...)
	}

	// 从文件加载黑名单
	if opts.Scan.BlacklistFile != "" {
		patterns, err := loadPatternsFromFile(opts.Scan.BlacklistFile)
		if err != nil {
			return nil, fmt.Errorf("加载黑名单文件失败: %v", err)
		}
		bl.patterns = append(bl.patterns, patterns...)
	}

	// 解析黑名单规则
	if err := bl.parsePatterns(); err != nil {
		return nil, err
	}

	log.Info("已启用URL黑名单", "规则数量", len(bl.patterns))
	return bl, nil
}

// loadPatternsFromFile 从文件加载黑名单规则
func loadPatternsFromFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

// parsePatterns 解析黑名单规则
func (bl *URLBlacklist) parsePatterns() error {
	for _, pattern := range bl.patterns {
		// 尝试解析为CIDR
		if _, ipNet, err := net.ParseCIDR(pattern); err == nil {
			bl.ipNetworks = append(bl.ipNetworks, ipNet)
			continue
		}

		// 尝试解析为IP地址
		if ip := net.ParseIP(pattern); ip != nil {
			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil { // IPv6
				mask = net.CIDRMask(128, 128)
			}
			ipNet := &net.IPNet{
				IP:   ip,
				Mask: mask,
			}
			bl.ipNetworks = append(bl.ipNetworks, ipNet)
			continue
		}

		// 检查是否为正则表达式
		if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") ||
			strings.Contains(pattern, "[") || strings.Contains(pattern, ".*") {
			// 转换通配符为正则表达式
			regexStr := pattern
			regexStr = strings.ReplaceAll(regexStr, ".", "\\.")
			regexStr = strings.ReplaceAll(regexStr, "*", ".*")
			regexStr = strings.ReplaceAll(regexStr, "?", ".")
			regexStr = "^" + regexStr + "$"

			re, err := regexp.Compile(regexStr)
			if err != nil {
				return fmt.Errorf("无效的正则表达式 '%s': %v", pattern, err)
			}
			bl.regexPatterns = append(bl.regexPatterns, re)
			continue
		}

		// 当作域名处理
		bl.domainPatterns = append(bl.domainPatterns, pattern)
	}
	return nil
}

// IsBlacklisted 检查URL是否在黑名单中
func (bl *URLBlacklist) IsBlacklisted(targetURL string) (bool, string) {
	// 如果黑名单未启用，直接返回false
	if !bl.enabled {
		return false, ""
	}

	// 解析URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		// 如果URL无法解析，出于安全考虑，返回true
		return true, "无效的URL格式"
	}

	// 提取主机名和端口
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	hostPort := host
	if port != "" {
		hostPort = fmt.Sprintf("%s:%s", host, port)
	}

	// 检查协议
	for _, re := range bl.regexPatterns {
		if re.MatchString(targetURL) {
			return true, fmt.Sprintf("匹配正则表达式黑名单规则: %s", re.String())
		}
	}

	// 检查主机名是否为域名模式
	for _, domain := range bl.domainPatterns {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true, fmt.Sprintf("匹配域名黑名单: %s", domain)
		}
	}

	// 检查IP地址
	ip := net.ParseIP(host)
	if ip != nil {
		for _, ipNet := range bl.ipNetworks {
			if ipNet.Contains(ip) {
				return true, fmt.Sprintf("IP地址在黑名单CIDR范围内: %s", ipNet.String())
			}
		}
	} else {
		// 尝试解析主机名为IP
		ips, err := net.LookupIP(host)
		if err == nil {
			for _, resolvedIP := range ips {
				for _, ipNet := range bl.ipNetworks {
					if ipNet.Contains(resolvedIP) {
						return true, fmt.Sprintf("解析的IP地址在黑名单CIDR范围内: %s -> %s", host, resolvedIP.String())
					}
				}
			}
		}
	}

	// 检查主机名:端口
	if port != "" {
		for _, re := range bl.regexPatterns {
			if re.MatchString(hostPort) {
				return true, fmt.Sprintf("匹配端口黑名单规则: %s", re.String())
			}
		}
	}

	return false, ""
}
