package listing

import (
	"log"
	"time"

	"github.com/jo3qma/sansai"
)

// FromSansaiItem は sansai.Item を Data に変換する。
func FromSansaiItem(item *sansai.Item) *Data {
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
	return &Data{
		Ref: Ref{
			Market:    Market(item.Market),
			ListingID: item.ID,
		},
		Title:       item.Title,
		URL:         item.URL,
		Price:       int64(item.Price),
		Status:      item.Status,
		Description: item.Description,
		ImageURLs:   images,
		SaleType:    SaleType(item.SaleType),
		EndTime:     endTime,
		IsActive:    item.IsActive,
	}
}
