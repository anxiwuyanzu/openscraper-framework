package util

import (
	"context"
	"time"
)

// sleep with cancel context
func Sleep(ctx context.Context, d time.Duration) int {
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-time.After(d):
			return 1
		}
	}
}

// Daily 每天{hour}点执行一次 fn
func Daily(hour int, fn func()) *time.Timer {
	var timer *time.Timer
	dailyFunc := func() {
		timer.Reset(24 * time.Hour)
		fn()
	}

	now := time.Now()
	var next time.Duration
	if now.Hour() < hour {
		next = time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()).Sub(now)
	} else {
		next = time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()).
			AddDate(0, 0, 1).Sub(now)
	}
	// 零点执行
	timer = time.AfterFunc(next, dailyFunc)
	return timer
}
