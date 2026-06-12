// Package sdk 提供 go-snir 截图能力的 Go API，供其他项目直接 import 调用
//
// 使用示例:
//
//	client, _ := sdk.NewClient(sdk.DefaultClientOptions())
//	defer client.Close()
//	result, _ := client.Screenshot("https://example.com", nil)
//	fmt.Println(result.Title, result.Filename)
package sdk

import (
	"fmt"

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
	runnerOpts := toRunnerOptions(c.opts)
	runnerOpts = mergeWithScreenshotOptions(runnerOpts, screenshotOpts)

	result, err := c.pool.Screenshot(url, &runnerOpts)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	if result.Failed {
		return result, fmt.Errorf("截图失败: %s", result.FailedReason)
	}

	return result, nil
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