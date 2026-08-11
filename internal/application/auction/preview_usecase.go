package auction

import (
	"context"
	"log"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

// Preview は Discord 表示用に Auction と Product を統合したデータ。
type Preview struct {
	AuctionID    string
	Title        string
	URL          string
	CurrentPrice int64
	Status       string
	Images       []string
	EndTime      *time.Time
	Product      *product.Product
}

// PreviewUsecase は Auction URL から Preview を取得するユースケース。
type PreviewUsecase struct {
	auctionClient auction.Client
	extractor     openai.Client
}

// NewPreviewUsecase は PreviewUsecase を生成する。
func NewPreviewUsecase(ac auction.Client, ex openai.Client) *PreviewUsecase {
	return &PreviewUsecase{auctionClient: ac, extractor: ex}
}

// Execute は auctionID から Preview を取得する。
func (u *PreviewUsecase) Execute(ctx context.Context, auctionID string) (*Preview, error) {
	data, err := u.auctionClient.GetAuction(ctx, auctionID)
	if err != nil {
		return nil, err
	}

	productData, err := u.extractor.Extract(ctx, product.ExtractInput{
		Title:       data.Title,
		Description: data.Description,
		ImageURLs:   data.Images,
	})
	if err != nil {
		log.Printf("[yahoo_auctions_bot] extraction failed for %s: %v", auctionID, err)
		if productData == nil {
			productData = product.EmptyProduct()
		}
	}

	url := "https://page.auctions.yahoo.co.jp/jp/auction/" + auctionID
	return &Preview{
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
