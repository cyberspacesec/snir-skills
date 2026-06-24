package api

import (
	"context"
	"net/http"
	"net/http/httptest"
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
func TestGlobalConcurrencyLimitMiddleware(t *testing.T) {
	// 初始化全局并发限制器
	InitConcurrencyLimiter(1, 1)

	// 创建中间件
	middleware := CreateConcurrencyLimitMiddleware()

	// 创建测试处理器
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟一个需要一些时间的处理
		time.Sleep(200 * time.Millisecond)
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

	// 测试并发限制
	t.Run("并发限制", func(t *testing.T) {
		// 启动第一个请求占用工作槽
		go func() {
			http.Get(srv.URL + "/screenshot")
		}()

		// 启动第二个请求占用队列中的位置
		go func() {
			http.Get(srv.URL + "/screenshot")
		}()

		// 给一些时间让前两个请求处理
		time.Sleep(50 * time.Millisecond)

		// 第三个请求应该返回429，因为队列已满
		resp, err := http.Get(srv.URL + "/screenshot")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTooManyRequests {
			t.Errorf("期望状态码 %d，但得到 %d", http.StatusTooManyRequests, resp.StatusCode)
		}

		// 等待所有请求完成
		time.Sleep(400 * time.Millisecond)
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
			timeoutCtx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)
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
