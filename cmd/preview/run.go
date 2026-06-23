package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	appmarket "jo3qma.com/yahoo_auctions_bot/internal/application/market"
	"jo3qma.com/yahoo_auctions_bot/internal/bootstrap"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/gemini"
)

// previewDeps は RunPreview の依存注入用。
type previewDeps struct {
	LoadConfig       func(path string) (*config.Config, error)
	NewGeminiClient  func(cfg *config.Config) (appauction.Extractor, error)
	NewAuctionClient func(baseURL string) infraauction.Client
	NewMarketUsecase func(cfg *config.Config) (*appmarket.EstimateUsecase, error)
}

func mergePreviewDeps(d *previewDeps) {
	if d.LoadConfig == nil {
		d.LoadConfig = config.Load
	}
	if d.NewGeminiClient == nil {
		d.NewGeminiClient = func(cfg *config.Config) (appauction.Extractor, error) {
			return gemini.NewClient(cfg.GeminiAPIKey, bootstrap.GeminiOptions(cfg))
		}
	}
	if d.NewAuctionClient == nil {
		d.NewAuctionClient = func(baseURL string) infraauction.Client {
			return infraauction.NewClient(baseURL, (*http.Client)(nil))
		}
	}
	if d.NewMarketUsecase == nil {
		d.NewMarketUsecase = bootstrap.MarketEstimateUsecase
	}
}

// RunPreview はプレビューCLIの本体。終了コード: 0=成功、1=空Product、2=引数エラー。
func RunPreview(stdout io.Writer, argv []string, cfgPath string, deps *previewDeps) int {
	if deps == nil {
		deps = &previewDeps{}
	}
	mergePreviewDeps(deps)

	if len(argv) < 1 {
		log.Print("usage: preview <auction_id>")
		return 2
	}
	auctionID := argv[0]

	cfg, err := deps.LoadConfig(cfgPath)
	if err != nil {
		log.Printf("config load: %v", err)
		return 2
	}
	if cfg.GeminiAPIKey == "" {
		log.Print("GEMINI_API_KEY is required")
		return 2
	}

	auctionClient := deps.NewAuctionClient(cfg.APIEndpoint)
	geminiClient, err := deps.NewGeminiClient(cfg)
	if err != nil {
		log.Printf("gemini client: %v", err)
		return 2
	}
	previewUsecase := appauction.NewPreviewUsecase(auctionClient, geminiClient)

	ctx := context.Background()
	preview, err := previewUsecase.Execute(ctx, auctionID)
	if err != nil {
		log.Printf("preview execute: %v", err)
		return 2
	}

	if marketUC, err := deps.NewMarketUsecase(cfg); err != nil {
		log.Printf("market estimate init: %v", err)
	} else if marketUC != nil {
		est, err := marketUC.Execute(ctx, preview.Title, preview.Description, preview.Product)
		if err != nil {
			log.Printf("market estimate: %v", err)
		} else {
			preview.MarketEstimate = est
		}
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
