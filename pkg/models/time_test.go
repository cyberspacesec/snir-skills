package models

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	before := time.Now()
	result := Now()
	after := time.Now()

	// 验证 Now() 返回的时间介于调用前后的时间之间
	if result.Before(before) || result.After(after) {
		t.Errorf("Now() 返回的时间应该在 %v 和 %v 之间，但得到 %v", before, after, result)
	}
}
