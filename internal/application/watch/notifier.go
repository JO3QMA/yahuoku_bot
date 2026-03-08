package watch

import (
	"context"

	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// Notifier は監視通知を送信するインターフェース。Presentation層が実装する。
type Notifier interface {
	NotifyPriceIncrease(ctx context.Context, item *domainwatch.WatchItem, oldPrice, newPrice int64, title string) error
	NotifyEndingSoon(ctx context.Context, item *domainwatch.WatchItem, currentPrice int64, title string) error
}
