package watch

import (
	"context"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// WatchUsecase は Watch の登録/解除を行うユースケース。
type WatchUsecase struct {
	repo watch.Repository
}

// NewWatchUsecase は WatchUsecase を生成する。
func NewWatchUsecase(repo watch.Repository) *WatchUsecase {
	return &WatchUsecase{repo: repo}
}

// Register は Watch を登録する。
func (u *WatchUsecase) Register(
	ctx context.Context,
	market listing.Market,
	listingID, userID, guildID, channelID, messageID string,
	currentPrice int64,
	endTime *time.Time,
) error {
	item := &watch.Watch{
		Market:         market,
		ListingID:      listingID,
		UserID:         userID,
		GuildID:        guildID,
		ChannelID:      channelID,
		MessageID:      messageID,
		LastKnownPrice: currentPrice,
		EndTime:        endTime,
	}
	return u.repo.Add(ctx, item)
}

// Unregister は Watch を解除する。
func (u *WatchUsecase) Unregister(ctx context.Context, market listing.Market, listingID, userID, messageID string) error {
	return u.repo.Remove(ctx, market, listingID, userID, messageID)
}
