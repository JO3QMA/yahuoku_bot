package listing

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jo3qma/sansai"

	domainlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
)

// Client は C2C マーケットから Listing を取得する。
type Client interface {
	Get(ctx context.Context, ref domainlisting.Ref) (*domainlisting.Data, error)
}

type client struct{}

// NewClient は sansai を使う Client を返す。
func NewClient() Client {
	return &client{}
}

func (c *client) Get(ctx context.Context, ref domainlisting.Ref) (*domainlisting.Data, error) {
	item, err := sansai.Get(ctx, sansai.Market(ref.Market), ref.ListingID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("listing not found: %s/%s", ref.Market, ref.ListingID)
	}
	return itemToData(item), nil
}

func itemToData(item *sansai.Item) *domainlisting.Data {
	if item == nil {
		return nil
	}
	images := item.ImageURLs
	if len(images) == 0 && item.ImageURL != "" {
		images = []string{item.ImageURL}
	}
	var endTime *time.Time
	if item.EndTime != "" {
		t, err := time.Parse(time.RFC3339, item.EndTime)
		if err != nil {
			log.Printf("[listing] parse end_time %q for %s/%s: %v", item.EndTime, item.Market, item.ID, err)
		} else {
			endTime = &t
		}
	}
	return &domainlisting.Data{
		Ref: domainlisting.Ref{
			Market:    domainlisting.Market(item.Market),
			ListingID: item.ID,
		},
		Title:       item.Title,
		URL:         item.URL,
		Price:       int64(item.Price),
		Status:      item.Status,
		Description: item.Description,
		ImageURLs:   images,
		SaleType:    domainlisting.SaleType(item.SaleType),
		EndTime:     endTime,
		IsActive:    item.IsActive,
	}
}
