package watch

import "time"

// Watch はユーザーが Auction を追跡する登録レコード。
type Watch struct {
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
