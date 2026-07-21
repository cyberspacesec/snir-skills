package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewConcurrencyLimiter(t *testing.T) {
	tests := []struct {
		name          string
		maxConcurrent int
		waitQueue     int
		expectNil     bool
	}{
		{
			name:          "正常参数",
			maxConcurrent: 10,
			waitQueue:     100,
			expectNil:     false,
		},
		{
			name:          "负的最大并发数",
			maxConcurrent: -1,
			waitQueue:     100,
			expectNil:     false, // 会使用默认值
		},
		{
			name:          "零最大并发数",
			maxConcurrent: 0,
			waitQueue:     100,
			expectNil:     false, // 会使用默认值
		},
		{
			name:          "负的等待队列",
			maxConcurrent: 10,
			waitQueue:     -1,
			expectNil:     false, // 会使用默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewConcurrencyLimiter(tt.maxConcurrent, tt.waitQueue)
			if (limiter == nil) != tt.expectNil {
				t.Errorf("NewConcurrencyLimiter() = %v, expectNil %v", limiter, tt.expectNil)
			}

			if limiter != nil {
				// 检查是否使用了默认值
				if tt.maxConcurrent <= 0 && limiter.maxConcurrent <= 0 {
					t.Errorf("应该使用默认并发数，但得到 %v", limiter.maxConcurrent)
				}
				if tt.waitQueue <= 0 && limiter.waitQueue <= 0 {
					t.Errorf("应该使用默认队列大小，但得到 %v", limiter.waitQueue)
				}
			}
		})
	}
}

func TestAcquireAndRelease(t *testing.T) {
	// 创建一个并发限制器，最大并发数为2
	limiter := NewConcurrencyLimiter(2, 5)

	// 测试能够获取两个并发许可
	ctx := context.Background()

	// 获取第一个许可
	err1 := limiter.Acquire(ctx)
	if err1 != nil {
		t.Errorf("获取第一个许可失败: %v", err1)
	}

	// 获取第二个许可
	err2 := limiter.Acquire(ctx)
	if err2 != nil {
		t.Errorf("获取第二个许可失败: %v", err2)
	}

	// 尝试获取第三个许可，应该会被阻塞，使用带超时的上下文测试
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		// 这个获取操作应该会阻塞直到超时
		err3 := limiter.Acquire(ctxWithTimeout)
		if err3 != context.DeadlineExceeded {
			t.Errorf("预期超时错误，但得到: %v", err3)
		}
		close(done)
	}()

	// 等待超时
	select {
	case <-done:
		// 正常，超时了
	case <-time.After(200 * time.Millisecond):
		t.Errorf("测试超时")
	}

	// 释放一个许可
	limiter.Release()

	// 现在应该可以获取另一个许可了
	err4 := limiter.Acquire(ctx)
	if err4 != nil {
		t.Errorf("释放后获取许可失败: %v", err4)
	}

	// 最后清理
	limiter.Release()
	limiter.Release()
}

func TestConcurrencyStats(t *testing.T) {
	// 创建一个并发限制器
	limiter := NewConcurrencyLimiter(5, 10)
	ctx := context.Background()

	// 获取两个许可
	_ = limiter.Acquire(ctx)
	_ = limiter.Acquire(ctx)

	// 获取统计信息
	active, waiting, maxConcurrent, queueSize := limiter.Stats()

	if active != 2 {
		t.Errorf("活跃连接数应为2，但得到: %v", active)
	}

	if waiting != 0 {
		t.Errorf("等待连接数应为0，但得到: %v", waiting)
	}

	if maxConcurrent != 5 {
		t.Errorf("最大并发数应为5，但得到: %v", maxConcurrent)
	}

	if queueSize != 10 {
		t.Errorf("队列大小应为10，但得到: %v", queueSize)
	}

	// 再次检查当前并发数
	if limiter.CurrentConcurrency() != 2 {
		t.Errorf("CurrentConcurrency()应返回2，但得到: %v", limiter.CurrentConcurrency())
	}

	// 最后清理
	limiter.Release()
	limiter.Release()
}

func TestProcessConcurrent(t *testing.T) {
	// 创建测试请求列表
	requests := []ScreenshotRequest{
		{URL: "http://example1.com"},
		{URL: "http://example2.com"},
		{URL: "http://example3.com"},
		{URL: "http://example4.com"},
		{URL: "http://example5.com"},
	}

	// 设置并发数
	concurrency := 2

	// 创建结果通道
	resultsChan := make(chan BatchResult, len(requests))

	// 记录处理顺序的互斥锁和切片
	var mu sync.Mutex
	var processed []string

	// 创建处理函数
	processor := func(req ScreenshotRequest) BatchResult {
		// 记录处理的URL
		mu.Lock()
		processed = append(processed, req.URL)
		mu.Unlock()

		// 模拟处理时间
		time.Sleep(10 * time.Millisecond)

		return BatchResult{
			URL: req.URL,
		}
	}

	// 启动并发处理
	startTime := time.Now()
	ProcessConcurrent(requests, concurrency, processor, resultsChan)
	duration := time.Since(startTime)

	// 检查是否收到所有结果
	var results []BatchResult
	for i := 0; i < len(requests); i++ {
		results = append(results, <-resultsChan)
	}

	// 验证结果计数
	if len(results) != len(requests) {
		t.Errorf("预期有%d个结果，但得到%d个", len(requests), len(results))
	}

	// 验证处理时间（应该比串行处理快）
	// 5个请求，每个10ms，串行需要50ms，并发2应该接近30ms
	// 考虑到测试环境的不确定性，我们只做粗略检查
	if duration > 50*time.Millisecond {
		t.Errorf("处理时间超过预期: %v", duration)
	}
}

// TestNewBasicConcurrencyLimiter 测试基本并发限制器的创建
func TestNewBasicConcurrencyLimiter(t *testing.T) {
	tests := []struct {
		name           string
		maxConcurrency int
		wantNil        bool
	}{
		{
			name:           "有效的配置",
			maxConcurrency: 10,
			wantNil:        false,
		},
		{
			name:           "非法的最大并发值",
			maxConcurrency: 0,
			wantNil:        true,
		},
		{
			name:           "负数最大并发值",
			maxConcurrency: -5,
			wantNil:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewBasicConcurrencyLimiter(tt.maxConcurrency)
			if (limiter == nil) != tt.wantNil {
				t.Errorf("NewBasicConcurrencyLimiter() = %v, wantNil %v", limiter, tt.wantNil)
				return
			}

			if !tt.wantNil && limiter == nil {
				t.Errorf("NewBasicConcurrencyLimiter() 返回了nil但应该返回一个有效对象")
			}

			if !tt.wantNil {
				// 测试MaxConcurrency方法
				maxConcurrency := limiter.MaxConcurrency()
				if maxConcurrency != tt.maxConcurrency {
					t.Errorf("MaxConcurrency() = %v, 期望 %v", maxConcurrency, tt.maxConcurrency)
				}
			}
		})
	}
}

// TestConcurrencyLimiter_Acquire 测试并发限制器的获取和释放
func TestConcurrencyLimiter_Acquire(t *testing.T) {
	t.Run("基本获取和释放", func(t *testing.T) {
		// 创建有效的限制器
		limiter := NewBasicConcurrencyLimiter(2)
		if limiter == nil {
			t.Fatal("无法创建限制器")
		}

		// 测试获取
		ctx := context.Background()
		err := limiter.Acquire(ctx)
		if err != nil {
			t.Errorf("Acquire() error = %v", err)
		}

		// 检查当前并发数
		if current := limiter.CurrentConcurrency(); current != 1 {
			t.Errorf("并发计数错误，当前值: %d, 期望: 1", current)
		}

		// 测试释放
		limiter.Release()

		// 检查当前并发数
		if current := limiter.CurrentConcurrency(); current != 0 {
			t.Errorf("Release后并发计数错误，当前值: %d, 期望: 0", current)
		}
	})

	t.Run("达到最大并发", func(t *testing.T) {
		// 创建一个小的限制器
		limiter := NewBasicConcurrencyLimiter(1)
		if limiter == nil {
			t.Fatal("无法创建限制器")
		}

		// 先获取一个许可
		ctx := context.Background()
		err := limiter.Acquire(ctx)
		if err != nil {
			t.Errorf("第一个Acquire() error = %v", err)
		}

		// 创建带超时的上下文
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// 尝试获取第二个许可，应该超时
		err = limiter.Acquire(timeoutCtx)
		if err == nil {
			t.Error("第二个Acquire()应该返回错误，但返回nil")
			// 释放额外获取的许可
			limiter.Release()
		}

		// 释放第一个许可
		limiter.Release()
	})
}

// TestGlobalConcurrencyLimitMiddleware 测试全局并发限制中间件
// 使用 AcquireConcurrencyPermit 手动占满工作槽与队列，再发起 HTTP 请求断言返回 429，
// 完全避免依赖异步 http.Get goroutine（否则会因 handler 阻塞导致测试 hang）。
func TestGlobalConcurrencyLimitMiddleware(t *testing.T) {
	// 重置全局并发限制器（防止被先执行的测试以更大上限初始化），再以 (1,1) 初始化。
	ResetConcurrencyLimiter()
	InitConcurrencyLimiter(1, 1)
	t.Cleanup(ResetConcurrencyLimiter)

	// 创建中间件
	middleware := CreateConcurrencyLimitMiddleware()

	// 创建测试处理器：立即返回，不阻塞。
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("处理完成"))
	})

	// 创建测试服务器
	handler := middleware(testHandler)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// 测试特殊路径不受限制
	t.Run("特殊路径不受限制", func(t *testing.T) {
		// 测试不同的特殊路径
		specialPaths := []string{
			"/health",
			"/stats",
			"/",
			"/favicon.ico",
			"/screenshots/1.png",
		}

		for _, path := range specialPaths {
			resp, err := http.Get(srv.URL + path)
			if err != nil {
				t.Errorf("路径 %s 请求失败: %v", path, err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("路径 %s 返回了意外的状态码: %d", path, resp.StatusCode)
			}
		}
	})

	// 测试并发限制：手动占满工作槽(1)与队列(1)，第三个请求应被拒绝返回 429
	t.Run("并发限制", func(t *testing.T) {
		ctx := context.Background()

		// 占用唯一的工作槽
		if err := AcquireConcurrencyPermit(ctx); err != nil {
			t.Fatalf("占用工作槽失败: %v", err)
		}
		// defer 在子测试结束前释放工作槽，避免污染后续测试
		defer ReleaseConcurrencyPermit()

		// 用一个带短超时的 goroutine 占满队列中的 1 个等待位（Acquire 会阻塞，超时后返回）
		waitDone := make(chan struct{})
		go func() {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			// 这次 Acquire 会进入等待队列（active 已满），占满队列后返回
			_ = AcquireConcurrencyPermit(timeoutCtx)
			close(waitDone)
		}()

		// 轮询等待队列被占满（最多 1s）
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			_, waiting, _, _, _ := GetConcurrencyStats()
			if waiting >= 1 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}

		// 此时工作槽与队列均满，第三个请求应返回 429
		resp, err := http.Get(srv.URL + "/screenshot")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusTooManyRequests {
			t.Errorf("期望状态码 %d，但得到 %d", http.StatusTooManyRequests, resp.StatusCode)
		}

		// 等待占满队列的 goroutine 超时退出（它会因 ctx 超时退出并 Release 等待计数）
		select {
		case <-waitDone:
		case <-time.After(3 * time.Second):
			t.Error("等待队列 goroutine 未在预期时间内退出")
		}
	})
}

// TestAcquireReleaseEdgeCases tests edge cases for Acquire and Release
func TestAcquireReleaseEdgeCases(t *testing.T) {
	t.Run("Acquire without wait queue uses simpler path", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(2, 0)
		ctx := context.Background()

		// Acquire without wait queue counting
		err := limiter.Acquire(ctx)
		if err != nil {
			t.Errorf("Acquire() without wait queue error = %v", err)
		}

		if current := limiter.CurrentConcurrency(); current != 1 {
			t.Errorf("CurrentConcurrency() = %v, want 1", current)
		}

		// Release without active count
		limiter.Release()
		if current := limiter.CurrentConcurrency(); current != 0 {
			t.Errorf("CurrentConcurrency() after release = %v, want 0", current)
		}
	})

	t.Run("Acquire with timeout without wait queue", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(1, 0)
		ctx := context.Background()

		// Take the only slot
		err := limiter.Acquire(ctx)
		if err != nil {
			t.Errorf("first Acquire() error = %v", err)
		}

		// Try to acquire with timeout
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err = limiter.Acquire(timeoutCtx)
		if err == nil {
			t.Error("expected timeout error, got nil")
			limiter.Release()
		}

		limiter.Release()
	})

	t.Run("Release without active count when waitQueue is 0", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(2, 0)

		// Release on empty semaphore should not panic
		limiter.Release()
		limiter.Release()
		// Should not panic or deadlock
	})

	t.Run("Release double-release prevention", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(2, 100)
		ctx := context.Background()

		limiter.Acquire(ctx)
		limiter.Release()
		// Double release should be safe
		limiter.Release()

		// Verify stats
		active, _, _, _ := limiter.Stats()
		if active < 0 {
			t.Errorf("active count should not be negative: %v", active)
		}
	})

	t.Run("Acquire queue full", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(1, 1)
		ctx := context.Background()

		// Take the slot
		limiter.Acquire(ctx)

		// Fill the queue
		timeoutCtx1, cancel1 := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel1()
		go limiter.Acquire(timeoutCtx1)
		time.Sleep(50 * time.Millisecond) // Give it time to enter queue

		// This should be rejected immediately because queue is full
		err := limiter.Acquire(ctx)
		if err == nil {
			t.Error("expected queue full error, got nil")
		} else if err.Error() != "服务器繁忙，请求队列已满" {
			t.Errorf("expected queue full message, got: %v", err.Error())
		}

		limiter.Release()
		time.Sleep(100 * time.Millisecond) // Let queued goroutine finish
	})

	t.Run("Stats with waitQueue=0", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(5, 0)
		active, waiting, maxConc, queueSize := limiter.Stats()

		if maxConc != 5 {
			t.Errorf("MaxConcurrent = %v, want 5", maxConc)
		}
		// NewConcurrencyLimiter overrides 0 with default 100
		if queueSize != 100 {
			t.Errorf("QueueSize = %v, want 100 (default override)", queueSize)
		}
		// active and waiting should be 0 initially
		if active != 0 || waiting != 0 {
			t.Errorf("active=%v waiting=%v, want both 0", active, waiting)
		}
	})
}

// TestServerCreateConcurrencyLimitMiddleware tests the server's CreateConcurrencyLimitMiddleware method
func TestServerCreateConcurrencyLimitMiddleware(t *testing.T) {
	// Test with nil concurrencyLimit (no limit set)
	t.Run("nil concurrency limit", func(t *testing.T) {
		server := &Server{
			concurrencyLimit: nil,
		}
		middleware := server.CreateConcurrencyLimitMiddleware()

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))

		req, _ := http.NewRequest("POST", "/screenshot", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	// Test with invalid type
	t.Run("invalid concurrencyLimit type", func(t *testing.T) {
		server := &Server{
			concurrencyLimit: "invalid",
		}
		middleware := server.CreateConcurrencyLimitMiddleware()

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))

		req, _ := http.NewRequest("POST", "/screenshot", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	// Test with valid limiter - special paths bypass
	t.Run("valid limiter special paths bypass", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(2, 5)
		server := &Server{
			concurrencyLimit: limiter,
		}
		middleware := server.CreateConcurrencyLimitMiddleware()

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))

		specialPaths := []string{"/health", "/stats", "/", "/screenshots/test.png"}
		for _, path := range specialPaths {
			req, _ := http.NewRequest("GET", path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("path %s: status = %v, want %v", path, rr.Code, http.StatusOK)
			}
		}
	})

	// Test with valid limiter - blocked by capacity
	t.Run("valid limiter blocks when full", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(1, 1)
		server := &Server{
			concurrencyLimit: limiter,
		}
		middleware := server.CreateConcurrencyLimitMiddleware()

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))

		// Take the slot
		ctx := context.Background()
		limiter.Acquire(ctx)

		// Fill the queue with one waiting request
		go func() {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			limiter.Acquire(timeoutCtx)
		}()
		time.Sleep(50 * time.Millisecond)

		// This should be rejected (queue full)
		req, _ := http.NewRequest("POST", "/screenshot", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("status = %v, want %v", rr.Code, http.StatusTooManyRequests)
		}

		// Clean up
		limiter.Release()
		time.Sleep(100 * time.Millisecond)
	})

	// Test with valid limiter - timeout
	t.Run("valid limiter timeout", func(t *testing.T) {
		limiter := NewConcurrencyLimiter(1, 5)
		server := &Server{
			concurrencyLimit: limiter,
		}
		middleware := server.CreateConcurrencyLimitMiddleware()

		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}))

		// Take the slot
		ctx := context.Background()
		limiter.Acquire(ctx)

		// Release after a delay so the middleware request can succeed
		go func() {
			time.Sleep(200 * time.Millisecond)
			limiter.Release()
		}()

		// This should succeed after the slot is released
		req, _ := http.NewRequest("POST", "/screenshot", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %v, want %v", rr.Code, http.StatusOK)
		}
	})
}

// TestConcurrencyLimiter_Acquire_QueueFull 覆盖 Acquire 的等待队列满拒绝分支
// （concurrency.go:62-65）。
func TestConcurrencyLimiter_Acquire_QueueFull(t *testing.T) {
	// maxConcurrent=1, waitQueue=1：占满信号量（1）+ 等待队列（1），第 3 个应被拒
	limiter := NewConcurrencyLimiter(1, 1)

	// 占用唯一的信号量槽位
	if err := limiter.Acquire(context.Background()); err != nil {
		t.Fatalf("首次 Acquire 应成功: %v", err)
	}

	// 发起 1 个阻塞 Acquire（waitCount→1=queue 满），用可取消 ctx 便于后续退出
	blockCtx, blockCancel := context.WithCancel(context.Background())
	blockDone := make(chan error, 1)
	go func() { blockDone <- limiter.Acquire(blockCtx) }()
	time.Sleep(50 * time.Millisecond) // 让它进入等待

	// 此时 waitCount=1=waitQueue，第 3 个 Acquire 应立即拒绝
	if err := limiter.Acquire(context.Background()); err == nil {
		t.Error("队列满时应返回错误")
	} else if !strings.Contains(err.Error(), "队列已满") {
		t.Logf("拒绝错误（预期）: %v", err)
	}

	// 取消阻塞的 Acquire 让它退出（释放槽位给后续清理）
	blockCancel()
	select {
	case <-blockDone:
	case <-time.After(time.Second):
		t.Fatal("阻塞 Acquire 未在取消后返回")
	}
	limiter.Release()
}

// TestConcurrencyLimiter_Acquire_CtxCancelled 覆盖 Acquire 的
// ctx.Done 分支（concurrency.go:78-82，waitQueue>0 时等待中被取消）。
func TestConcurrencyLimiter_Acquire_CtxCancelled(t *testing.T) {
	limiter := NewConcurrencyLimiter(1, 10)
	// 占用信号量
	if err := limiter.Acquire(context.Background()); err != nil {
		t.Fatalf("首次 Acquire 应成功: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- limiter.Acquire(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("ctx 取消应返回错误")
		} else if !errors.Is(err, context.Canceled) {
			t.Errorf("应返回 context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Acquire 未在取消后返回")
	}

	limiter.Release()
}

// TestConcurrencyLimiter_Release_NoActive 覆盖 Release 的
// activeCount==0 跳过分支（concurrency.go:100 false 分支）。
func TestConcurrencyLimiter_Release_NoActive(t *testing.T) {
	limiter := NewConcurrencyLimiter(2, 10)
	// 未 Acquire 直接 Release，activeCount=0 → 不释放信号量
	limiter.Release()
	limiter.Release() // 重复释放不应 panic
}

// TestConcurrencyLimiter_Acquire_NoQueueCtxCancelled 覆盖 Acquire 的
// waitQueue<=0 简化分支的 ctx.Done（concurrency.go:89-90）。
func TestConcurrencyLimiter_Acquire_NoQueueCtxCancelled(t *testing.T) {
	// waitQueue=0（实际 NewConcurrencyLimiter 会设默认 100，需手动构造 0）
	limiter := &ConcurrencyLimiter{
		maxConcurrent: 1,
		semaphore:     make(chan struct{}, 1),
		waitQueue:     0,
	}
	// 占用信号量
	limiter.semaphore <- struct{}{}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- limiter.Acquire(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("ctx 取消应返回错误")
		}
	case <-time.After(time.Second):
		t.Fatal("Acquire 未在取消后返回")
	}
}

// TestConcurrencyLimiter_Release_NoQueueDefault 覆盖 Release 的
// waitQueue<=0 简化分支的 default（无信号量可释放，concurrency.go:109-111）。
func TestConcurrencyLimiter_Release_NoQueueDefault(t *testing.T) {
	limiter := &ConcurrencyLimiter{
		maxConcurrent: 2,
		semaphore:     make(chan struct{}, 2),
		waitQueue:     0,
	}
	// 未占用直接 Release → default 分支，不 panic
	limiter.Release()
}
