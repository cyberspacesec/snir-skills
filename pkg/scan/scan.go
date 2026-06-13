package scan

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/models"
	"github.com/cyberspacesec/go-snir/pkg/runner"
)

// Config 表示扫描配置
// 包含扫描目标和选项信息
type Config struct {
	Target     string          // 单个扫描目标
	TargetFile string          // 包含目标的文件
	Targets    []string        // 目标列表
	Options    *runner.Options // 扫描选项
	UsePool    bool            // 是否使用连接池（复用 Chrome 进程）
}

// Scanner 表示扫描器
// 负责执行网站扫描和截图操作
type Scanner struct {
	Config    *Config            // 扫描配置
	Driver    runner.Driver      // 浏览器驱动
	Writers   []runner.Writer    // 结果写入器
	Runner    *runner.Runner     // 运行器
	CookieJar *runner.CookieJar  // Cookie 持久化存储
}

// NewScanner 创建一个新的扫描器
// 初始化驱动和写入器，准备扫描环境
// 参数:
//   - config: 扫描配置
//
// 返回:
//   - 初始化的扫描器和可能的错误
func NewScanner(config *Config) (*Scanner, error) {
	// 验证配置
	if config == nil || config.Options == nil {
		return nil, fmt.Errorf("扫描配置不能为空")
	}

	// 创建驱动（根据配置选择连接池或单次模式）
	driver, err := createDriver(config.Options, config.UsePool)
	if err != nil {
		return nil, fmt.Errorf("创建浏览器驱动失败: %v", err)
	}

	// 创建结果写入器
	writers, err := createWriters(config.Options)
	if err != nil {
		return nil, fmt.Errorf("创建结果写入器失败: %v", err)
	}

	scanner := &Scanner{
		Config:  config,
		Driver:  driver,
		Writers: writers,
	}

	// 加载 Cookie 持久化存储
	if config.Options.Scan.CookiesFile != "" {
		jar, err := runner.NewCookieJar(config.Options.Scan.CookiesFile)
		if err != nil {
			log.Warn("加载 Cookie 文件失败", "file", config.Options.Scan.CookiesFile, "error", err)
		} else {
			scanner.CookieJar = jar
			log.Info("Cookie 持久化存储已加载", "file", config.Options.Scan.CookiesFile)
		}
	}

	// 导入 Netscape 格式 Cookie
	if config.Options.Scan.CookieImport != "" {
		_, imported, err := runner.LoadNetscapeCookieFileToJar(config.Options.Scan.CookieImport, true, "import")
		if err != nil {
			log.Warn("导入 Netscape Cookie 文件失败", "file", config.Options.Scan.CookieImport, "error", err)
		} else {
			// 将导入的 Cookie 合并到 opts
			for _, pc := range imported {
				config.Options.Scan.Cookies = append(config.Options.Scan.Cookies, pc.ToCustomCookie())
			}
			log.Info("Netscape Cookie 已导入", "file", config.Options.Scan.CookieImport, "count", len(imported))
		}
	}

	// 解析内联 Cookie 字符串
	if len(config.Options.Scan.CookieStrings) > 0 {
		for _, cs := range config.Options.Scan.CookieStrings {
		parsed := runner.ParseCookieHeader(cs, "")
		config.Options.Scan.Cookies = append(config.Options.Scan.Cookies, parsed...)
		}
		log.Info("内联 Cookie 已解析", "count", len(config.Options.Scan.CookieStrings))
	}

	return scanner, nil
}

// NewPooledScanner 创建一个使用连接池的扫描器
// 复用 Chrome 进程，适合批量截图场景
// maxConcurrent: 最大并发截图数
func NewPooledScanner(config *Config, maxConcurrent int) (*Scanner, error) {
	if config == nil || config.Options == nil {
		return nil, fmt.Errorf("扫描配置不能为空")
	}

	poolDriver, err := runner.NewPoolDriver(config.Options, maxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("创建连接池驱动失败: %v", err)
	}

	writers, err := createWriters(config.Options)
	if err != nil {
		poolDriver.Close()
		return nil, fmt.Errorf("创建结果写入器失败: %v", err)
	}

	scanner := &Scanner{
		Config:  config,
		Driver:  poolDriver,
		Writers: writers,
	}

	// 加载 Cookie 持久化存储
	if config.Options.Scan.CookiesFile != "" {
		jar, err := runner.NewCookieJar(config.Options.Scan.CookiesFile)
		if err != nil {
			log.Warn("加载 Cookie 文件失败", "file", config.Options.Scan.CookiesFile, "error", err)
		} else {
			scanner.CookieJar = jar
			log.Info("Cookie 持久化存储已加载", "file", config.Options.Scan.CookiesFile)
		}
	}

	return scanner, nil
}

// createDriver 创建浏览器驱动
// 根据 usePool 配置选择使用连接池或单次模式
func createDriver(options *runner.Options, usePool bool) (runner.Driver, error) {
	if usePool {
		// 使用连接池，并发数等于 Threads
		maxConcurrent := options.Scan.Threads
		if maxConcurrent <= 0 {
			maxConcurrent = 2
		}
		return runner.NewPoolDriver(options, maxConcurrent)
	}
	// 使用单次 ChromeDP
	return runner.NewChromeDP(options)
}

// createWriters 创建结果写入器
// 根据配置创建适当的结果输出写入器
// 参数:
//   - options: 写入器选项
//
// 返回:
//   - 写入器列表和可能的错误
func createWriters(options *runner.Options) ([]runner.Writer, error) {
	return runner.CreateWriters(options)
}

// ensureProtocol 确保URL包含协议前缀
// 根据配置为URL添加适当的协议前缀
// 参数:
//   - target: 目标URL
//   - useHTTPS: 是否使用HTTPS
//   - useHTTP: 是否使用HTTP
//
// 返回:
//   - 带有协议前缀的URL
func ensureProtocol(target string, useHTTPS, useHTTP bool) string {
	// 已有协议前缀则直接返回
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}

	// 根据配置添加协议前缀
	if useHTTPS {
		return "https://" + target
	} else if useHTTP {
		return "http://" + target
	}

	// 默认使用HTTPS
	return "https://" + target
}

// extractDomainFromURL 从 URL 中提取域名
func extractDomainFromURL(rawURL string) string {
	u := rawURL
	for _, prefix := range []string{"http://", "https://"} {
		if len(u) > len(prefix) && u[:len(prefix)] == prefix {
			u = u[len(prefix):]
			break
		}
	}
	for i, c := range u {
		if c == '/' || c == ':' || c == '?' || c == '#' {
			return u[:i]
		}
	}
	return u
}

// ScanSingle 扫描单个URL
// 对单个URL执行截图和信息收集
// 参数:
//   - target: 目标URL
//
// 返回:
//   - 扫描结果和可能的错误
func (s *Scanner) ScanSingle(target string) (*models.Result, error) {
	// 确保URL格式正确
	target = ensureProtocol(target, s.Config.Options.Scan.HTTPS, s.Config.Options.Scan.HTTP)

	// 验证URL格式
	_, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("无效的URL: %v", err)
	}

	log.Info("开始扫描单个URL", "url", target)

	// 从 CookieJar 注入 Cookie
	if s.CookieJar != nil {
		domain := extractDomainFromURL(target)
		jarCookies := s.CookieJar.GetCookies(domain)
		if len(jarCookies) > 0 {
			s.Config.Options.Scan.Cookies = append(jarCookies, s.Config.Options.Scan.Cookies...)
		}
	}

	// 创建Runner（如果尚未创建）
	if s.Runner == nil {
		runner, err := runner.NewRunner(log.GetLogger(), s.Driver, *s.Config.Options, s.Writers)
		if err != nil {
			return nil, fmt.Errorf("创建扫描运行器失败: %v", err)
		}
		s.Runner = runner
	}

	// 尝试执行扫描，最多重试指定次数
	var result *models.Result
	var lastErr error
	maxRetries := s.Config.Options.Scan.MaxRetries

	// 至少尝试一次
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Info(fmt.Sprintf("第 %d 次重试扫描", attempt), "url", target)
			// 重试前等待一小段时间，逐渐增加等待时间
			time.Sleep(time.Duration(2*attempt) * time.Second)
		}

		// 执行扫描
		result, lastErr = s.Driver.Witness(target, s.Config.Options)

		// 如果成功或者是特定类型的错误不应重试，则跳出循环
		if lastErr == nil ||
			(strings.Contains(lastErr.Error(), "net::ERR_NAME_NOT_RESOLVED") ||
				strings.Contains(lastErr.Error(), "net::ERR_CONNECTION_REFUSED")) {
			break
		}
	}

	// 如果所有尝试都失败，返回最后一个错误
	if lastErr != nil {
		return nil, fmt.Errorf("扫描失败: %v", lastErr)
	}

	// 运行写入器
	s.writeResult(result)

	return result, nil
}

// writeResult 写入扫描结果
// 将结果传递给所有写入器
// 参数:
//   - result: 扫描结果
func (s *Scanner) writeResult(result *models.Result) {
	for _, writer := range s.Writers {
		if err := writer.Write(result); err != nil {
			log.Error("写入结果失败", "error", err)
		}
	}
}

// ScanMulti 扫描多个URL
// 并发处理多个URL的扫描
// 参数:
//   - targets: 目标URL列表
//
// 返回:
//   - 可能的错误
func (s *Scanner) ScanMulti(targets []string) error {
	// 创建Runner（如果尚未创建）
	if s.Runner == nil {
		runner, err := runner.NewRunner(log.GetLogger(), s.Driver, *s.Config.Options, s.Writers)
		if err != nil {
			return fmt.Errorf("创建扫描运行器失败: %v", err)
		}
		s.Runner = runner
	}

	// 启动扫描，向通道发送目标
	go func() {
		for _, target := range targets {
			// 确保URL格式正确
			formattedTarget := ensureProtocol(target, s.Config.Options.Scan.HTTPS, s.Config.Options.Scan.HTTP)
			s.Runner.Targets <- formattedTarget
		}
		close(s.Runner.Targets)
	}()

	// 执行扫描
	return s.Runner.Run()
}

// Close 关闭扫描器
// 释放资源，关闭驱动和写入器
// 返回:
//   - 可能的错误
func (s *Scanner) Close() error {
	var err error
	// 关闭Runner
	if s.Runner != nil {
		err = s.Runner.Close()
	} else {
		// 关闭驱动
		s.Driver.Close()

		// 关闭写入器
		for _, writer := range s.Writers {
			if writerErr := writer.Close(); writerErr != nil {
				log.Error("关闭写入器失败", "error", writerErr)
				if err == nil {
					err = writerErr
				}
			}
		}
	}
	return err
}
