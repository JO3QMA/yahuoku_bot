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
	p    *pipeline
	sess *session
}

// NewClient は Gemini Extraction クライアントを生成する。ExtractionMode で pipeline / session を切り替える。
func NewClient(apiKey string, opts Options) (Client, error) {
	opts = opts.Normalize()
	api, err := newGenAIAPI(apiKey)
	if err != nil {
		return nil, err
	}
	return newClientFromAPI(api, opts), nil
}

// NewTestClient は genAIAPI を注入する（テスト用）。
func NewTestClient(api *genAIAPI, opts Options) Client {
	return newClientFromAPI(api, opts.Normalize())
}

func newClientFromAPI(api *genAIAPI, opts Options) Client {
	c := &client{}
	switch opts.ExtractionMode {
	case ExtractionModeSession:
		c.sess = newSession(api, opts)
	default:
		c.p = newPipeline(api, opts)
	}
	return c
}

// Extract はタイトル・説明・画像から Category 判別と Field 抽出（Extraction）を行う。
func (c *client) Extract(ctx context.Context, in product.ExtractInput) (*product.Product, error) {
	if c.sess != nil {
		return c.sess.run(ctx, in)
	}
	return c.p.run(ctx, in)
}
