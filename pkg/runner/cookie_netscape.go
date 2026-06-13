package runner

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// LoadNetscapeCookieFile 加载 Netscape/Mozilla 格式的 Cookie 文件
// 这是 curl --cookie-jar、wget 和浏览器扩展导出的标准格式
//
// 文件格式：
//
//	# HTTP Cookie File
//	.example.com	TRUE	/	FALSE	0	session	abc123
//	.example.com	TRUE	/	TRUE	1735689600	auth_token	xyz789
//
// 每行字段（制表符分隔）：
//  1. domain    - 域名（以 . 开头表示包含子域名）
//  2. incl_sub  - TRUE/FALSE 是否包含子域名
//  3. path      - Cookie 路径
//  4. secure    - TRUE/FALSE 是否仅 HTTPS
//  5. expires   - 过期时间（Unix 时间戳，0=会话 Cookie）
//  6. name      - Cookie 名称
//  7. value     - Cookie 值
func LoadNetscapeCookieFile(filePath string) ([]CustomCookie, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 Cookie 文件失败: %v", err)
	}
	defer file.Close()

	var cookies []CustomCookie
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析字段（制表符分隔）
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			// 尝试空格分隔（某些工具使用空格而非制表符）
			fields = strings.Fields(line)
			if len(fields) < 7 {
				log.Debug("跳过格式不正确的 Cookie 行", "line", lineNum)
				continue
			}
		}

		cookie := CustomCookie{
			Name:  fields[5],
			Value: fields[6],
		}

		// 域名
		domain := fields[0]
		if domain != "" {
			cookie.Domain = domain
		}

		// 路径
		if len(fields) > 2 && fields[2] != "" {
			cookie.Path = fields[2]
		}

		// Secure
		if len(fields) > 3 {
			cookie.Secure = strings.ToUpper(fields[3]) == "TRUE"
		}

		// HttpOnly 没有在 Netscape 格式中直接表示
		// 但某些扩展版本在第 8 列添加了 httpOnly 标记
		if len(fields) > 7 {
			cookie.HttpOnly = strings.ToUpper(fields[7]) == "TRUE" ||
				strings.ToLower(fields[7]) == "httponly"
		}

		cookies = append(cookies, cookie)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 Cookie 文件失败: %v", err)
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("Cookie 文件为空或格式不正确: %s", filePath)
	}

	return cookies, nil
}

// LoadNetscapeCookieFileToJar 加载 Netscape 格式 Cookie 文件到 CookieJar
// persistent: 是否将 Cookie 标记为持久化
// source: 来源标记
func LoadNetscapeCookieFileToJar(filePath string, persistent bool, source string) (*CookieJar, []PersistentCookie, error) {
	// 先解析 Netscape 格式
	// 解析 Netscape 格式（需要读取 expires 信息）
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	// 解析为 PersistentCookie（保留 expires 信息）
	var persistentCookies []PersistentCookie
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			fields = strings.Fields(line)
			if len(fields) < 7 {
				continue
			}
		}

		// 解析 expires
		var expiresAt int64
		if len(fields) > 4 {
			expires, _ := strconv.ParseInt(fields[4], 10, 64)
			if expires > 0 {
				expiresAt = expires
			}
		}

		// 会话 Cookie（expires=0）也是有效的
		isSessionCookie := len(fields) > 4 && fields[4] == "0"

		pc := PersistentCookie{
			Name:       fields[5],
			Value:      fields[6],
			Domain:     fields[0],
			Path:       fields[2],
			Secure:     strings.ToUpper(fields[3]) == "TRUE",
			Persistent: persistent,
			ExpiresAt:  expiresAt,
			Source:     source,
		}

		if len(fields) > 7 {
			pc.HttpOnly = strings.ToUpper(fields[7]) == "TRUE" ||
				strings.ToLower(fields[7]) == "httponly"
		}

		// 会话 Cookie 不设过期时间
		if isSessionCookie {
			pc.ExpiresAt = 0
		}

		// 跳过已过期的 Cookie
		if pc.IsExpired() {
			continue
		}

		persistentCookies = append(persistentCookies, pc)
	}

	// 创建内存 CookieJar
	jar, err := NewCookieJar("")
	if err != nil {
		return nil, nil, err
	}

	for _, pc := range persistentCookies {
		jar.AddCookie(pc)
	}

	log.Info("从 Netscape Cookie 文件加载", "file", filePath, "cookies", len(persistentCookies))

	return jar, persistentCookies, nil
}

// SaveNetscapeCookieFile 将 Cookie 保存为 Netscape 格式
// 兼容 curl --cookie-jar 输出格式
func SaveNetscapeCookieFile(filePath string, cookies []PersistentCookie) error {
	var lines []string
	lines = append(lines, "# Netscape HTTP Cookie File")
	lines = append(lines, "# This is a generated file!  Do not edit.")
	lines = append(lines, "")

	for _, c := range cookies {
		if c.IsExpired() {
			continue
		}

		domain := c.Domain
		if domain == "" {
			domain = "_global"
		}

		// 判断域名是否以 . 开头（包含子域名）
		inclSub := "FALSE"
		if strings.HasPrefix(domain, ".") {
			inclSub = "TRUE"
		}

		path := c.Path
		if path == "" {
			path = "/"
		}

		secure := "FALSE"
		if c.Secure {
			secure = "TRUE"
		}

		expires := "0"
		if c.ExpiresAt > 0 {
			expires = strconv.FormatInt(c.ExpiresAt, 10)
		}

		httpOnly := "FALSE"
		if c.HttpOnly {
			httpOnly = "TRUE"
		}

		// Netscape 格式：domain \t incl_sub \t path \t secure \t expires \t name \t value [\t httpOnly]
		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			domain, inclSub, path, secure, expires, c.Name, c.Value, httpOnly)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(filePath, []byte(content), 0644)
}

// ExportResultCookiesToNetscape 将截图结果的 Cookie 导出为 Netscape 格式
func ExportResultCookiesToNetscape(filePath string, resultCookies []models.Cookie, url string) error {
	var cookies []PersistentCookie
	domain := extractDomainSimple(url)

	for _, c := range resultCookies {
		cookieDomain := c.Domain
		if cookieDomain == "" {
			cookieDomain = domain
		}
		cookies = append(cookies, PersistentCookie{
			Name:       c.Name,
			Value:      c.Value,
			Domain:     cookieDomain,
			Path:       c.Path,
			Persistent: true,
			Source:     "export",
		})
	}

	return SaveNetscapeCookieFile(filePath, cookies)
}
