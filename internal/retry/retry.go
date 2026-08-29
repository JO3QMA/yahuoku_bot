package retry

import (
	"context"
	"time"
)

const MaxAttempts = 5

var backoff = []time.Duration{0, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second}

// Do calls fn up to MaxAttempts times with exponential backoff.
// retryable returns false for errors that should fail immediately.
// onRetry is called before each retry wait (nil-safe).
func Do(ctx context.Context, fn func() error, retryable func(error) bool, onRetry func(attempt int, err error)) error {
	var err error
	for attempt := 0; attempt < MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff[attempt]):
			}
		}
		err = fn()
		if err == nil {
			return nil
		}
		if !retryable(err) {
			return err
		}
		if attempt < MaxAttempts-1 && onRetry != nil {
			onRetry(attempt+1, err)
		}
	}
	return err
}
