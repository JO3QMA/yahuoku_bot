package discord

import (
	"testing"
	"time"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	infraauction "jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
)

func TestNewBot_smoke(t *testing.T) {
	end := time.Now()
	pu := appauction.NewPreviewUsecase(&stubPreviewFetch{data: &infraauction.AuctionData{
		AuctionID: "a", Title: "T", CurrentPrice: 1, Status: "S", Description: "d", EndTime: &end,
	}}, &stubProductExt{pd: &product.ProductDetail{}})
	repo := &memWatchRepo{}
	wu := appwatch.NewWatchUsecase(repo)
	ac := &stubAuction{data: &infraauction.AuctionData{CurrentPrice: 1}}
	_, err := NewBot("Bot unit-test-token.invalid", pu, NewAllowedFilter(nil, nil), wu, ac, repo, BotConfig{
		CheckIntervalMinutes: 120,
		PollDelayMs:          9999,
	})
	if err != nil {
		t.Fatal(err)
	}
}
