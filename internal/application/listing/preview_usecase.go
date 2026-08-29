package listing

import (
	"context"
	"log"
	"time"

	domainlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	infralisting "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

// Preview は Discord 表示用に Listing と Product を統合したデータ。
type Preview struct {
	Ref          domainlisting.Ref
	Title        string
	URL          string
	CurrentPrice int64
	Status       string
	SaleType     domainlisting.SaleType
	Images       []string
	EndTime      *time.Time
	Product      *product.Product
}

// PreviewUsecase は Listing 参照から Preview を取得するユースケース。
type PreviewUsecase struct {
	listingClient infralisting.Client
	extractor     openai.Client
}

// NewPreviewUsecase は PreviewUsecase を生成する。
func NewPreviewUsecase(lc infralisting.Client, ex openai.Client) *PreviewUsecase {
	return &PreviewUsecase{listingClient: lc, extractor: ex}
}

// Execute は Listing 参照から Preview を取得する。
func (u *PreviewUsecase) Execute(ctx context.Context, ref domainlisting.Ref) (*Preview, error) {
	data, err := u.listingClient.Get(ctx, ref)
	if err != nil {
		return nil, err
	}

	productData, err := u.extractor.Extract(ctx, data.Title, data.Description, data.ImageURLs)
	if err != nil {
		log.Printf("[listing] extraction failed for %s/%s: %v", ref.Market, ref.ListingID, err)
		if productData == nil {
			productData = product.EmptyProduct()
		}
	}

	return &Preview{
		Ref:          data.Ref,
		Title:        data.Title,
		URL:          data.URL,
		CurrentPrice: data.Price,
		Status:       data.Status,
		SaleType:     data.SaleType,
		Images:       data.ImageURLs,
		EndTime:      data.EndTime,
		Product:      productData,
	}, nil
}
