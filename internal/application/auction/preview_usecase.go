package auction

import (
	"context"
	"log"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
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
	Product      *product.ProductDetail
}

// AuctionFetcher はオークション情報を取得するインターフェース。
type AuctionFetcher interface {
	GetAuction(ctx context.Context, auctionID string) (*auction.AuctionData, error)
}

// ProductExtractor は商品説明からジャンル別商品情報を抽出するインターフェース。
type ProductExtractor interface {
	ExtractProduct(ctx context.Context, title, description string) (*product.ProductDetail, error)
}

// PreviewUsecase はオークションURLからプレビュー情報を取得するユースケース。
type PreviewUsecase struct {
	auctionClient  AuctionFetcher
	productExtractor ProductExtractor
}

// NewPreviewUsecase はPreviewUsecaseを生成する。
func NewPreviewUsecase(ac AuctionFetcher, pe ProductExtractor) *PreviewUsecase {
	return &PreviewUsecase{auctionClient: ac, productExtractor: pe}
}

// Execute はオークションIDからプレビュー情報を取得する。
func (u *PreviewUsecase) Execute(ctx context.Context, auctionID string) (*AuctionPreview, error) {
	data, err := u.auctionClient.GetAuction(ctx, auctionID)
	if err != nil {
		return nil, err
	}

	productData, err := u.productExtractor.ExtractProduct(ctx, data.Title, data.Description)
	if err != nil {
		log.Printf("[yahoo_auctions_bot] product extract failed for %s: %v", auctionID, err)
		productData = product.EmptyProductDetail()
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
		Product:      productData,
	}, nil
}
