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

// 全局并发限制器
var (
	limiterMu          sync.Mutex
	activeRequests     int           // 当前活跃请求数
	waitingRequests    int           // 等待中的请求数
	maxConcurrent      = 10          // 默认最大并发数
	maxQueueSize       = 100         // 默认等待队列大小
	concurrencySemaCh  chan struct{} // 信号量通道
	limiterInitialized bool
	startTime          = time.Now()
)

// InitConcurrencyLimiter 初始化全局并发限制器
func InitConcurrencyLimiter(max, queueSize int) {
	limiterMu.Lock()
	defer limiterMu.Unlock()

	if limiterInitialized {
		return
	}

	if max <= 0 {
		max = 10
	}

	if queueSize <= 0 {
		queueSize = 100
	}

	maxConcurrent = max
	maxQueueSize = queueSize
	concurrencySemaCh = make(chan struct{}, max)
	limiterInitialized = true

	log.Info("初始化并发限制器", "max_concurrent", max, "queue_size", queueSize)
}

// ResetConcurrencyLimiter 重置全局并发限制器状态，仅用于测试。
// 由于 InitConcurrencyLimiter 是幂等的（首次初始化后忽略后续调用），
// 测试需要精确控制并发上限时必须先调用本函数清空旧状态。
func ResetConcurrencyLimiter() {
	limiterMu.Lock()
	defer limiterMu.Unlock()

	maxConcurrent = 10
	maxQueueSize = 100
	activeRequests = 0
	waitingRequests = 0
	concurrencySemaCh = nil
	limiterInitialized = false
}

// AcquireConcurrencyPermit 尝试获取并发许可
func AcquireConcurrencyPermit(ctx context.Context) error {
	if !limiterInitialized {
		return nil // 未初始化，直接通过
	}

	limiterMu.Lock()
	// 检查等待队列是否已满
	if waitingRequests >= maxQueueSize {
		limiterMu.Unlock()
		return fmt.Errorf("服务器繁忙，请求队列已满")
	}

	waitingRequests++
	limiterMu.Unlock()

	// 尝试获取信号量
	select {
	case concurrencySemaCh <- struct{}{}:
		limiterMu.Lock()
		waitingRequests--
		activeRequests++
		limiterMu.Unlock()
		return nil
	case <-ctx.Done():
		limiterMu.Lock()
		waitingRequests--
		limiterMu.Unlock()
		return ctx.Err()
	}
}

// ReleaseConcurrencyPermit 释放并发许可
func ReleaseConcurrencyPermit() {
	if !limiterInitialized {
		return
	}

	limiterMu.Lock()
	if activeRequests > 0 {
		activeRequests--
		<-concurrencySemaCh
	}
	limiterMu.Unlock()
}

// GetConcurrencyStats 获取并发限制器状态
func GetConcurrencyStats() (active, waiting, max, queue int, uptime time.Duration) {
	limiterMu.Lock()
	active = activeRequests
	waiting = waitingRequests
	max = maxConcurrent
	queue = maxQueueSize
	limiterMu.Unlock()
	uptime = time.Since(startTime)
	return
}

// CreateConcurrencyLimitMiddleware 创建并发限制中间件
func CreateConcurrencyLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过对状态检查和静态资源的限制
			if r.URL.Path == "/health" || r.URL.Path == "/stats" ||
				r.URL.Path == "/" || r.URL.Path == "/favicon.ico" ||
				r.URL.Path == "/favicon.png" ||
				r.Method == http.MethodOptions ||
				strings.HasPrefix(r.URL.Path, "/screenshots/") {
				next.ServeHTTP(w, r)
				return
			}

			// 设置超时上下文
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			// 尝试获取许可
			err := AcquireConcurrencyPermit(ctx)
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
			defer ReleaseConcurrencyPermit()

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// HandleStats Stats处理器 - 获取服务器状态
func HandleStats(w http.ResponseWriter, r *http.Request) {
	active, waiting, max, queue, uptime := GetConcurrencyStats()

	stats := map[string]interface{}{
		"active_requests":  active,
		"waiting_requests": waiting,
		"max_concurrent":   max,
		"queue_size":       queue,
		"uptime":           uptime.String(),
		"started_at":       startTime.Format(time.RFC3339),
	}

	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// HandleHealth Health检查处理器
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	SendJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "服务正常运行",
	})
}
