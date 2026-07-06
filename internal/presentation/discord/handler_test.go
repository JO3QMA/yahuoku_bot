package discord

import (
	"context"
	"errors"
	"testing"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	appmarket "jo3qma.com/yahoo_auctions_bot/internal/application/market"
	domainmarket "jo3qma.com/yahoo_auctions_bot/internal/domain/market"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"

	"github.com/diamondburned/arikawa/v3/discord"
)

type handlerFakeWeb struct {
	est *domainmarket.MarketEstimate
	err error
}

func (f *handlerFakeWeb) Estimate(context.Context, string, string, *product.Product, bool) (*domainmarket.MarketEstimate, error) {
	return f.est, f.err
}

func gpuPreview() *appauction.Preview {
	return &appauction.Preview{
		AuctionID: "abc12345",
		Title:     "GTX 1080",
		Product: &product.Product{
			Category: product.CategoryGPU,
			Fields:   []product.Field{{Key: "model", Value: "GTX 1080"}},
		},
	}
}

func editCalled(api *stubSessionAPI) bool {
	return api.lastEdit.Embeds != nil && len(*api.lastEdit.Embeds) > 0
}

func Test_attachMarketEstimate_nilMarket(t *testing.T) {
	api := &stubSessionAPI{}
	h := NewHandler(nil, nil, NewEmbedBuilder(api), nil, 0)
	h.attachMarketEstimate(gpuPreview(), 1, 2)
	if editCalled(api) {
		t.Fatal("edit should not happen")
	}
}

func Test_attachMarketEstimate_nilResult(t *testing.T) {
	api := &stubSessionAPI{}
	market := appmarket.NewEstimateUsecase(nil, nil, appmarket.Config{})
	h := NewHandler(nil, market, NewEmbedBuilder(api), nil, 0)
	h.attachMarketEstimate(gpuPreview(), 1, 2)
	if editCalled(api) {
		t.Fatal("edit should not happen")
	}
}

func Test_attachMarketEstimate_error(t *testing.T) {
	api := &stubSessionAPI{}
	market := appmarket.NewEstimateUsecase(nil, &handlerFakeWeb{err: errors.New("fail")}, appmarket.Config{})
	h := NewHandler(nil, market, NewEmbedBuilder(api), nil, 0)
	h.attachMarketEstimate(gpuPreview(), 1, 2)
	if editCalled(api) {
		t.Fatal("edit should not happen on error")
	}
}

func Test_attachMarketEstimate_success(t *testing.T) {
	api := &stubSessionAPI{}
	est := &domainmarket.MarketEstimate{LowPrice: 8000, HighPrice: 12000, Note: "Web検索"}
	market := appmarket.NewEstimateUsecase(nil, &handlerFakeWeb{est: est}, appmarket.Config{})
	h := NewHandler(nil, market, NewEmbedBuilder(api), nil, 0)
	h.attachMarketEstimate(gpuPreview(), discord.MessageID(1), discord.ChannelID(2))
	if !editCalled(api) {
		t.Fatal("expected embed edit")
	}
}

func Test_attachMarketEstimate_editErrorNoPanic(t *testing.T) {
	api := &stubSessionAPI{editErr: errors.New("discord down")}
	est := &domainmarket.MarketEstimate{LowPrice: 1, HighPrice: 2, Note: "Web検索"}
	market := appmarket.NewEstimateUsecase(nil, &handlerFakeWeb{est: est}, appmarket.Config{})
	h := NewHandler(nil, market, NewEmbedBuilder(api), nil, 0)
	h.attachMarketEstimate(gpuPreview(), 1, 2)
}
