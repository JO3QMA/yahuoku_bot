package auction

import (
	"context"
	"errors"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"
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
	spec *spec.Spec
	err  error
}

func (f *fakeExtractor) ExtractSpec(ctx context.Context, title, description string) (*spec.Spec, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.spec, nil
}

func TestPreviewUsecase_Execute_success(t *testing.T) {
	end := time.Now().Add(time.Hour)
	ff := &fakeFetcher{data: &infraauction.AuctionData{
		AuctionID: "a1", Title: "T", CurrentPrice: 500,
		Status: "S", Images: []string{"i"}, Description: "d", EndTime: &end,
	}}
	fe := &fakeExtractor{spec: &spec.Spec{CPUModelLine: "cpu"}}
	u := NewPreviewUsecase(ff, fe)
	out, err := u.Execute(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	if out.URL != "https://page.auctions.yahoo.co.jp/jp/auction/a1" {
		t.Fatalf("url %s", out.URL)
	}
	if out.Spec.CPUModelLine != "cpu" {
		t.Fatal("spec missing")
	}
}

func TestPreviewUsecase_Execute_fetchError(t *testing.T) {
	u := NewPreviewUsecase(&fakeFetcher{err: errors.New("e")}, &fakeExtractor{})
	_, err := u.Execute(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPreviewUsecase_Execute_specFallback(t *testing.T) {
	ff := &fakeFetcher{data: &infraauction.AuctionData{AuctionID: "a", Title: "t"}}
	fe := &fakeExtractor{err: errors.New("extract fail")}
	u := NewPreviewUsecase(ff, fe)
	out, err := u.Execute(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	if out.Spec == nil {
		t.Fatal("expected empty spec object")
	}
}
