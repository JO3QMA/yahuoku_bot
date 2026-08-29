package listing

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jo3qma/sansai"

	domainlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/openai"
)

// SansaiGetItem は sansai.Get の差し替え用（テストのみ）。
var SansaiGetItem = sansai.Get

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
	extractor openai.Client
}

// NewPreviewUsecase は PreviewUsecase を生成する。
func NewPreviewUsecase(ex openai.Client) *PreviewUsecase {
	return &PreviewUsecase{extractor: ex}
}

// Execute は Listing 参照から Preview を取得する。
func (u *PreviewUsecase) Execute(ctx context.Context, ref domainlisting.Ref) (*Preview, error) {
	item, err := SansaiGetItem(ctx, sansai.Market(ref.Market), ref.ListingID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("listing not found: %s/%s", ref.Market, ref.ListingID)
	}
	return u.previewFromData(ctx, ref, domainlisting.FromSansaiItem(item))
}

func (u *PreviewUsecase) previewFromData(ctx context.Context, ref domainlisting.Ref, data *domainlisting.Data) (*Preview, error) {
	productData, err := u.extractor.Extract(ctx, product.ExtractInput{
		Title:       data.Title,
		Description: data.Description,
		ImageURLs:   data.ImageURLs,
	})
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
