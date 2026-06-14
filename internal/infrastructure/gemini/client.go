package gemini

import (
	"context"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// Client はGemini APIを用いて商品説明から商品情報を抽出するクライアント。
type Client interface {
	ExtractProduct(ctx context.Context, in appauction.ExtractInput) (*product.ProductDetail, error)
}

type pipelineRunner interface {
	run(ctx context.Context, in appauction.ExtractInput) (*product.ProductDetail, error)
}

// client はClientの実装。
type client struct {
	runner pipelineRunner
}

// NewClient は多段推論パイプライン付き Gemini クライアントを生成する。
func NewClient(apiKey string, opts Options) (Client, error) {
	opts = opts.Normalize()
	api, err := newGenAIAPI(apiKey)
	if err != nil {
		return nil, err
	}
	return &client{runner: newPipeline(api, opts)}, nil
}

// NewClientWithRunner は runner を注入する（テスト用）。
func NewClientWithRunner(runner pipelineRunner) Client {
	return &client{runner: runner}
}

// NewClientWithAPI は API ラッパーを注入する（テスト用）。
func NewClientWithAPI(api *genAIAPI, opts Options) Client {
	return &client{runner: newPipeline(api, opts.Normalize())}
}

// ExtractProduct はタイトル・説明・画像からジャンル判別とテンプレート項目を抽出する。
func (c *client) ExtractProduct(ctx context.Context, in appauction.ExtractInput) (*product.ProductDetail, error) {
	return c.runner.run(ctx, in)
}
