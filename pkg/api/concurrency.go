package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
)

// ConcurrencyLimiter 用于控制并发请求数
type ConcurrencyLimiter struct {
	maxConcurrent int           // 最大并发数
	semaphore     chan struct{} // 信号量通道
	waitQueue     int           // 等待队列长度
	activeCount   int           // 当前活跃请求数
	waitCount     int           // 当前等待请求数
	mu            sync.Mutex    // 互斥锁，保护计数器
}

// NewConcurrencyLimiter 创建一个新的并发限制器
func NewConcurrencyLimiter(maxConcurrent, waitQueue int) *ConcurrencyLimiter {
	// 确保参数有效
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // 默认值
		log.Warn("设置了无效的最大并发数，使用默认值", "default", maxConcurrent)
	}
	if waitQueue <= 0 {
		waitQueue = 100 // 默认值
		log.Warn("设置了无效的队列大小，使用默认值", "default", waitQueue)
	}

	return &ConcurrencyLimiter{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
		waitQueue:     waitQueue,
	}
}

// NewBasicConcurrencyLimiter 初始化基本并发限制器
func NewBasicConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	if maxConcurrent <= 0 {
		return nil
	}

	return &ConcurrencyLimiter{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
		waitQueue:     100, // 默认等待队列大小
	}
}

// Acquire 尝试获取许可
func (cl *ConcurrencyLimiter) Acquire(ctx context.Context) error {
	// 如果使用了等待队列功能
	if cl.waitQueue > 0 {
		cl.mu.Lock()
		// 如果等待队列已满，直接拒绝
		if cl.waitCount >= cl.waitQueue {
			cl.mu.Unlock()
			return fmt.Errorf("服务器繁忙，请求队列已满")
		}

		cl.waitCount++
		cl.mu.Unlock()

		// 尝试获取信号量
		select {
		case cl.semaphore <- struct{}{}: // 获取到信号量
			cl.mu.Lock()
			cl.waitCount--
			cl.activeCount++
			cl.mu.Unlock()
			return nil
		case <-ctx.Done(): // 请求被取消或超时
			cl.mu.Lock()
			cl.waitCount--
			cl.mu.Unlock()
			return ctx.Err()
		}
	} else {
		// 简化版本，不使用等待队列计数
		select {
		case cl.semaphore <- struct{}{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Release 释放许可
func (cl *ConcurrencyLimiter) Release() {
	// 如果使用了活跃计数
	if cl.waitQueue > 0 {
		cl.mu.Lock()
		if cl.activeCount > 0 {
			cl.activeCount--
			<-cl.semaphore // 释放信号量
		}
		cl.mu.Unlock()
	} else {
		// 简化版本
		select {
		case <-cl.semaphore:
		default:
			// 防止重复释放导致的问题
		}
	}
}

// Stats 获取当前状态
func (cl *ConcurrencyLimiter) Stats() (active, waiting, maxConcurrent, queueSize int) {
	cl.mu.Lock()
	active = cl.activeCount
	waiting = cl.waitCount
	maxConcurrent = cl.maxConcurrent
	queueSize = cl.waitQueue
	cl.mu.Unlock()
	return
}

// CurrentConcurrency 获取当前并发数
func (cl *ConcurrencyLimiter) CurrentConcurrency() int {
	return len(cl.semaphore)
}

// MaxConcurrency 获取最大并发数
func (cl *ConcurrencyLimiter) MaxConcurrency() int {
	return cl.maxConcurrent
}

// ProcessConcurrent 使用指定的并发数处理请求
func ProcessConcurrent(requests []ScreenshotRequest, concurrency int, processor func(ScreenshotRequest) BatchResult, results chan<- BatchResult) {
	// 创建工作池
	limiter := make(chan struct{}, concurrency)

	// 创建等待组
	var wg sync.WaitGroup

	// 处理每个请求
	for _, req := range requests {
		wg.Add(1)
		go func(request ScreenshotRequest) {
			defer wg.Done()

			// 获取信号量
			limiter <- struct{}{}
			defer func() { <-limiter }()

			// 处理请求
			result := processor(request)

			// 发送结果
			results <- result
		}(req)
	}

	// 等待所有请求完成
	wg.Wait()
}

// CreateConcurrencyLimitMiddleware 创建并发限制中间件
func (s *Server) CreateConcurrencyLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// 确保启用了并发限制
		if s.concurrencyLimit == nil {
			return next
		}

		limiter, ok := s.concurrencyLimit.(*ConcurrencyLimiter)
		if !ok {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过对状态检查和静态资源的限制
			if r.URL.Path == "/health" || r.URL.Path == "/stats" ||
				r.URL.Path == "/" || r.URL.Path == "/favicon.ico" ||
				strings.HasPrefix(r.URL.Path, "/screenshots/") {
				next.ServeHTTP(w, r)
				return
			}

			// 设置超时上下文
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			// 尝试获取许可
			err := limiter.Acquire(ctx)
			if err != nil {
				if err == context.DeadlineExceeded {
					log.Warn("请求等待超时", "path", r.URL.Path, "method", r.Method)
					SendJSONResponse(w, http.StatusServiceUnavailable, APIResponse{
						Success: false,
						Error:   "服务器繁忙，请稍后重试",
					})
				} else {
					log.Warn("请求被拒绝，队列已满", "path", r.URL.Path, "method", r.Method)
					SendJSONResponse(w, http.StatusTooManyRequests, APIResponse{
						Success: false,
						Error:   "服务器繁忙，请求队列已满，请稍后重试",
					})
				}
				return
			}

			// 请求完成后释放许可
			defer limiter.Release()

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}
