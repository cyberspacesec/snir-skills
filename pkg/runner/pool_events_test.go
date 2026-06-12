package runner

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestEventBus_On(t *testing.T) {
	eb := newEventBus()

	var received []PoolEventType
	var mu sync.Mutex

	eb.On(func(event PoolEvent) {
		mu.Lock()
		received = append(received, event.Type)
		mu.Unlock()
	})

	eb.emitScreenshotStart("https://example.com")
	eb.emitScreenshotComplete("https://example.com", 100*time.Millisecond, nil)
	eb.emitScreenshotFailed("https://example.com", 50*time.Millisecond, fmt.Errorf("timeout"))
	eb.emitReconnect(1)
	eb.emitIdleClose()
	eb.emitPoolClosed()

	// 等待异步事件处理
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// 因为异步，顺序不保证，只验证所有事件都收到了
	expected := map[PoolEventType]bool{
		EventScreenshotStart:    false,
		EventScreenshotComplete: false,
		EventScreenshotFailed:   false,
		EventReconnect:          false,
		EventIdleClose:          false,
		EventPoolClosed:         false,
	}

	for _, e := range received {
		if _, ok := expected[e]; ok {
			expected[e] = true
		}
	}

	for e, found := range expected {
		if !found {
			t.Errorf("缺少事件: %s", e)
		}
	}

	if len(received) != len(expected) {
		t.Errorf("收到 %d 个事件, 期望 %d", len(received), len(expected))
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	eb := newEventBus()

	count1, count2 := 0, 0
	var mu sync.Mutex

	eb.On(func(event PoolEvent) {
		mu.Lock()
		count1++
		mu.Unlock()
	})

	eb.On(func(event PoolEvent) {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	eb.emitScreenshotStart("https://example.com")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if count1 != 1 || count2 != 1 {
		t.Errorf("handler1 = %d, handler2 = %d, 期望都是 1", count1, count2)
	}
}

func TestEventBus_PanicRecovery(t *testing.T) {
	eb := newEventBus()

	// 注册一个会 panic 的处理器
	eb.On(func(event PoolEvent) {
		panic("test panic")
	})

	// 注册一个正常的处理器
	normalCalled := false
	var mu sync.Mutex
	eb.On(func(event PoolEvent) {
		mu.Lock()
		normalCalled = true
		mu.Unlock()
	})

	// 不应 panic
	eb.emitScreenshotStart("https://example.com")

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !normalCalled {
		t.Error("正常处理器应该被调用")
	}
}

func TestPoolEvent_Fields(t *testing.T) {
	eb := newEventBus()

	var captured PoolEvent
	var mu sync.Mutex

	eb.On(func(event PoolEvent) {
		mu.Lock()
		captured = event
		mu.Unlock()
	})

	eb.emitScreenshotFailed("https://example.com", 200*time.Millisecond, fmt.Errorf("net::ERR_TIMED_OUT"))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if captured.Type != EventScreenshotFailed {
		t.Errorf("Type = %s, 期望 %s", captured.Type, EventScreenshotFailed)
	}
	if captured.URL != "https://example.com" {
		t.Errorf("URL = %s, 期望 https://example.com", captured.URL)
	}
	if captured.Duration != 200*time.Millisecond {
		t.Errorf("Duration = %v, 期望 200ms", captured.Duration)
	}
	if captured.Error == nil || captured.Error.Error() != "net::ERR_TIMED_OUT" {
		t.Errorf("Error = %v, 期望 net::ERR_TIMED_OUT", captured.Error)
	}
	if captured.Timestamp.IsZero() {
		t.Error("Timestamp 应有值")
	}
}

func TestDriverPool_Events(t *testing.T) {
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		t.Skip("跳过需要浏览器的测试")
	}

	opts := &Options{}
	opts.Chrome.Headless = true
	opts.Chrome.WindowX = 1280
	opts.Chrome.WindowY = 800
	opts.Chrome.Timeout = 30
	opts.Scan.ScreenshotPath = t.TempDir()
	opts.Scan.ScreenshotFormat = "png"

	pool, err := NewDriverPool(opts, 2)
	if err != nil {
		t.Fatalf("NewDriverPool() error = %v", err)
	}
	defer pool.Close()

	var events []PoolEventType
	var mu sync.Mutex

	pool.On(func(event PoolEvent) {
		mu.Lock()
		events = append(events, event.Type)
		mu.Unlock()
	})

	_, err = pool.Screenshot("https://www.baidu.com", nil)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}

	// 等待异步事件处理
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// 应该收到 screenshot_start 和 screenshot_complete
	hasStart := false
	hasComplete := false
	for _, e := range events {
		if e == EventScreenshotStart {
			hasStart = true
		}
		if e == EventScreenshotComplete {
			hasComplete = true
		}
	}

	if !hasStart {
		t.Error("缺少 screenshot_start 事件")
	}
	if !hasComplete {
		t.Error("缺少 screenshot_complete 事件")
	}
}
