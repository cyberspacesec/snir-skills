// Package sdk 提供 go-snir 截图能力的 Go API，供其他项目直接 import 调用
//
// 使用示例:
//
//	client, _ := sdk.NewClient(sdk.DefaultClientOptions())
//	defer client.Close()
//
//	// 基本截图
//	result, _ := client.Screenshot("https://example.com", nil)
//	fmt.Println(result.Title, result.Filename)
//
//	// 获取截图字节数据（不写磁盘）
//	imgBytes, result, _ := client.ScreenshotBytes("https://example.com", nil)
//
//	// 批量截图
//	results := client.BatchScreenshot([]string{"https://a.com", "https://b.com"}, nil)
//
//	// 流式批量截图（每完成一个立即返回）
//	ch := client.BatchScreenshotStreaming(ctx, urls, nil)
//	for result := range ch {
//	    fmt.Println(result.URL, result.Result.Title)
//	}
//
//	// 带取消的截图
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	result, _ := client.ScreenshotWithContext(ctx, "https://example.com", nil)
package sdk

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// Client 是 go-snir 截图 SDK 的主入口
// 其他 Go 项目通过 import 此包来复用截图能力
// 内部持有 DriverPool，多个调用方共享同一个 Chrome 浏览器进程
type Client struct {
	pool      *runner.DriverPool
	opts      ClientOptions
	cookieJar *runner.CookieJar // Cookie 持久化存储
}

// NewClient 创建一个新的截图客户端
// 内部初始化 Chrome 浏览器进程池，多个截图请求复用同一浏览器进程
func NewClient(opts ClientOptions) (*Client, error) {
	runnerOpts := toRunnerOptions(opts)
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("初始化截图客户端失败: %v", err)
	}

	client := &Client{
		pool: pool,
		opts: opts,
	}

	// 加载 Cookie 持久化存储
	if opts.CookieFile != "" {
		jar, err := runner.NewCookieJar(opts.CookieFile)
		if err != nil {
			log.Warn("加载 Cookie 文件失败", "file", opts.CookieFile, "error", err)
		} else {
			client.cookieJar = jar
			log.Info("Cookie 持久化存储已加载", "file", opts.CookieFile)
		}
	}

	mode := "本地"
	if opts.WSSURL != "" {
		mode = "远程"
	}
	log.Info("截图SDK客户端已创建", "mode", mode, "max_concurrent", opts.MaxConcurrent)
	return client, nil
}

// NewRemoteClient 创建一个连接到远程 Chrome 的截图客户端
// wsURL: 远程 Chrome 的 WebSocket URL（如 ws://hostname:9222/devtools/browser/xxxx）
// maxConcurrent: 最大并发截图数
// 其他选项使用默认值
func NewRemoteClient(wsURL string, maxConcurrent int) (*Client, error) {
	opts := DefaultClientOptions()
	opts.WSSURL = wsURL
	if maxConcurrent > 0 {
		opts.MaxConcurrent = maxConcurrent
	}

	runnerOpts := toRunnerOptions(opts)
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("连接远程浏览器失败: %v", err)
	}

	log.Info("远程截图SDK客户端已创建", "ws_url", wsURL, "max_concurrent", opts.MaxConcurrent)
	return &Client{
		pool: pool,
		opts: opts,
	}, nil
}

// ---------------------------------------------------------------------------
// 截图方法
// ---------------------------------------------------------------------------

// Screenshot 对指定 URL 执行截图
// url: 目标网页 URL
// screenshotOpts: 单次截图的可选配置，可覆盖客户端默认配置，传 nil 使用默认配置
// 返回截图结果，包含页面标题、截图文件路径、状态码等信息
func (c *Client) Screenshot(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	return c.ScreenshotWithContext(context.Background(), url, screenshotOpts)
}

// ScreenshotWithContext 支持取消的截图
// ctx 可用于取消长时间运行的截图任务
func (c *Client) ScreenshotWithContext(ctx context.Context, url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	runnerOpts := toRunnerOptions(c.opts)
	runnerOpts = mergeWithScreenshotOptions(runnerOpts, screenshotOpts)

	// 合并 CookieJar 中的 Cookie
	if c.cookieJar != nil {
		jarCookies := c.cookieJar.GetCookies(extractDomain(url))
		if len(jarCookies) > 0 {
			// 合并：jar Cookie 附加到 opts Cookie 之后（jar Cookie 优先级低）
			allCookies := append(jarCookies, runnerOpts.Scan.Cookies...)
			runnerOpts.Scan.Cookies = allCookies
		}
	}

	result, err := c.pool.ScreenshotWithContext(ctx, url, &runnerOpts)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	if result.Failed {
		return result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}

	return result, nil
}

// ScreenshotBytes 对指定 URL 执行截图，返回截图的原始字节数据
// 适合在内存中直接使用截图数据（如上传到 S3、写入 HTTP response 等）
// 返回 PNG/JPEG 字节数据、截图元信息、错误
func (c *Client) ScreenshotBytes(url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	return c.ScreenshotBytesWithContext(context.Background(), url, screenshotOpts)
}

// ScreenshotBytesWithContext 支持取消的截图字节数据获取
func (c *Client) ScreenshotBytesWithContext(ctx context.Context, url string, screenshotOpts *ScreenshotOptions) ([]byte, *models.Result, error) {
	runnerOpts := toRunnerOptions(c.opts)
	runnerOpts = mergeWithScreenshotOptions(runnerOpts, screenshotOpts)

	// 截图保存到临时目录，然后读取字节
	result, err := c.pool.ScreenshotWithContext(ctx, url, &runnerOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("截图失败: %v", err)
	}

	if result.Failed {
		return nil, result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}

	if result.Screenshot == "" {
		return nil, result, fmt.Errorf("截图文件路径为空")
	}

	// 读取截图文件字节
	data, err := os.ReadFile(result.Screenshot)
	if err != nil {
		return nil, result, fmt.Errorf("读取截图文件失败: %v", err)
	}

	return data, result, nil
}

// ScreenshotHTML 截图并同时获取页面 HTML 源码
// 等价于设置 SaveHTML=true 后截图，便捷方法
func (c *Client) ScreenshotHTML(url string, screenshotOpts *ScreenshotOptions) (string, *models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.SaveHTML = true

	result, err := c.Screenshot(url, opts)
	if err != nil {
		return "", nil, err
	}
	return result.HTML, result, nil
}

// ScreenshotWithActions 截图前执行交互动作序列
// 适用于需要点击按钮、填写表单、滚动页面等交互后再截图的场景
//
// 示例:
//
//	actions := []runner.InteractionAction{
//	    {Type: "type", Selector: "#search", Value: "go-snir"},
//	    {Type: "click", Selector: "#search-btn"},
//	    {Type: "wait", WaitTime: 2},
//	}
//	result, _ := client.ScreenshotWithActions("https://example.com", actions, nil)
func (c *Client) ScreenshotWithActions(url string, actions []runner.InteractionAction, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Actions = actions
	return c.Screenshot(url, opts)
}

// ScreenshotWithForm 截图前填写并提交表单
// 适用于登录页面、搜索框等需要表单交互的场景
//
// 示例:
//
//	form := runner.Form{
//	    Fields: []runner.FormField{
//	        {Selector: "#username", Value: "admin"},
//	        {Selector: "#password", Value: "pass123"},
//	    },
//	    SubmitSelector: "#login-btn",
//	    WaitAfterSubmit: 3,
//	}
//	result, _ := client.ScreenshotWithForm("https://example.com/login", form, nil)
func (c *Client) ScreenshotWithForm(url string, form runner.Form, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Form = form
	return c.Screenshot(url, opts)
}

// ScreenshotWithCookies 截图前注入自定义 Cookie
// 适用于需要认证状态的页面
//
// 示例:
//
//	cookies := []runner.CustomCookie{
//	    {Name: "session", Value: "abc123", Domain: "example.com"},
//	}
//	result, _ := client.ScreenshotWithCookies("https://example.com/dashboard", cookies, nil)
func (c *Client) ScreenshotWithCookies(url string, cookies []runner.CustomCookie, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Cookies = cookies
	return c.Screenshot(url, opts)
}

// ScreenshotElement 截取指定 CSS 选择器匹配的元素
// 便捷方法，等价于设置 Selector 后截图
func (c *Client) ScreenshotElement(url string, selector string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.Selector = selector
	return c.Screenshot(url, opts)
}

// ScreenshotFullPage 截取完整页面（含滚动区域）
// 便捷方法，等价于设置 CaptureFullPage=true 后截图
func (c *Client) ScreenshotFullPage(url string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.CaptureFullPage = true
	return c.Screenshot(url, opts)
}

// ScreenshotWithJS 截图前执行 JavaScript
// 便捷方法，适用于需要操作 DOM 后截图的场景
func (c *Client) ScreenshotWithJS(url string, js string, screenshotOpts *ScreenshotOptions) (*models.Result, error) {
	opts := c.ensureScreenshotOptions(screenshotOpts)
	opts.JavaScript = js
	return c.Screenshot(url, opts)
}

// ---------------------------------------------------------------------------
// 批量截图
// ---------------------------------------------------------------------------

// BatchScreenshot 批量截图，并发执行
// urls: 要截图的 URL 列表
// screenshotOpts: 所有 URL 共享的截图配置，传 nil 使用默认配置
// 返回每个 URL 的截图结果，失败的结果也会包含在列表中（检查 Error 字段）
func (c *Client) BatchScreenshot(urls []string, screenshotOpts *ScreenshotOptions) []BatchResult {
	return c.BatchScreenshotWithContext(context.Background(), urls, screenshotOpts)
}

// BatchScreenshotWithContext 支持取消的批量截图
func (c *Client) BatchScreenshotWithContext(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions) []BatchResult {
	results := make([]BatchResult, len(urls))
	var wg sync.WaitGroup

	for i, url := range urls {
		wg.Add(1)
		go func(idx int, target string) {
			defer wg.Done()

			result, err := c.ScreenshotWithContext(ctx, target, screenshotOpts)
			results[idx] = BatchResult{
				URL:    target,
				Result: result,
				Error:  err,
			}
		}(i, url)
	}

	wg.Wait()
	return results
}

// BatchScreenshotStreaming 流式批量截图
// 每完成一个截图立即通过 channel 返回，不用等全部完成
// 适用于大量 URL 截图、进度展示、实时处理等场景
//
// 示例:
//
//	ch := client.BatchScreenshotStreaming(ctx, urls, nil)
//	for result := range ch {
//	    if result.Error != nil {
//	        log.Printf("失败: %s - %v", result.URL, result.Error)
//	    } else {
//	        log.Printf("完成: %s - %s", result.URL, result.Result.Title)
//	    }
//	}
func (c *Client) BatchScreenshotStreaming(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions) <-chan BatchResult {
	ch := make(chan BatchResult, len(urls))

	go func() {
		defer close(ch)

		var wg sync.WaitGroup
		for _, url := range urls {
			// 检查 context 是否已取消
			select {
			case <-ctx.Done():
				ch <- BatchResult{URL: url, Error: ctx.Err()}
				continue
			default:
			}

			wg.Add(1)
			go func(target string) {
				defer wg.Done()

				result, err := c.ScreenshotWithContext(ctx, target, screenshotOpts)
				ch <- BatchResult{
					URL:    target,
					Result: result,
					Error:  err,
				}
			}(url)
		}

		wg.Wait()
	}()

	return ch
}

// BatchScreenshotCallback 批量截图，每完成一个调用回调函数
// 适用于需要逐个处理结果的场景
// callback 在截图完成时同步调用，可用于进度展示、结果处理等
func (c *Client) BatchScreenshotCallback(ctx context.Context, urls []string, screenshotOpts *ScreenshotOptions, callback func(BatchResult)) {
	ch := c.BatchScreenshotStreaming(ctx, urls, screenshotOpts)
	for result := range ch {
		if callback != nil {
			callback(result)
		}
	}
}

// ---------------------------------------------------------------------------
// 池管理
// ---------------------------------------------------------------------------

// Stats 返回连接池统计信息
func (c *Client) Stats() runner.PoolStats {
	return c.pool.Stats()
}

// SetIdleTimeout 设置空闲超时
// 当客户端空闲超过此时间后，自动关闭浏览器进程释放资源
// 下次截图时会自动重启浏览器进程
// 设为 0 表示不自动关闭（默认行为）
func (c *Client) SetIdleTimeout(timeout time.Duration) {
	c.pool.SetIdleTimeout(timeout)
}

// OnEvent 注册池事件监听器
// 事件类型:
//   - screenshot_start: 截图开始
//   - screenshot_complete: 截图完成（含耗时和结果）
//   - screenshot_failed: 截图失败（含错误信息）
//   - reconnect: 浏览器进程重新连接
//   - idle_close: 空闲超时关闭浏览器
//   - pool_closed: 连接池关闭
//
// 回调是异步执行的，不会阻塞截图流程
func (c *Client) OnEvent(handler runner.PoolEventHandler) {
	c.pool.On(handler)
}

// ActiveCount 返回当前正在执行的截图数
func (c *Client) ActiveCount() int {
	return c.pool.ActiveCount()
}

// SetCookieJar 设置 Cookie 持久化存储
// 设置后，截图时自动从 CookieJar 加载对应域名的 Cookie
// 一次性 Cookie 获取后自动移除，持久化 Cookie 保留
func (c *Client) SetCookieJar(jar *runner.CookieJar) {
	c.cookieJar = jar
}

// CookieJar 返回当前的 Cookie 持久化存储
func (c *Client) CookieJar() *runner.CookieJar {
	return c.cookieJar
}

// AddCookie 添加 Cookie 到持久化存储
// 如果没有 CookieJar，会自动创建一个内存中的 CookieJar
func (c *Client) AddCookie(cookie runner.PersistentCookie) error {
	if c.cookieJar == nil {
		jar, err := runner.NewCookieJar("")
		if err != nil {
			return err
		}
		c.cookieJar = jar
	}
	return c.cookieJar.AddCookie(cookie)
}

// AddPersistentCookie 添加持久化 Cookie
func (c *Client) AddPersistentCookie(name, value, domain string) error {
	return c.AddCookie(runner.PersistentCookie{
		Name:       name,
		Value:      value,
		Domain:     domain,
		Persistent: true,
	})
}

// AddOneTimeCookie 添加一次性 Cookie（获取后自动移除）
func (c *Client) AddOneTimeCookie(name, value, domain string) error {
	return c.AddCookie(runner.PersistentCookie{
		Name:       name,
		Value:      value,
		Domain:     domain,
		Persistent: false,
	})
}

// Close 关闭客户端，释放浏览器进程
// 调用后客户端不可再使用
func (c *Client) Close() {
	c.pool.Close()
	log.Info("截图SDK客户端已关闭")
}

// ---------------------------------------------------------------------------
// 类型定义
// ---------------------------------------------------------------------------

// BatchResult 批量截图中的单个结果
type BatchResult struct {
	URL    string         `json:"url"`
	Result *models.Result `json:"result,omitempty"`
	Error  error          `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// 内部辅助
// ---------------------------------------------------------------------------

// ensureScreenshotOptions 确保 ScreenshotOptions 不为 nil
// 如果传入 nil，返回一个新的零值 ScreenshotOptions
func (c *Client) ensureScreenshotOptions(opts *ScreenshotOptions) *ScreenshotOptions {
	if opts != nil {
		return opts
	}
	return &ScreenshotOptions{}
}

// extractDomain 从 URL 中提取域名
func extractDomain(rawURL string) string {
	// 简单提取：去掉协议和路径
	u := rawURL
	// 去掉协议
	for _, prefix := range []string{"http://", "https://"} {
		if len(u) > len(prefix) && u[:len(prefix)] == prefix {
			u = u[len(prefix):]
			break
		}
	}
	// 去掉路径
	for i, c := range u {
		if c == '/' || c == ':' || c == '?' || c == '#' {
			u = u[:i]
			break
		}
	}
	return u
}
