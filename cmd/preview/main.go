// プレビューCLI: オークションIDを引数に取り、取得結果とGemini解析スペックをJSONで標準出力する。
// 使用例: go run ./cmd/preview k1218678393
// 必要: .env に GEMINI_API_KEY、API_ENDPOINT（未設定時は http://localhost:8080）
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/config"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/gemini"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: preview <auction_id>")
	}
	auctionID := os.Args[1]

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("config load: %v", err)
	}
	if cfg.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY is required")
	}

	auctionClient := infraauction.NewClient(cfg.APIEndpoint, nil)
	geminiClient, err := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	if err != nil {
		log.Fatalf("gemini client: %v", err)
	}
	previewUsecase := appauction.NewPreviewUsecase(auctionClient, geminiClient)

	ctx := context.Background()
	preview, err := previewUsecase.Execute(ctx, auctionID)
	if err != nil {
		log.Fatalf("preview execute: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(preview); err != nil {
		log.Fatalf("encode: %v", err)
	}

	if isSpecEmpty(preview.Spec) {
		log.Print("warning: Spec extraction returned no usable data")
		os.Exit(1)
	}
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
