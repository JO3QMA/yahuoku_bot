package watch

import (
	"context"
	"time"

	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// Notifier は Watch 通知を送信するインターフェース。Presentation 層が実装する。
type Notifier interface {
	NotifyPriceAlert(ctx context.Context, item *domainwatch.Watch, oldPrice, newPrice int64, title string) error
	// NotifyEndingReminder は EndingReminder を送信する。remaining は終了までの残り時間。
	NotifyEndingReminder(ctx context.Context, item *domainwatch.Watch, currentPrice int64, title string, remaining time.Duration) error
}
