package watch

import (
	"context"
	"errors"
	"testing"
	"time"

	dwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/sqlite"
)

func TestGroupByAuctionID(t *testing.T) {
	items := []*dwatch.WatchItem{
		{AuctionID: "a"}, {AuctionID: "a"}, {AuctionID: "b"},
	}
	g := groupByAuctionID(items)
	if len(g) != 2 || len(g["a"]) != 2 || len(g["b"]) != 1 {
		t.Fatalf("%v", g)
	}
}

type errRepo struct{}

func (errRepo) Add(context.Context, *dwatch.WatchItem) error { return nil }
func (errRepo) Remove(context.Context, string, string, string) error { return nil }
func (errRepo) ListActive(context.Context) ([]*dwatch.WatchItem, error) {
	return nil, errors.New("list")
}
func (errRepo) UpdatePrice(context.Context, int64, int64) error { return nil }
func (errRepo) MarkReminded(context.Context, int64) error      { return nil }
func (errRepo) UpdateThreadID(context.Context, string, string) error {
	return nil
}
func (errRepo) FindByMessage(context.Context, string) ([]*dwatch.WatchItem, error) {
	return nil, nil
}
func (errRepo) RemoveByAuctionID(context.Context, string) error { return nil }

func TestPollingWorker_poll_listError(t *testing.T) {
	w := NewPollingWorker(errRepo{}, &stubFetch{}, &noopN{}, 60, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.poll(ctx)
}

func TestPollingWorker_poll_empty(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewWatchRepository(db)
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	w.poll(context.Background())
}

func TestPollingWorker_poll_getAuctionErr(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewWatchRepository(db)
	end := time.Now().Add(time.Hour)
	_ = repo.Add(context.Background(), &dwatch.WatchItem{
		AuctionID: "x", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1, EndTime: &end,
	})
	w := NewPollingWorker(repo, &errFetch{}, &noopN{}, 60, 1)
	w.poll(context.Background())
}

func TestPollingWorker_processGroup_canceled(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewWatchRepository(db)
	end := time.Now().Add(time.Hour)
	_ = repo.Add(context.Background(), &dwatch.WatchItem{
		AuctionID: "x", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1, EndTime: &end,
	})
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	items := []*dwatch.WatchItem{{AuctionID: "x", LastKnownPrice: 1}}
	w.processGroup(ctx, items, &auction.AuctionData{
		AuctionID: "x", CurrentPrice: 2, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
}

func TestPollingWorker_processGroup_canceledStatus(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewWatchRepository(db)
	end := time.Now().Add(time.Hour)
	_ = repo.Add(context.Background(), &dwatch.WatchItem{
		AuctionID: "x", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1, EndTime: &end,
	})
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "x", LastKnownPrice: 1, Reminded: false}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "x", Status: "AUCTION_STATUS_CANCELED", CurrentPrice: 1, EndTime: &end,
	})
}

func TestPollingWorker_processGroup_notifyPriceErr(t *testing.T) {
	end := time.Now().Add(time.Hour)
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &errN{}, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 9, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
}

func TestPollingWorker_processGroup_updatePriceErr(t *testing.T) {
	end := time.Now().Add(time.Hour)
	repo := &memRepoPoll{upErr: errors.New("u")}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 9, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
}

func TestPollingWorker_processGroup_endingNotifyErr(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &errN{}, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1, Reminded: false}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 1, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
}

func TestPollingWorker_processGroup_markRemindedErr(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{mrErr: errors.New("m")}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1, Reminded: false}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 1, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
}

func TestPollingWorker_processGroup_removeByAuctionErr(t *testing.T) {
	end := time.Now()
	repo := &memRepoPoll{rmAuctionErr: errors.New("r")}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.WatchItem{{AuctionID: "a"}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", Status: "AUCTION_STATUS_FINISHED", EndTime: &end,
	})
}

func TestPollingWorker_poll_ctxCanceledBeforeGetAuction(t *testing.T) {
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.poll(ctx)
}

func TestPollingWorker_poll_cancelInDelay(t *testing.T) {
	repo := &memRepoTwoAuction{}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 40)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(8 * time.Millisecond)
		cancel()
	}()
	w.poll(ctx)
}

func TestPollingWorker_Start_tickerTick(t *testing.T) {
	repo := &memRepoTwoAuction{}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 0, WithPollInterval(15*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()
	time.Sleep(45 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

type memRepoTwoAuction struct{}

func (memRepoTwoAuction) Add(context.Context, *dwatch.WatchItem) error { return nil }
func (memRepoTwoAuction) Remove(context.Context, string, string, string) error { return nil }
func (memRepoTwoAuction) ListActive(context.Context) ([]*dwatch.WatchItem, error) {
	e := time.Now().Add(time.Hour)
	return []*dwatch.WatchItem{
		{ID: 1, AuctionID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m1", LastKnownPrice: 1, EndTime: &e},
		{ID: 2, AuctionID: "b", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m2", LastKnownPrice: 1, EndTime: &e},
	}, nil
}
func (memRepoTwoAuction) UpdatePrice(context.Context, int64, int64) error { return nil }
func (memRepoTwoAuction) MarkReminded(context.Context, int64) error { return nil }
func (memRepoTwoAuction) UpdateThreadID(context.Context, string, string) error { return nil }
func (memRepoTwoAuction) FindByMessage(context.Context, string) ([]*dwatch.WatchItem, error) {
	return nil, nil
}
func (memRepoTwoAuction) RemoveByAuctionID(context.Context, string) error { return nil }

type stubFetch struct{}

func (stubFetch) GetAuction(_ context.Context, auctionID string) (*auction.AuctionData, error) {
	e := time.Now().Add(time.Hour)
	return &auction.AuctionData{
		AuctionID: auctionID, CurrentPrice: 100, Status: "AUCTION_STATUS_ACTIVE", EndTime: &e,
	}, nil
}

type errFetch struct{}

func (errFetch) GetAuction(context.Context, string) (*auction.AuctionData, error) {
	return nil, errors.New("e")
}

type noopN struct{}

func (noopN) NotifyPriceIncrease(context.Context, *dwatch.WatchItem, int64, int64, string) error {
	return nil
}
func (noopN) NotifyEndingSoon(context.Context, *dwatch.WatchItem, int64, string, time.Duration) error { return nil }

type errN struct{}

func (errN) NotifyPriceIncrease(context.Context, *dwatch.WatchItem, int64, int64, string) error {
	return errors.New("n")
}
func (errN) NotifyEndingSoon(context.Context, *dwatch.WatchItem, int64, string, time.Duration) error {
	return errors.New("n")
}

type memRepoPoll struct {
	upErr          error
	mrErr          error
	rmAuctionErr   error
	markedReminded []int64
	mrFailUntil    int // 最初の N 回 MarkReminded を失敗させる（リトライテスト用）
}

func (m *memRepoPoll) Add(context.Context, *dwatch.WatchItem) error { return nil }
func (m *memRepoPoll) Remove(context.Context, string, string, string) error {
	return nil
}
func (m *memRepoPoll) ListActive(context.Context) ([]*dwatch.WatchItem, error) {
	end := time.Now().Add(time.Hour)
	return []*dwatch.WatchItem{{
		ID: 1, AuctionID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1, EndTime: &end,
	}}, nil
}
func (m *memRepoPoll) UpdatePrice(context.Context, int64, int64) error { return m.upErr }
func (m *memRepoPoll) MarkReminded(_ context.Context, id int64) error {
	m.markedReminded = append(m.markedReminded, id)
	if m.mrFailUntil > 0 {
		m.mrFailUntil--
		return errors.New("transient")
	}
	return m.mrErr
}
func (m *memRepoPoll) UpdateThreadID(context.Context, string, string) error {
	return nil
}
func (m *memRepoPoll) FindByMessage(context.Context, string) ([]*dwatch.WatchItem, error) {
	return nil, nil
}
func (m *memRepoPoll) RemoveByAuctionID(context.Context, string) error { return m.rmAuctionErr }

func TestPollingWorker_processGroup_endingNotifyErr_noMarkReminded(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &errN{}, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 1, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
	if len(repo.markedReminded) != 0 {
		t.Fatalf("expected no MarkReminded, got %v", repo.markedReminded)
	}
}

func TestPollingWorker_processGroup_fallbackItemEndTime(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{}
	notifier := &trackEndingN{}
	w := NewPollingWorker(repo, &stubFetch{}, notifier, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 1, Status: "AUCTION_STATUS_ACTIVE", EndTime: nil,
	})
	if !notifier.called {
		t.Fatal("expected ending soon notification via item.EndTime fallback")
	}
	if len(repo.markedReminded) != 1 {
		t.Fatalf("expected MarkReminded once, got %v", repo.markedReminded)
	}
}

func TestPollingWorker_processGroup_triggerWindowWithLongInterval(t *testing.T) {
	end := time.Now().Add(12 * time.Minute)
	repo := &memRepoPoll{}
	notifier := &trackEndingN{}
	w := NewPollingWorker(repo, &stubFetch{}, notifier, 60, 1, WithPollInterval(15*time.Minute))
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 1, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
	if !notifier.called {
		t.Fatal("expected ending soon within extended trigger window")
	}
}

func TestPollingWorker_processGroup_noEarlyTrigger(t *testing.T) {
	end := time.Now().Add(20 * time.Minute)
	repo := &memRepoPoll{}
	notifier := &trackEndingN{}
	w := NewPollingWorker(repo, &stubFetch{}, notifier, 60, 1, WithPollInterval(15*time.Minute))
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 1, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
	if notifier.called {
		t.Fatal("expected no ending soon notification before threshold window")
	}
	if len(repo.markedReminded) != 0 {
		t.Fatalf("expected no MarkReminded, got %v", repo.markedReminded)
	}
}

func TestPollingWorker_processGroup_markRemindedRetry(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{mrFailUntil: 2}
	notifier := &trackEndingN{}
	w := NewPollingWorker(repo, &stubFetch{}, notifier, 60, 1)
	items := []*dwatch.WatchItem{{ID: 1, AuctionID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, &auction.AuctionData{
		AuctionID: "a", CurrentPrice: 1, Status: "AUCTION_STATUS_ACTIVE", EndTime: &end,
	})
	if !notifier.called {
		t.Fatal("expected ending soon notification")
	}
	if len(repo.markedReminded) != 3 {
		t.Fatalf("expected 3 MarkReminded attempts, got %d", len(repo.markedReminded))
	}
}

func TestShouldNotifyEndingSoon(t *testing.T) {
	threshold := 10 * time.Minute
	interval := 15 * time.Minute
	cases := []struct {
		remaining time.Duration
		want      bool
	}{
		{5 * time.Minute, true},
		{10 * time.Minute, true},
		{12 * time.Minute, true},  // next poll after end
		{20 * time.Minute, false}, // next poll at 5min, still in window
		{0, false},
		{-1 * time.Minute, false},
	}
	for _, tc := range cases {
		got := shouldNotifyEndingSoon(tc.remaining, interval, threshold)
		if got != tc.want {
			t.Errorf("remaining=%v: got %v, want %v", tc.remaining, got, tc.want)
		}
	}
}

type trackEndingN struct {
	called bool
}

func (trackEndingN) NotifyPriceIncrease(context.Context, *dwatch.WatchItem, int64, int64, string) error {
	return nil
}
func (t *trackEndingN) NotifyEndingSoon(context.Context, *dwatch.WatchItem, int64, string, time.Duration) error {
	t.called = true
	return nil
}
