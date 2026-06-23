package market

import (
	"context"
	"errors"
	"testing"

	domainmarket "jo3qma.com/yahoo_auctions_bot/internal/domain/market"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

type fakeSold struct {
	prices []int64
	err    error
}

func (f *fakeSold) SearchSoldPrices(context.Context, product.Category, string, string, int) ([]int64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.prices, nil
}

type fakeWeb struct {
	est *domainmarket.MarketEstimate
	err error
}

func (f *fakeWeb) Estimate(context.Context, string, string, *product.Product, bool) (*domainmarket.MarketEstimate, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.est, nil
}

func gpuProduct(model string) *product.Product {
	return &product.Product{
		Category: product.CategoryGPU,
		Fields:   []product.Field{{Key: "model", Value: model}},
	}
}

func TestEstimateUsecase_APIPath(t *testing.T) {
	u := NewEstimateUsecase(&fakeSold{prices: []int64{8000, 9000, 10000, 11000, 12000}}, nil, Config{})
	est, err := u.Execute(context.Background(), "t", "d", gpuProduct("GTX 1080"))
	if err != nil {
		t.Fatal(err)
	}
	if est == nil || est.LowPrice != 9000 || est.HighPrice != 11000 {
		t.Fatalf("got %+v", est)
	}
	if est.Note == "" {
		t.Fatal("note empty")
	}
}

func TestEstimateUsecase_APITooFewFallsBackWeb(t *testing.T) {
	webEst := &domainmarket.MarketEstimate{LowPrice: 1, HighPrice: 2, Note: "Web"}
	u := NewEstimateUsecase(
		&fakeSold{prices: []int64{1000, 2000}},
		&fakeWeb{est: webEst},
		Config{},
	)
	est, err := u.Execute(context.Background(), "t", "d", gpuProduct("GTX 1080"))
	if err != nil {
		t.Fatal(err)
	}
	if est != webEst {
		t.Fatalf("got %+v", est)
	}
}

func TestEstimateUsecase_NoIdentityUsesWeb(t *testing.T) {
	webEst := &domainmarket.MarketEstimate{LowPrice: 3, HighPrice: 4, Note: "未特定"}
	u := NewEstimateUsecase(&fakeSold{prices: []int64{1, 2, 3, 4, 5}}, &fakeWeb{est: webEst}, Config{})
	p := &product.Product{Category: product.CategoryGPU}
	est, err := u.Execute(context.Background(), "t", "d", p)
	if err != nil {
		t.Fatal(err)
	}
	if est != webEst {
		t.Fatalf("got %+v", est)
	}
}

func TestEstimateUsecase_SoldError(t *testing.T) {
	u := NewEstimateUsecase(&fakeSold{err: errors.New("api")}, nil, Config{})
	_, err := u.Execute(context.Background(), "t", "d", gpuProduct("x"))
	if err == nil {
		t.Fatal("expected error")
	}
}
