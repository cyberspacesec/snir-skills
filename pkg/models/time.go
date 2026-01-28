package models

import "time"

// Now 返回当前时间
// 这个函数封装了标准库的 time.Now() 函数
// 提供了在整个应用中统一获取当前时间的方式
// 便于在需要的地方替换为模拟时间用于测试
func Now() time.Time {
	return time.Now()
}
