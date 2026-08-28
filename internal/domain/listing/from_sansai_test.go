package listing

import (
	"testing"
	"time"

	"github.com/jo3qma/sansai"
)

func TestFromSansaiItem(t *testing.T) {
	end := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	item := &sansai.Item{
		Market:      sansai.MarketYahooAuction,
		ID:          "a1",
		Title:       "T",
		Price:       500,
		URL:         "https://example.com/a1",
		ImageURL:    "https://example.com/img.jpg",
		Description: "desc",
		SaleType:    "auction",
		EndTime:     end.Format(time.RFC3339),
		Status:      "open",
		IsActive:    true,
	}
	data := FromSansaiItem(item)
	if data.Ref.Market != MarketYahooAuction || data.Ref.ListingID != "a1" {
		t.Fatalf("ref %+v", data.Ref)
	}
	if len(data.ImageURLs) != 1 || data.ImageURLs[0] != item.ImageURL {
		t.Fatalf("images %v", data.ImageURLs)
	}
	if data.EndTime == nil || !data.EndTime.Equal(end) {
		t.Fatalf("end_time %v", data.EndTime)
	}
	if data.Price != 500 || !data.IsActive {
		t.Fatalf("price/active %d %v", data.Price, data.IsActive)
	}
}

func TestFromSansaiItem_nil(t *testing.T) {
	if FromSansaiItem(nil) != nil {
		t.Fatal("expected nil")
	}
}
