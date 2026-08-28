package listing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jo3qma/sansai"

	domainlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

type fakeExtractor struct {
	product *product.Product
	err     error
}

func (f *fakeExtractor) Extract(ctx context.Context, in product.ExtractInput) (*product.Product, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.product, nil
}

func TestPreviewUsecase_Execute_success(t *testing.T) {
	end := time.Now().Add(time.Hour)
	ref := domainlisting.Ref{Market: domainlisting.MarketYahooAuction, ListingID: "a1"}
	data := &domainlisting.Data{
		Ref: ref, Title: "T", Price: 500,
		URL: "https://auctions.yahoo.co.jp/jp/auction/a1",
		ImageURLs: []string{"i"}, Description: "d", EndTime: &end,
		SaleType: domainlisting.SaleTypeAuction, IsActive: true,
	}
	prev := SansaiGetItem
	SansaiGetItem = func(context.Context, sansai.Market, string) (*sansai.Item, error) {
		return &sansai.Item{
			Market: sansai.MarketYahooAuction, ID: "a1", Title: "T", Price: 500,
			URL: data.URL, ImageURLs: data.ImageURLs, Description: "d",
			SaleType: "auction", EndTime: end.Format(time.RFC3339), IsActive: true,
		}, nil
	}
	t.Cleanup(func() { SansaiGetItem = prev })

	fe := &fakeExtractor{product: &product.Product{
		Category: product.CategoryServer,
		Fields:   []product.Field{{Key: "cpu_model_line", Value: "cpu"}},
	}}
	u := NewPreviewUsecase(fe)
	out, err := u.Execute(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if out.URL != "https://auctions.yahoo.co.jp/jp/auction/a1" {
		t.Fatalf("url %s", out.URL)
	}
	if len(out.Product.Fields) != 1 || out.Product.Fields[0].Value != "cpu" {
		t.Fatal("product missing")
	}
}

func TestPreviewUsecase_Execute_fetchError(t *testing.T) {
	prev := SansaiGetItem
	SansaiGetItem = func(context.Context, sansai.Market, string) (*sansai.Item, error) {
		return nil, errors.New("e")
	}
	t.Cleanup(func() { SansaiGetItem = prev })

	u := NewPreviewUsecase(&fakeExtractor{})
	_, err := u.Execute(context.Background(), domainlisting.Ref{Market: domainlisting.MarketMercari, ListingID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPreviewUsecase_Execute_extractErrorContinues(t *testing.T) {
	end := time.Now().Add(time.Hour)
	ref := domainlisting.Ref{Market: domainlisting.MarketYahooAuction, ListingID: "a1"}
	prev := SansaiGetItem
	SansaiGetItem = func(context.Context, sansai.Market, string) (*sansai.Item, error) {
		return &sansai.Item{
			Market: sansai.MarketYahooAuction, ID: "a1", Title: "T", Price: 500,
			Description: "d", SaleType: "auction",
			EndTime: end.Format(time.RFC3339), IsActive: true,
		}, nil
	}
	t.Cleanup(func() { SansaiGetItem = prev })

	u := NewPreviewUsecase(&fakeExtractor{err: errors.New("extract")})
	out, err := u.Execute(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if out.Product.Category != product.CategoryOther {
		t.Fatalf("got %s", out.Product.Category)
	}
}

func TestPreviewUsecase_Execute_partialExtractSuccess(t *testing.T) {
	end := time.Now().Add(time.Hour)
	ref := domainlisting.Ref{Market: domainlisting.MarketYahooAuction, ListingID: "a1"}
	prev := SansaiGetItem
	SansaiGetItem = func(context.Context, sansai.Market, string) (*sansai.Item, error) {
		return &sansai.Item{
			Market: sansai.MarketYahooAuction, ID: "a1", Title: "T", Price: 500,
			Description: "d", SaleType: "auction",
			EndTime: end.Format(time.RFC3339), IsActive: true,
		}, nil
	}
	t.Cleanup(func() { SansaiGetItem = prev })

	partial := &product.Product{
		Category: product.CategoryGPU,
		Fields:   []product.Field{{Key: "model", Value: "RTX 3080"}},
	}
	u := NewPreviewUsecase(&fakeExtractorPartial{detail: partial, err: errors.New("stage3")})
	out, err := u.Execute(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if out.Product.Category != product.CategoryGPU {
		t.Fatalf("got %s", out.Product.Category)
	}
	if len(out.Product.Fields) != 1 || out.Product.Fields[0].Value != "RTX 3080" {
		t.Fatalf("product %+v", out.Product)
	}
}

type fakeExtractorPartial struct {
	detail *product.Product
	err    error
}

func (f *fakeExtractorPartial) Extract(context.Context, product.ExtractInput) (*product.Product, error) {
	return f.detail, f.err
}
