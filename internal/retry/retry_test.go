package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo_retriesThenSucceeds(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("503")
		}
		return nil
	}, func(err error) bool { return err.Error() == "503" }, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d want 3", calls)
	}
}

func TestDo_ctxCancelDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := Do(ctx, func() error { return errors.New("503") }, func(error) bool { return true }, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v want deadline exceeded", err)
	}
}
