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
//	results, _ := client.BatchScreenshot([]string{"https://a.com", "https://b.com"}, nil)
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

	"github.com/cyberspacesec/go-snir/pkg/log"
	"github.com/cyberspacesec/go-snir/pkg/models"
	"github.com/cyberspacesec/go-snir/pkg/runner"
)

// Client 是 go-snir 截图 SDK 的主入口
// 其他 Go 项目通过 import 此包来复用截图能力
// 内部持有 DriverPool，多个调用方共享同一个 Chrome 浏览器进程
type Client struct {
	pool *runner.DriverPool
	opts ClientOptions
}

// NewClient 创建一个新的截图客户端
// 内部初始化 Chrome 浏览器进程池，多个截图请求复用同一浏览器进程
func NewClient(opts ClientOptions) (*Client, error) {
	runnerOpts := toRunnerOptions(opts)
	pool, err := runner.NewDriverPool(&runnerOpts, opts.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("初始化截图客户端失败: %v", err)
	}

	log.Info("截图SDK客户端已创建", "max_concurrent", opts.MaxConcurrent)
	return &Client{
		pool: pool,
		opts: opts,
	}, nil
}

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

// ActiveCount 返回当前正在执行的截图数
func (c *Client) ActiveCount() int {
	return c.pool.ActiveCount()
}

// Close 关闭客户端，释放浏览器进程
// 调用后客户端不可再使用
func (c *Client) Close() {
	c.pool.Close()
	log.Info("截图SDK客户端已关闭")
}

// BatchResult 批量截图中的单个结果
type BatchResult struct {
	URL    string         `json:"url"`
	Result *models.Result `json:"result,omitempty"`
	Error  error          `json:"error,omitempty"`
}