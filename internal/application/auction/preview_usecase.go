package auction

import (
	"context"
	"log"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
)

// AuctionPreview はEmbed表示用に統合されたオークション情報。
type AuctionPreview struct {
	AuctionID    string
	Title        string
	URL          string
	CurrentPrice int64
	Status       string
	Images       []string
	EndTime      *time.Time
	Spec         *spec.Spec
}

// AuctionFetcher はオークション情報を取得するインターフェース。
type AuctionFetcher interface {
	GetAuction(ctx context.Context, auctionID string) (*auction.AuctionData, error)
}

// SpecExtractor は商品説明からスペックを抽出するインターフェース。
type SpecExtractor interface {
	ExtractSpec(ctx context.Context, title, description string) (*spec.Spec, error)
}

// PreviewUsecase はオークションURLからプレビュー情報を取得するユースケース。
type PreviewUsecase struct {
	auctionClient AuctionFetcher
	specExtractor SpecExtractor
}

// NewPreviewUsecase はPreviewUsecaseを生成する。
func NewPreviewUsecase(ac AuctionFetcher, se SpecExtractor) *PreviewUsecase {
	return &PreviewUsecase{auctionClient: ac, specExtractor: se}
}

// Execute はオークションIDからプレビュー情報を取得する。
func (u *PreviewUsecase) Execute(ctx context.Context, auctionID string) (*AuctionPreview, error) {
	data, err := u.auctionClient.GetAuction(ctx, auctionID)
	if err != nil {
		return nil, err
	}

	specData, err := u.specExtractor.ExtractSpec(ctx, data.Title, data.Description)
	if err != nil {
		log.Printf("[yahoo_auctions_bot] spec extract failed for %s: %v", auctionID, err)
		specData = &spec.Spec{} // 解析失敗時は空のSpecで続行
	}

	url := "https://page.auctions.yahoo.co.jp/jp/auction/" + auctionID
	return &AuctionPreview{
		AuctionID:    data.AuctionID,
		Title:        data.Title,
		URL:          url,
		CurrentPrice: data.CurrentPrice,
		Status:       data.Status,
		Images:       data.Images,
		EndTime:      data.EndTime,
		Spec:         specData,
	}, nil
}
