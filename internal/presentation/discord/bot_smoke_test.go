package discord

import (
	"testing"
	"time"

	applisting "jo3qma.com/yahoo_auctions_bot/internal/application/listing"
	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	dlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func TestNewBot_smoke(t *testing.T) {
	end := time.Now()
	data := &dlisting.Data{
		Ref:         dlisting.Ref{Market: dlisting.MarketYahooAuction, ListingID: "a"},
		Title:       "T",
		Price:       1,
		Description: "d",
		EndTime:     &end,
		SaleType:    dlisting.SaleTypeAuction,
		IsActive:    true,
	}
	stubPreviewSansai(t, data, nil)
	stubReactionSansai(t, data, nil)
	stubPollingSansai(t)
	pu := applisting.NewPreviewUsecase(&stubProductExt{pd: &product.Product{}})
	repo := &memWatchRepo{}
	wu := appwatch.NewWatchUsecase(repo)
	_, err := NewBot("Bot unit-test-token.invalid", pu, NewAllowedFilter(nil, nil), wu, repo, BotConfig{
		CheckIntervalMinutes: 120,
		PollDelayMs:          9999,
	})
	if err != nil {
		t.Fatal(err)
	}
}
