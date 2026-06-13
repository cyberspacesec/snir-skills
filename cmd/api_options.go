package cmd

import (
	"github.com/cyberspacesec/snir-skills/pkg/api"
)

// 应用单个截图请求的配置选项
func applyScreenshotOptions(req *api.ScreenshotRequest) {
	// 应用HTTP/HTTPS选项
	if req.HTTP {
		opts.Scan.HTTP = true
	}
	if req.HTTPS {
		opts.Scan.HTTPS = true
	}

	// 应用Chrome相关选项
	if req.UserAgent != "" {
		opts.Chrome.UserAgent = req.UserAgent
	}
	if req.Proxy != "" {
		opts.Chrome.Proxy = req.Proxy
	}
	if req.Timeout > 0 {
		opts.Chrome.Timeout = req.Timeout
	}
	if req.Delay > 0 {
		opts.Chrome.Delay = req.Delay
	}
}

// 应用批量截图请求的配置选项
func applyBatchScreenshotOptions(req *api.BatchScreenshotRequest) {
	// 应用HTTP/HTTPS选项
	if req.HTTP {
		opts.Scan.HTTP = true
	}
	if req.HTTPS {
		opts.Scan.HTTPS = true
	}

	// 应用Chrome相关选项
	if req.UserAgent != "" {
		opts.Chrome.UserAgent = req.UserAgent
	}
	if req.Proxy != "" {
		opts.Chrome.Proxy = req.Proxy
	}
	if req.Timeout > 0 {
		opts.Chrome.Timeout = req.Timeout
	}
	if req.Delay > 0 {
		opts.Chrome.Delay = req.Delay
	}

	// 应用并发线程数
	if req.Threads > 0 {
		opts.Scan.Threads = req.Threads
	}
}
