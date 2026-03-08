package watch

import "context"

// Repository は監視アイテムの永続化インターフェース。
type Repository interface {
	Add(ctx context.Context, item *WatchItem) error
	Remove(ctx context.Context, auctionID, userID, messageID string) error
	ListActive(ctx context.Context) ([]*WatchItem, error)
	UpdatePrice(ctx context.Context, id int64, newPrice int64) error
	MarkReminded(ctx context.Context, id int64) error
	UpdateThreadID(ctx context.Context, messageID, threadID string) error
	FindByMessage(ctx context.Context, messageID string) ([]*WatchItem, error)
	RemoveByAuctionID(ctx context.Context, auctionID string) error
}
