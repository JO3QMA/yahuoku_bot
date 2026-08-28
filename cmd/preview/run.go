package main

import (
	"context"
	"encoding/json"
	"io"
	"log"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	infralisting "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

// previewDeps は RunPreview の依存注入用。
type previewDeps struct {
	LoadConfig      func() (*config.Config, error)
	NewOpenAIClient func(cfg *config.Config) (openai.Client, error)
	NewListingClient func() infralisting.Client
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
	if d.NewListingClient == nil {
		d.NewListingClient = infralisting.NewClient
	}
}

// RunPreview はプレビューCLIの本体。終了コード: 0=成功、1=空Product、2=引数エラー。
func RunPreview(stdout io.Writer, argv []string, deps *previewDeps) int {
	if deps == nil {
		deps = &previewDeps{}
	}
	mergePreviewDeps(deps)

	if len(argv) < 2 {
		log.Print("usage: preview <market> <listing_id>")
		return 2
	}
	market := listing.Market(argv[0])
	if !market.Valid() {
		log.Printf("unknown market %q (want yahoo_auction, yahoo_flea, or mercari)", argv[0])
		return 2
	}
	ref := listing.Ref{Market: market, ListingID: argv[1]}

	cfg, err := deps.LoadConfig()
	if err != nil {
		log.Printf("config load: %v", err)
		return 2
	}
	if cfg.OpenAIAPIKey == "" {
		log.Print("OPENAI_API_KEY is required")
		return 2
	}

	listingClient := deps.NewListingClient()
	openaiClient, err := deps.NewOpenAIClient(cfg)
	if err != nil {
		log.Printf("openai client: %v", err)
		return 2
	}
	previewUsecase := applisting.NewPreviewUsecase(listingClient, openaiClient)

	ctx := context.Background()
	preview, err := previewUsecase.Execute(ctx, ref)
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
