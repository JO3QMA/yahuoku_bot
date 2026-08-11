package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

// previewDeps は RunPreview の依存注入用。
type previewDeps struct {
	LoadConfig       func() (*config.Config, error)
	NewOpenAIClient  func(cfg *config.Config) (openai.Client, error)
	NewAuctionClient func(baseURL string) infraauction.Client
}

func mergePreviewDeps(d *previewDeps) {
	if d.LoadConfig == nil {
		d.LoadConfig = config.Load
	}
	if d.NewOpenAIClient == nil {
		d.NewOpenAIClient = func(cfg *config.Config) (openai.Client, error) {
			opts := openai.NewOptions(
				cfg.OpenAIBaseURL, cfg.OpenAIModel, cfg.OpenAIModelVision, cfg.OpenAIModelAgent,
				cfg.OpenAIMaxImages, cfg.OpenAIMaxSearchCalls, cfg.OpenAIPipelineTimeoutSec,
			)
			return openai.NewClient(cfg.OpenAIAPIKey, opts)
		}
	}
	if d.NewAuctionClient == nil {
		d.NewAuctionClient = func(baseURL string) infraauction.Client {
			return infraauction.NewClient(baseURL, (*http.Client)(nil))
		}
	}
}

// RunPreview はプレビューCLIの本体。終了コード: 0=成功、1=空Product、2=引数エラー。
func RunPreview(stdout io.Writer, argv []string, deps *previewDeps) int {
	if deps == nil {
		deps = &previewDeps{}
	}
	mergePreviewDeps(deps)

	if len(argv) < 1 {
		log.Print("usage: preview <auction_id>")
		return 2
	}
	auctionID := argv[0]

	cfg, err := deps.LoadConfig()
	if err != nil {
		log.Printf("config load: %v", err)
		return 2
	}
	if cfg.OpenAIAPIKey == "" {
		log.Print("OPENAI_API_KEY is required")
		return 2
	}

	auctionClient := deps.NewAuctionClient(cfg.APIEndpoint)
	openaiClient, err := deps.NewOpenAIClient(cfg)
	if err != nil {
		log.Printf("openai client: %v", err)
		return 2
	}
	previewUsecase := appauction.NewPreviewUsecase(auctionClient, openaiClient)

	ctx := context.Background()
	preview, err := previewUsecase.Execute(ctx, auctionID)
	if err != nil {
		log.Printf("preview execute: %v", err)
		return 2
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(preview); err != nil {
		log.Printf("encode: %v", err)
		return 2
	}

	if preview.Product.IsEffectivelyEmpty() {
		log.Print("warning: product extraction returned no usable data")
		return 1
	}
	return 0
}
