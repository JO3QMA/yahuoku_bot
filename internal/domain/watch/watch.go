package watch

import "time"

// WatchItem は監視対象のオークション商品を表すドメインモデル。
type WatchItem struct {
	ID             int64
	AuctionID      string
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
