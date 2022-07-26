package util

import (
	"context"
	"time"
)

// Ticker return a ticker, closed when ctx Done
func Ticker(ctx context.Context, duration time.Duration) <-chan time.Time {
	ticker := time.NewTicker(duration)
	go func() {
		<-ctx.Done()
		ticker.Stop()
	}()
	return ticker.C
}
