package watch

import (
	"context"
	"fmt"
	"log"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
)

const reminderThreshold = 10 * time.Minute

// PollingWorker は監視リストを定期的にポーリングし、通知条件を判定するワーカー。
type PollingWorker struct {
	repo     watch.Repository
	fetcher  auction.Client
	notifier Notifier
	interval time.Duration
	delay    time.Duration
}

// PollingOption は NewPollingWorker の挙動を上書きする（主にテスト用）。
type PollingOption func(*PollingWorker)

// WithPollInterval はティッカー間隔を上書きする（既定は intervalMinutes 分）。
func WithPollInterval(d time.Duration) PollingOption {
	return func(w *PollingWorker) { w.interval = d }
}

// NewPollingWorker はPollingWorkerを生成する。
func NewPollingWorker(repo watch.Repository, fetcher auction.Client, notifier Notifier, intervalMinutes, delayMs int, opts ...PollingOption) *PollingWorker {
	w := &PollingWorker{
		repo:     repo,
		fetcher:  fetcher,
		notifier: notifier,
		interval: time.Duration(intervalMinutes) * time.Minute,
		delay:    time.Duration(delayMs) * time.Millisecond,
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start はポーリングループを開始する。ctx がキャンセルされると終了する。
func (w *PollingWorker) Start(ctx context.Context) {
	log.Printf("[PollingWorker] started (interval=%v, delay=%v)", w.interval, w.delay)

	// 起動直後に1回実行
	w.poll(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[PollingWorker] stopped")
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *PollingWorker) poll(ctx context.Context) {
	items, err := w.repo.ListActive(ctx)
	if err != nil {
		log.Printf("[PollingWorker] list active: %v", err)
		return
	}

	if len(items) == 0 {
		return
	}

	log.Printf("[PollingWorker] polling %d items", len(items))

	// auction_id でグループ化し、同一商品への重複リクエストを避ける
	grouped := groupByAuctionID(items)

	for auctionID, group := range grouped {
		if ctx.Err() != nil {
			return
		}

		data, err := w.fetcher.GetAuction(ctx, auctionID)
		if err != nil {
			log.Printf("[PollingWorker] GetAuction %s: %v", auctionID, err)
			continue
		}

		w.processGroup(ctx, group, data)

		// 負荷軽減のためディレイ
		select {
		case <-ctx.Done():
			return
		case <-time.After(w.delay):
		}
	}
}

func (w *PollingWorker) processGroup(ctx context.Context, items []*watch.WatchItem, data *auction.AuctionData) {
	// 終了済みオークションは全監視を解除
	if data.Status == "AUCTION_STATUS_FINISHED" || data.Status == "AUCTION_STATUS_CANCELED" {
		log.Printf("[PollingWorker] auction %s ended (status=%s), removing all watchers", data.AuctionID, data.Status)
		if err := w.repo.RemoveByAuctionID(ctx, data.AuctionID); err != nil {
			log.Printf("[PollingWorker] remove by auction: %v", err)
		}
		return
	}

	now := time.Now()

	for _, item := range items {
		// 価格上昇チェック
		if data.CurrentPrice > item.LastKnownPrice {
			if err := w.notifier.NotifyPriceIncrease(ctx, item, item.LastKnownPrice, data.CurrentPrice, data.Title); err != nil {
				log.Printf("[PollingWorker] notify price increase: %v", err)
			}
			if err := w.repo.UpdatePrice(ctx, item.ID, data.CurrentPrice); err != nil {
				log.Printf("[PollingWorker] update price: %v", err)
			}
		}

		// 終了間近リマインドチェック
		endTime := effectiveEndTime(data, item)
		if !item.Reminded && endTime != nil {
			remaining := endTime.Sub(now)
			if shouldNotifyEndingSoon(remaining, w.interval, reminderThreshold) {
				if err := w.notifier.NotifyEndingSoon(ctx, item, data.CurrentPrice, data.Title, remaining); err != nil {
					log.Printf("[PollingWorker] notify ending soon: %v", err)
					continue
				}
				if err := w.markRemindedWithRetry(ctx, item.ID); err != nil {
					log.Printf("[PollingWorker] %v", err)
				}
			}
		}
	}
}

// shouldNotifyEndingSoon は終了間近リマインドを送るべきか判定する。
// 10分以内、または次回ポール前にオークションが終了する場合に true（取りこぼし防止）。
func shouldNotifyEndingSoon(remaining, interval, threshold time.Duration) bool {
	if remaining <= 0 {
		return false
	}
	if remaining <= threshold {
		return true
	}
	return remaining-interval <= 0
}

const markRemindedAttempts = 3

// markRemindedWithRetry は通知成功後に reminded フラグを設定する。DB 障害時は稀に重複通知の可能性あり。
func (w *PollingWorker) markRemindedWithRetry(ctx context.Context, itemID int64) error {
	var err error
	for attempt := 0; attempt < markRemindedAttempts; attempt++ {
		if err = w.repo.MarkReminded(ctx, itemID); err == nil {
			return nil
		}
	}
	return fmt.Errorf("mark reminded failed after %d attempts: %w", markRemindedAttempts, err)
}

// effectiveEndTime は API の終了時刻を優先し、なければ DB 保存値を使う。
func effectiveEndTime(data *auction.AuctionData, item *watch.WatchItem) *time.Time {
	if data.EndTime != nil {
		return data.EndTime
	}
	return item.EndTime
}

func groupByAuctionID(items []*watch.WatchItem) map[string][]*watch.WatchItem {
	grouped := make(map[string][]*watch.WatchItem)
	for _, item := range items {
		grouped[item.AuctionID] = append(grouped[item.AuctionID], item)
	}
	return grouped
}
