package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

// previewDeps は RunPreview の依存注入用。
type previewDeps struct {
	LoadConfig       func(path string) (*config.Config, error)
	NewSpecExtractor func(apiKey, model, baseURL string) (appauction.SpecExtractor, error)
	NewAuctionClient func(baseURL string) infraauction.Client
}

func mergePreviewDeps(d *previewDeps) {
	if d.LoadConfig == nil {
		d.LoadConfig = config.Load
	}
	if d.NewSpecExtractor == nil {
		d.NewSpecExtractor = func(k, m, u string) (appauction.SpecExtractor, error) {
			return openai.NewClient(k, m, u)
		}
	}
	if d.NewAuctionClient == nil {
		d.NewAuctionClient = func(baseURL string) infraauction.Client {
			return infraauction.NewClient(baseURL, (*http.Client)(nil))
		}
	}
}

// RunPreview はプレビューCLIの本体。終了コード: 0=成功、1=空Spec、2=引数エラー。
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
	if cfg.OpenAIAPIKey == "" {
		log.Print("OPENAI_API_KEY is required")
		return 2
	}

	auctionClient := deps.NewAuctionClient(cfg.APIEndpoint)
	specExtractor, err := deps.NewSpecExtractor(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIBaseURL)
	if err != nil {
		log.Printf("openai client: %v", err)
		return 2
	}
	previewUsecase := appauction.NewPreviewUsecase(auctionClient, specExtractor)

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

	if isSpecEmpty(preview.Spec) {
		log.Print("warning: Spec extraction returned no usable data")
		return 1
	}
	return 0
}

// isSpecEmpty は Spec が実質的に空（すべてゼロ値・空）かどうかを返す。
func isSpecEmpty(s *spec.Spec) bool {
	if s == nil {
		return true
	}
	return s.CPUModelLine == "" && s.CoreThreadInfo == "" && s.SocketCount == 0 &&
		s.MemoryInfo == "" && s.StorageType == "" && s.StorageCapacity == "" &&
		s.OtherNotes == "" && s.Condition == ""
}
