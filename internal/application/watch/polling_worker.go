package watch

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jo3qma/sansai"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// SansaiGetItem は sansai.Get の差し替え用（テストのみ）。
var SansaiGetItem = sansai.Get

const reminderThreshold = 10 * time.Minute

// PollingWorker は Watch リストを定期的にポーリングし、通知条件を判定するワーカー。
type PollingWorker struct {
	repo     watch.Repository
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
func NewPollingWorker(repo watch.Repository, notifier Notifier, intervalMinutes, delayMs int, opts ...PollingOption) *PollingWorker {
	w := &PollingWorker{
		repo:     repo,
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

	grouped := groupByListing(items)

	for ref, group := range grouped {
		if ctx.Err() != nil {
			return
		}

		item, err := SansaiGetItem(ctx, sansai.Market(ref.Market), ref.ListingID)
		if err != nil {
			log.Printf("[PollingWorker] Get %s/%s: %v", ref.Market, ref.ListingID, err)
			continue
		}
		if item == nil {
			log.Printf("[PollingWorker] Get %s/%s: listing not found", ref.Market, ref.ListingID)
			continue
		}
		data := listing.FromSansaiItem(item)

		w.processGroup(ctx, group, data)

		select {
		case <-ctx.Done():
			return
		case <-time.After(w.delay):
		}
	}
}

func (w *PollingWorker) processGroup(ctx context.Context, items []*watch.Watch, data *listing.Data) {
	if !data.IsActive {
		log.Printf("[PollingWorker] listing %s/%s inactive, removing all watchers", data.Ref.Market, data.Ref.ListingID)
		if err := w.repo.RemoveByListing(ctx, data.Ref.Market, data.Ref.ListingID); err != nil {
			log.Printf("[PollingWorker] remove by listing: %v", err)
		}
		return
	}

	now := time.Now()

	for _, item := range items {
		if data.Price != item.LastKnownPrice {
			if err := w.notifier.NotifyPriceAlert(ctx, item, item.LastKnownPrice, data.Price, data.Title); err != nil {
				log.Printf("[PollingWorker] notify price alert: %v", err)
			}
			if err := w.repo.UpdatePrice(ctx, item.ID, data.Price); err != nil {
				log.Printf("[PollingWorker] update price: %v", err)
			}
		}

		if data.IsAuction() {
			endTime := effectiveEndTime(data, item)
			if !item.Reminded && endTime != nil {
				remaining := endTime.Sub(now)
				if shouldSendEndingReminder(remaining, w.interval, reminderThreshold) {
					if err := w.notifier.NotifyEndingReminder(ctx, item, data.Price, data.Title, remaining); err != nil {
						log.Printf("[PollingWorker] notify ending reminder: %v", err)
						continue
					}
					if err := w.markRemindedWithRetry(ctx, item.ID); err != nil {
						log.Printf("[PollingWorker] %v", err)
					}
				}
			}
		}
	}
}

func shouldSendEndingReminder(remaining, interval, threshold time.Duration) bool {
	if remaining <= 0 {
		return false
	}
	if remaining <= threshold {
		return true
	}
	return remaining-interval <= 0
}

const markRemindedAttempts = 3

func (w *PollingWorker) markRemindedWithRetry(ctx context.Context, itemID int64) error {
	var err error
	for attempt := 0; attempt < markRemindedAttempts; attempt++ {
		if err = w.repo.MarkReminded(ctx, itemID); err == nil {
			return nil
		}
	}
	return fmt.Errorf("mark reminded failed after %d attempts: %w", markRemindedAttempts, err)
}

func effectiveEndTime(data *listing.Data, item *watch.Watch) *time.Time {
	if data.EndTime != nil {
		return data.EndTime
	}
	return item.EndTime
}

func groupByListing(items []*watch.Watch) map[listing.Ref][]*watch.Watch {
	grouped := make(map[listing.Ref][]*watch.Watch)
	for _, item := range items {
		ref := listing.Ref{Market: item.Market, ListingID: item.ListingID}
		grouped[ref] = append(grouped[ref], item)
	}
	return grouped
}
