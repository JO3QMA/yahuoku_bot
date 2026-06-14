package gemini

import (
	"context"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// Client は Gemini API で Extraction を行うクライアント。
type Client interface {
	Extract(ctx context.Context, in appauction.ExtractInput) (*product.Product, error)
}

type pipelineRunner interface {
	run(ctx context.Context, in appauction.ExtractInput) (*product.Product, error)
}

// client はClientの実装。
type client struct {
	runner pipelineRunner
}

// NewClient は多段 Extraction パイプライン付き Gemini クライアントを生成する。
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

// Extract はタイトル・説明・画像から Category 判別と Field 抽出（Extraction）を行う。
func (c *client) Extract(ctx context.Context, in appauction.ExtractInput) (*product.Product, error) {
	return c.runner.run(ctx, in)
}
