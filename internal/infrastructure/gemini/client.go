package gemini

import (
	"context"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// Client は Gemini API で Extraction を行うクライアント。
type Client interface {
	Extract(ctx context.Context, in product.ExtractInput) (*product.Product, error)
}

type client struct {
	sess *session
}

// NewClient は Gemini Extraction クライアントを生成する。
func NewClient(apiKey string, opts Options) (Client, error) {
	opts = opts.Normalize()
	api, err := newGenAIAPI(apiKey)
	if err != nil {
		return nil, err
	}
	return &client{sess: newSession(api, opts)}, nil
}

// NewTestClient は genAIAPI を注入する（テスト用）。
func NewTestClient(api *genAIAPI, opts Options) Client {
	return &client{sess: newSession(api, opts.Normalize())}
}

// Extract はタイトル・説明・画像から Category 判別と Field 抽出（Extraction）を行う。
func (c *client) Extract(ctx context.Context, in product.ExtractInput) (*product.Product, error) {
	return c.sess.run(ctx, in)
}
