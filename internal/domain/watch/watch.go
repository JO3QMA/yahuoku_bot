package watch

import (
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
)

// Watch はユーザーが Listing を追跡する登録レコード。
type Watch struct {
	ID             int64
	Market         listing.Market
	ListingID      string
	UserID         string
	GuildID        string
	ChannelID      string
	MessageID      string
	LastKnownPrice int64
	EndTime        *time.Time
	Reminded       bool
	ThreadID       string
	CreatedAt      time.Time
}
