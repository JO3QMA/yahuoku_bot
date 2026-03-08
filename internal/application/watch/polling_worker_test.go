package watch_test

import (
	"context"
	"sync"
	"testing"
	"time"

	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/sqlite"
)

type mockNotifier struct {
	mu               sync.Mutex
	priceIncreases   []priceNotification
	endingSoonAlerts []endingNotification
}

type priceNotification struct {
	AuctionID string
	UserID    string
	OldPrice  int64
	NewPrice  int64
}

type endingNotification struct {
	AuctionID    string
	UserID       string
	CurrentPrice int64
}

func (m *mockNotifier) NotifyPriceIncrease(_ context.Context, item *domainwatch.WatchItem, oldPrice, newPrice int64, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.priceIncreases = append(m.priceIncreases, priceNotification{
		AuctionID: item.AuctionID, UserID: item.UserID, OldPrice: oldPrice, NewPrice: newPrice,
	})
	return nil
}

func (m *mockNotifier) NotifyEndingSoon(_ context.Context, item *domainwatch.WatchItem, currentPrice int64, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endingSoonAlerts = append(m.endingSoonAlerts, endingNotification{
		AuctionID: item.AuctionID, UserID: item.UserID, CurrentPrice: currentPrice,
	})
	return nil
}

type mockFetcher struct {
	data map[string]*auction.AuctionData
}

func (m *mockFetcher) GetAuction(_ context.Context, auctionID string) (*auction.AuctionData, error) {
	return m.data[auctionID], nil
}

func setupTestRepo(t *testing.T) domainwatch.Repository {
	t.Helper()
	db, err := sqlite.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlite.NewWatchRepository(db)
}

func TestPollingWorker_PriceIncrease(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	endTime := time.Now().Add(2 * time.Hour)
	if err := repo.Add(ctx, &domainwatch.WatchItem{
		AuctionID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000, EndTime: &endTime,
	}); err != nil {
		t.Fatalf("repo.Add: %v", err)
	}

	notifier := &mockNotifier{}
	fetcher := &mockFetcher{data: map[string]*auction.AuctionData{
		"auc1": {
			AuctionID: "auc1", Title: "Test Item", CurrentPrice: 2000,
			Status: "AUCTION_STATUS_ACTIVE", EndTime: &endTime,
		},
	}}

	worker := appwatch.NewPollingWorker(repo, fetcher, notifier, 1, 0)

	pollCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		worker.Start(pollCtx)
		close(done)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	if len(notifier.priceIncreases) != 1 {
		t.Fatalf("expected 1 price increase notification, got %d", len(notifier.priceIncreases))
	}
	n := notifier.priceIncreases[0]
	if n.OldPrice != 1000 || n.NewPrice != 2000 {
		t.Errorf("price notification: old=%d new=%d, want old=1000 new=2000", n.OldPrice, n.NewPrice)
	}
}

func TestPollingWorker_EndingSoon(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	endTime := time.Now().Add(5 * time.Minute)
	if err := repo.Add(ctx, &domainwatch.WatchItem{
		AuctionID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000, EndTime: &endTime,
	}); err != nil {
		t.Fatalf("repo.Add: %v", err)
	}

	notifier := &mockNotifier{}
	fetcher := &mockFetcher{data: map[string]*auction.AuctionData{
		"auc1": {
			AuctionID: "auc1", Title: "Test Item", CurrentPrice: 1000,
			Status: "AUCTION_STATUS_ACTIVE", EndTime: &endTime,
		},
	}}

	worker := appwatch.NewPollingWorker(repo, fetcher, notifier, 1, 0)

	pollCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		worker.Start(pollCtx)
		close(done)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	if len(notifier.endingSoonAlerts) != 1 {
		t.Fatalf("expected 1 ending soon notification, got %d", len(notifier.endingSoonAlerts))
	}
}

func TestPollingWorker_FinishedAuctionCleanup(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	endTime := time.Now().Add(-1 * time.Hour)
	if err := repo.Add(ctx, &domainwatch.WatchItem{
		AuctionID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000, EndTime: &endTime,
	}); err != nil {
		t.Fatalf("repo.Add: %v", err)
	}

	notifier := &mockNotifier{}
	fetcher := &mockFetcher{data: map[string]*auction.AuctionData{
		"auc1": {
			AuctionID: "auc1", Title: "Test Item", CurrentPrice: 1500,
			Status: "AUCTION_STATUS_FINISHED", EndTime: &endTime,
		},
	}}

	worker := appwatch.NewPollingWorker(repo, fetcher, notifier, 1, 0)

	pollCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		worker.Start(pollCtx)
		close(done)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()
	<-done

	items, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items after finished cleanup, got %d", len(items))
	}
}
