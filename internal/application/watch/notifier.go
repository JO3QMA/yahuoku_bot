package watch

import (
	"context"
	"time"

	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// Notifier は監視通知を送信するインターフェース。Presentation層が実装する。
type Notifier interface {
	NotifyPriceIncrease(ctx context.Context, item *domainwatch.WatchItem, oldPrice, newPrice int64, title string) error
	// NotifyEndingSoon は終了間近通知を送信する。remaining は終了までの残り時間。
	NotifyEndingSoon(ctx context.Context, item *domainwatch.WatchItem, currentPrice int64, title string, remaining time.Duration) error
}
