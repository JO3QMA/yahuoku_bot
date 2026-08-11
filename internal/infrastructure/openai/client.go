package openai

import (
	"context"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// Client は OpenAI 互換 API で Extraction を行うクライアント。
type Client interface {
	Extract(ctx context.Context, in product.ExtractInput) (*product.Product, error)
}

// NewClient は OpenAI 互換 API を使用する Extraction クライアントを生成する。
func NewClient(apiKey string, opts Options) (Client, error) {
	opts = opts.Normalize()
	api, err := newAPIClient(apiKey, opts.BaseURL)
	if err != nil {
		return nil, err
	}
	return newSession(api, opts), nil
}

// NewTestClient は apiClient を注入する（テスト用）。
func NewTestClient(api *apiClient, opts Options) Client {
	return newSession(api, opts.Normalize())
}
