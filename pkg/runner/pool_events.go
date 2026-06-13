package runner

import (
	"sync"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// PoolEventType 池事件类型
type PoolEventType string

const (
	// EventScreenshotStart 截图开始
	EventScreenshotStart PoolEventType = "screenshot_start"
	// EventScreenshotComplete 截图成功完成
	EventScreenshotComplete PoolEventType = "screenshot_complete"
	// EventScreenshotFailed 截图失败
	EventScreenshotFailed PoolEventType = "screenshot_failed"
	// EventReconnect 浏览器进程重新连接
	EventReconnect PoolEventType = "reconnect"
	// EventIdleClose 空闲超时关闭浏览器进程
	EventIdleClose PoolEventType = "idle_close"
	// EventPoolClosed 连接池关闭
	EventPoolClosed PoolEventType = "pool_closed"
)

// PoolEvent 池事件
type PoolEvent struct {
	Type           PoolEventType  // 事件类型
	URL            string         // 截图 URL（仅截图相关事件）
	Duration       time.Duration  // 截图耗时（仅截图完成/失败事件）
	Error          error          // 错误信息（仅失败事件）
	ReconnectCount int64          // 重连次数（仅重连事件）
	Timestamp      time.Time      // 事件时间
	Result         *models.Result // 截图结果（仅完成事件）
}

// PoolEventHandler 池事件回调函数
type PoolEventHandler func(event PoolEvent)

// eventBus 池事件总线
// 支持注册多个事件监听器，在池状态变化时触发回调
type eventBus struct {
	mu       sync.RWMutex
	handlers []PoolEventHandler
}

// newEventBus 创建事件总线
func newEventBus() *eventBus {
	return &eventBus{
		handlers: make([]PoolEventHandler, 0),
	}
}

// On 注册事件监听器
func (eb *eventBus) On(handler PoolEventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers = append(eb.handlers, handler)
}

// emit 触发事件
func (eb *eventBus) emit(event PoolEvent) {
	eb.mu.RLock()
	handlers := make([]PoolEventHandler, len(eb.handlers))
	copy(handlers, eb.handlers)
	eb.mu.RUnlock()

	for _, handler := range handlers {
		// 异步调用，避免阻塞主流程
		go func(h PoolEventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Warn("事件处理器 panic", "error", r)
				}
			}()
			h(event)
		}(handler)
	}
}

// emitScreenshotStart 触发截图开始事件
func (eb *eventBus) emitScreenshotStart(url string) {
	eb.emit(PoolEvent{
		Type:      EventScreenshotStart,
		URL:       url,
		Timestamp: time.Now(),
	})
}

// emitScreenshotComplete 触发截图完成事件
func (eb *eventBus) emitScreenshotComplete(url string, duration time.Duration, result *models.Result) {
	eb.emit(PoolEvent{
		Type:      EventScreenshotComplete,
		URL:       url,
		Duration:  duration,
		Result:    result,
		Timestamp: time.Now(),
	})
}

// emitScreenshotFailed 触发截图失败事件
func (eb *eventBus) emitScreenshotFailed(url string, duration time.Duration, err error) {
	eb.emit(PoolEvent{
		Type:      EventScreenshotFailed,
		URL:       url,
		Duration:  duration,
		Error:     err,
		Timestamp: time.Now(),
	})
}

// emitReconnect 触发浏览器重连事件
func (eb *eventBus) emitReconnect(reconnectCount int64) {
	eb.emit(PoolEvent{
		Type:           EventReconnect,
		ReconnectCount: reconnectCount,
		Timestamp:      time.Now(),
	})
}

// emitIdleClose 触发空闲关闭事件
func (eb *eventBus) emitIdleClose() {
	eb.emit(PoolEvent{
		Type:      EventIdleClose,
		Timestamp: time.Now(),
	})
}

// emitPoolClosed 触发池关闭事件
func (eb *eventBus) emitPoolClosed() {
	eb.emit(PoolEvent{
		Type:      EventPoolClosed,
		Timestamp: time.Now(),
	})
}
