package rqlite

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	rqlitehttp "github.com/rqlite/rqlite-go-http"

	"jo3qma.com/yahoo_auctions_bot/internal/retry"
)

// HTTPClient は rqlite-go-http の Client 操作を抽象化する（テストで差し替え可能）。
type HTTPClient interface {
	PromoteErrors(b bool)
	ExecuteSingle(ctx context.Context, statement string, args ...any) (*rqlitehttp.ExecuteResponse, error)
	QuerySingle(ctx context.Context, statement string, args ...any) (*rqlitehttp.QueryResponse, error)
	Close() error
}

// Client は rqlite への接続を表す。watch.Repository の実装はこの Client を経由して利用する。
type Client struct {
	h HTTPClient
}

// NewClientOption は Open の挙動を上書きする。
type NewClientOption func(*openClientCfg)

type openClientCfg struct {
	newRqliteHTTPClient func(baseURL string, httpClient *http.Client) (HTTPClient, error)
}

// WithRqliteHTTPClientFactory は rqlite HTTP クライアント生成を差し替える（テスト用）。
func WithRqliteHTTPClientFactory(fn func(baseURL string, httpClient *http.Client) (HTTPClient, error)) NewClientOption {
	return func(c *openClientCfg) {
		c.newRqliteHTTPClient = fn
	}
}

// Open は baseURL の rqlite に接続し、スキーマを適用して Client を返す。
// rqlite のリーダー選出前に 503 が返る場合があるため、リトライ可能なエラーはバックオフ付きで再試行する。
func Open(ctx context.Context, baseURL string, opts ...NewClientOption) (*Client, error) {
	cfg := openClientCfg{
		newRqliteHTTPClient: func(baseURL string, httpClient *http.Client) (HTTPClient, error) {
			return rqlitehttp.NewClient(baseURL, httpClient)
		},
	}
	for _, o := range opts {
		o(&cfg)
	}

	raw, err := cfg.newRqliteHTTPClient(baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("rqlite client: %w", err)
	}
	raw.PromoteErrors(true)

	statements := splitSchema(schema)
	for _, stmt := range statements {
		if err := retry.Do(ctx, func() error {
			_, err := raw.ExecuteSingle(ctx, stmt)
			return err
		}, isRetryableRqliteError, nil); err != nil {
			_ = raw.Close()
			return nil, fmt.Errorf("rqlite init schema: %w", err)
		}
	}

	return &Client{h: raw}, nil
}

func isRetryableRqliteError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "503") ||
		strings.Contains(s, "leader not found") ||
		strings.Contains(s, "store not open")
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
	return c.h.Close()
}
