package rqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	rqlitehttp "github.com/rqlite/rqlite-go-http"
)

// Client は rqlite への接続を表す。watch.Repository の実装はこの Client を経由して利用する。
type Client struct {
	c *rqlitehttp.Client
}

// Open は baseURL の rqlite に接続し、スキーマを適用して Client を返す。
// rqlite のリーダー選出前に 503 が返る場合があるため、リトライ可能なエラーはバックオフ付きで再試行する。
func Open(ctx context.Context, baseURL string) (*Client, error) {
	c, err := rqlitehttp.NewClient(baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("rqlite client: %w", err)
	}
	c.PromoteErrors(true)

	statements := splitSchema(schema)
	const maxRetries = 5
	backoff := []time.Duration{0, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second}
	for _, stmt := range statements {
		if stmt == "" {
			continue
		}
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					_ = c.Close()
					return nil, ctx.Err()
				case <-time.After(backoff[attempt]):
				}
			}
			_, err = c.ExecuteSingle(ctx, stmt)
			if err == nil {
				break
			}
			if !isRetryableRqliteError(err) {
				_ = c.Close()
				return nil, fmt.Errorf("rqlite init schema: %w", err)
			}
		}
		if err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("rqlite init schema: %w", err)
		}
	}

	return &Client{c: c}, nil
}

func isRetryableRqliteError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "503") || strings.Contains(s, "leader not found")
}

func splitSchema(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// Close は接続を閉じる。
func (c *Client) Close() error {
	return c.c.Close()
}
