package watch_test

import (
	"context"
	"sync"
	"testing"
	"time"

	appwatch "jo3qma.com/yahoo_auctions_bot/internal/application/watch"
	dlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

const testMarket = dlisting.MarketYahooAuction

type mockNotifier struct {
	mu               sync.Mutex
	priceIncreases   []priceNotification
	endingSoonAlerts []endingNotification
}

type priceNotification struct {
	ListingID string
	UserID    string
	OldPrice  int64
	NewPrice  int64
}

type endingNotification struct {
	ListingID    string
	UserID       string
	CurrentPrice int64
}

func (m *mockNotifier) NotifyPriceAlert(_ context.Context, item *domainwatch.Watch, oldPrice, newPrice int64, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.priceIncreases = append(m.priceIncreases, priceNotification{
		ListingID: item.ListingID, UserID: item.UserID, OldPrice: oldPrice, NewPrice: newPrice,
	})
	return nil
}

func (m *mockNotifier) NotifyEndingReminder(_ context.Context, item *domainwatch.Watch, currentPrice int64, _ string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endingSoonAlerts = append(m.endingSoonAlerts, endingNotification{
		ListingID: item.ListingID, UserID: item.UserID, CurrentPrice: currentPrice,
	})
	return nil
}

type mockFetcher struct {
	data map[string]*dlisting.Data
}

func (m *mockFetcher) Get(_ context.Context, ref dlisting.Ref) (*dlisting.Data, error) {
	return m.data[ref.ListingID], nil
}

func setupTestRepo(t *testing.T) domainwatch.Repository {
	t.Helper()
	return newStoreRepo()
}

type storeRepo struct {
	items  []*domainwatch.Watch
	nextID int64
}

func newStoreRepo() *storeRepo {
	return &storeRepo{}
}

func (s *storeRepo) Add(_ context.Context, item *domainwatch.Watch) error {
	s.nextID++
	cp := *item
	cp.ID = s.nextID
	s.items = append(s.items, &cp)
	return nil
}

func (s *storeRepo) Remove(_ context.Context, market dlisting.Market, listingID, userID, messageID string) error {
	out := s.items[:0]
	for _, it := range s.items {
		if it.Market == market && it.ListingID == listingID && it.UserID == userID && it.MessageID == messageID {
			continue
		}
		out = append(out, it)
	}
	s.items = out
	return nil
}

func (s *storeRepo) ListActive(context.Context) ([]*domainwatch.Watch, error) {
	return s.items, nil
}

func (s *storeRepo) UpdatePrice(_ context.Context, id int64, newPrice int64) error {
	for _, it := range s.items {
		if it.ID == id {
			it.LastKnownPrice = newPrice
			return nil
		}
	}
	return nil
}

func (s *storeRepo) MarkReminded(_ context.Context, id int64) error {
	for _, it := range s.items {
		if it.ID == id {
			it.Reminded = true
			return nil
		}
	}
	return nil
}

func (s *storeRepo) UpdateThreadID(_ context.Context, messageID, threadID string) error {
	for _, it := range s.items {
		if it.MessageID == messageID {
			it.ThreadID = threadID
		}
	}
	return nil
}

func (s *storeRepo) FindByMessage(_ context.Context, messageID string) ([]*domainwatch.Watch, error) {
	var out []*domainwatch.Watch
	for _, it := range s.items {
		if it.MessageID == messageID {
			out = append(out, it)
		}
	}
	return out, nil
}

func (s *storeRepo) RemoveByListing(_ context.Context, market dlisting.Market, listingID string) error {
	out := s.items[:0]
	for _, it := range s.items {
		if it.Market == market && it.ListingID == listingID {
			continue
		}
		out = append(out, it)
	}
	s.items = out
	return nil
}

func TestPollingWorker_PriceAlert(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	endTime := time.Now().Add(2 * time.Hour)
	if err := repo.Add(ctx, &domainwatch.Watch{
		Market: testMarket, ListingID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000, EndTime: &endTime,
	}); err != nil {
		t.Fatalf("repo.Add: %v", err)
	}

	notifier := &mockNotifier{}
	fetcher := &mockFetcher{data: map[string]*dlisting.Data{
		"auc1": {
			Ref: dlisting.Ref{Market: testMarket, ListingID: "auc1"},
			Title: "Test Item", Price: 2000,
			SaleType: dlisting.SaleTypeAuction, IsActive: true, EndTime: &endTime,
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

func TestPollingWorker_EndingReminder(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	endTime := time.Now().Add(5 * time.Minute)
	if err := repo.Add(ctx, &domainwatch.Watch{
		Market: testMarket, ListingID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000, EndTime: &endTime,
	}); err != nil {
		t.Fatalf("repo.Add: %v", err)
	}

	notifier := &mockNotifier{}
	fetcher := &mockFetcher{data: map[string]*dlisting.Data{
		"auc1": {
			Ref: dlisting.Ref{Market: testMarket, ListingID: "auc1"},
			Title: "Test Item", Price: 1000,
			SaleType: dlisting.SaleTypeAuction, IsActive: true, EndTime: &endTime,
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
	if err := repo.Add(ctx, &domainwatch.Watch{
		Market: testMarket, ListingID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000, EndTime: &endTime,
	}); err != nil {
		t.Fatalf("repo.Add: %v", err)
	}

	notifier := &mockNotifier{}
	fetcher := &mockFetcher{data: map[string]*dlisting.Data{
		"auc1": {
			Ref: dlisting.Ref{Market: testMarket, ListingID: "auc1"},
			Title: "Test Item", Price: 1500,
			SaleType: dlisting.SaleTypeAuction, IsActive: false, EndTime: &endTime,
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
