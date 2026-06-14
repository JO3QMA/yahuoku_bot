package auction

import (
	"context"
	"errors"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
)

type fakeFetcher struct {
	data *infraauction.AuctionData
	err  error
}

func (f *fakeFetcher) GetAuction(ctx context.Context, auctionID string) (*infraauction.AuctionData, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

type fakeExtractor struct {
	product *product.Product
	err     error
}

func (f *fakeExtractor) Extract(ctx context.Context, in ExtractInput) (*product.Product, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.product, nil
}

func TestPreviewUsecase_Execute_success(t *testing.T) {
	end := time.Now().Add(time.Hour)
	ff := &fakeFetcher{data: &infraauction.AuctionData{
		AuctionID: "a1", Title: "T", CurrentPrice: 500,
		Status: "S", Images: []string{"i"}, Description: "d", EndTime: &end,
	}}
	fe := &fakeExtractor{product: &product.Product{
		Category: product.CategoryServer,
		Fields:   []product.Field{{Key: "cpu_model_line", Value: "cpu"}},
	}}
	u := NewPreviewUsecase(ff, fe)
	out, err := u.Execute(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	if out.URL != "https://page.auctions.yahoo.co.jp/jp/auction/a1" {
		t.Fatalf("url %s", out.URL)
	}
	if len(out.Product.Fields) != 1 || out.Product.Fields[0].Value != "cpu" {
		t.Fatal("product missing")
	}
}

func TestPreviewUsecase_Execute_fetchError(t *testing.T) {
	u := NewPreviewUsecase(&fakeFetcher{err: errors.New("e")}, &fakeExtractor{})
	_, err := u.Execute(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPreviewUsecase_Execute_extractErrorContinues(t *testing.T) {
	end := time.Now().Add(time.Hour)
	ff := &fakeFetcher{data: &infraauction.AuctionData{
		AuctionID: "a1", Title: "T", CurrentPrice: 500,
		Status: "S", Description: "d", EndTime: &end,
	}}
	u := NewPreviewUsecase(ff, &fakeExtractor{err: errors.New("extract")})
	out, err := u.Execute(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Product.Category != product.CategoryOther {
		t.Fatalf("got %s", out.Product.Category)
	}
}

func TestPreviewUsecase_Execute_partialExtractSuccess(t *testing.T) {
	end := time.Now().Add(time.Hour)
	ff := &fakeFetcher{data: &infraauction.AuctionData{
		AuctionID: "a1", Title: "T", CurrentPrice: 500,
		Status: "S", Description: "d", EndTime: &end,
	}}
	partial := &product.Product{
		Category: product.CategoryGPU,
		Fields:   []product.Field{{Key: "model", Value: "RTX 3080"}},
	}
	u := NewPreviewUsecase(ff, &fakeExtractorPartial{detail: partial, err: errors.New("stage3")})
	out, err := u.Execute(context.Background(), "a1")
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

func (f *fakeExtractorPartial) Extract(context.Context, ExtractInput) (*product.Product, error) {
	return f.detail, f.err
}
