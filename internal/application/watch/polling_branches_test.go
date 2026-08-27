package watch

import (
	"context"
	"errors"
	"testing"
	"time"

	dlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	dwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

const testMarket = dlisting.MarketYahooAuction

func testRef(id string) dlisting.Ref {
	return dlisting.Ref{Market: testMarket, ListingID: id}
}

func testListingData(id string, price int64, active bool, end *time.Time) *dlisting.Data {
	return &dlisting.Data{
		Ref:      testRef(id),
		Price:    price,
		SaleType: dlisting.SaleTypeAuction,
		IsActive: active,
		EndTime:  end,
	}
}

func TestGroupByListing(t *testing.T) {
	items := []*dwatch.Watch{
		{Market: testMarket, ListingID: "a"},
		{Market: testMarket, ListingID: "a"},
		{Market: testMarket, ListingID: "b"},
	}
	g := groupByListing(items)
	if len(g) != 2 || len(g[testRef("a")]) != 2 || len(g[testRef("b")]) != 1 {
		t.Fatalf("%v", g)
	}
}

type errRepo struct{}

func (errRepo) Add(context.Context, *dwatch.Watch) error { return nil }
func (errRepo) Remove(context.Context, dlisting.Market, string, string, string) error {
	return nil
}
func (errRepo) ListActive(context.Context) ([]*dwatch.Watch, error) {
	return nil, errors.New("list")
}
func (errRepo) UpdatePrice(context.Context, int64, int64) error { return nil }
func (errRepo) MarkReminded(context.Context, int64) error      { return nil }
func (errRepo) UpdateThreadID(context.Context, string, string) error {
	return nil
}
func (errRepo) FindByMessage(context.Context, string) ([]*dwatch.Watch, error) {
	return nil, nil
}
func (errRepo) RemoveByListing(context.Context, dlisting.Market, string) error { return nil }

func TestPollingWorker_poll_listError(t *testing.T) {
	w := NewPollingWorker(errRepo{}, &stubFetch{}, &noopN{}, 60, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.poll(ctx)
}

type storeRepo struct {
	items  []*dwatch.Watch
	nextID int64
}

func newStoreRepo() *storeRepo {
	return &storeRepo{}
}

func (s *storeRepo) Add(_ context.Context, item *dwatch.Watch) error {
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

func (s *storeRepo) ListActive(context.Context) ([]*dwatch.Watch, error) {
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

func (s *storeRepo) FindByMessage(_ context.Context, messageID string) ([]*dwatch.Watch, error) {
	var out []*dwatch.Watch
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

func TestPollingWorker_poll_empty(t *testing.T) {
	repo := newStoreRepo()
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	w.poll(context.Background())
}

func TestPollingWorker_poll_getListingErr(t *testing.T) {
	repo := newStoreRepo()
	end := time.Now().Add(time.Hour)
	_ = repo.Add(context.Background(), &dwatch.Watch{
		Market: testMarket, ListingID: "x", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1, EndTime: &end,
	})
	w := NewPollingWorker(repo, &errFetch{}, &noopN{}, 60, 1)
	w.poll(context.Background())
}

func TestPollingWorker_processGroup_inactive(t *testing.T) {
	repo := newStoreRepo()
	end := time.Now().Add(time.Hour)
	_ = repo.Add(context.Background(), &dwatch.Watch{
		Market: testMarket, ListingID: "x", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1, EndTime: &end,
	})
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "x", LastKnownPrice: 1, Reminded: false}}
	w.processGroup(context.Background(), items, testListingData("x", 1, false, &end))
}

func TestPollingWorker_processGroup_notifyPriceErr(t *testing.T) {
	end := time.Now().Add(time.Hour)
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &errN{}, 60, 1)
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1}}
	w.processGroup(context.Background(), items, testListingData("a", 9, true, &end))
}

func TestPollingWorker_processGroup_updatePriceErr(t *testing.T) {
	end := time.Now().Add(time.Hour)
	repo := &memRepoPoll{upErr: errors.New("u")}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1}}
	w.processGroup(context.Background(), items, testListingData("a", 9, true, &end))
}

func TestPollingWorker_processGroup_endingNotifyErr(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &errN{}, 60, 1)
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1, Reminded: false}}
	w.processGroup(context.Background(), items, testListingData("a", 1, true, &end))
}

func TestPollingWorker_processGroup_markRemindedErr(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{mrErr: errors.New("m")}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1, Reminded: false}}
	w.processGroup(context.Background(), items, testListingData("a", 1, true, &end))
}

func TestPollingWorker_processGroup_removeByListingErr(t *testing.T) {
	end := time.Now()
	repo := &memRepoPoll{rmListingErr: errors.New("r")}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 1)
	items := []*dwatch.Watch{{Market: testMarket, ListingID: "a"}}
	w.processGroup(context.Background(), items, testListingData("a", 0, false, &end))
}

func TestPollingWorker_poll_ctxCanceledBeforeGet(t *testing.T) {
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.poll(ctx)
}

func TestPollingWorker_poll_cancelInDelay(t *testing.T) {
	repo := &memRepoTwoListing{}
	w := NewPollingWorker(repo, &stubFetch{}, &noopN{}, 60, 40)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(8 * time.Millisecond)
		cancel()
	}()
	w.poll(ctx)
}

func TestPollingWorker_Start_tickerTick(t *testing.T) {
	repo := &memRepoTwoListing{}
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

type memRepoTwoListing struct{}

func (memRepoTwoListing) Add(context.Context, *dwatch.Watch) error { return nil }
func (memRepoTwoListing) Remove(context.Context, dlisting.Market, string, string, string) error {
	return nil
}
func (memRepoTwoListing) ListActive(context.Context) ([]*dwatch.Watch, error) {
	e := time.Now().Add(time.Hour)
	return []*dwatch.Watch{
		{ID: 1, Market: testMarket, ListingID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m1", LastKnownPrice: 1, EndTime: &e},
		{ID: 2, Market: testMarket, ListingID: "b", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m2", LastKnownPrice: 1, EndTime: &e},
	}, nil
}
func (memRepoTwoListing) UpdatePrice(context.Context, int64, int64) error { return nil }
func (memRepoTwoListing) MarkReminded(context.Context, int64) error       { return nil }
func (memRepoTwoListing) UpdateThreadID(context.Context, string, string) error {
	return nil
}
func (memRepoTwoListing) FindByMessage(context.Context, string) ([]*dwatch.Watch, error) {
	return nil, nil
}
func (memRepoTwoListing) RemoveByListing(context.Context, dlisting.Market, string) error { return nil }

type stubFetch struct{}

func (stubFetch) Get(_ context.Context, ref dlisting.Ref) (*dlisting.Data, error) {
	e := time.Now().Add(time.Hour)
	return testListingData(ref.ListingID, 100, true, &e), nil
}

type errFetch struct{}

func (errFetch) Get(context.Context, dlisting.Ref) (*dlisting.Data, error) {
	return nil, errors.New("e")
}

type noopN struct{}

func (noopN) NotifyPriceAlert(context.Context, *dwatch.Watch, int64, int64, string) error {
	return nil
}
func (noopN) NotifyEndingReminder(context.Context, *dwatch.Watch, int64, string, time.Duration) error {
	return nil
}

type errN struct{}

func (errN) NotifyPriceAlert(context.Context, *dwatch.Watch, int64, int64, string) error {
	return errors.New("n")
}
func (errN) NotifyEndingReminder(context.Context, *dwatch.Watch, int64, string, time.Duration) error {
	return errors.New("n")
}

type memRepoPoll struct {
	upErr          error
	mrErr          error
	rmListingErr   error
	markedReminded []int64
	mrFailUntil    int
}

func (m *memRepoPoll) Add(context.Context, *dwatch.Watch) error { return nil }
func (m *memRepoPoll) Remove(context.Context, dlisting.Market, string, string, string) error {
	return nil
}
func (m *memRepoPoll) ListActive(context.Context) ([]*dwatch.Watch, error) {
	end := time.Now().Add(time.Hour)
	return []*dwatch.Watch{{
		ID: 1, Market: testMarket, ListingID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
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
func (m *memRepoPoll) UpdateThreadID(context.Context, string, string) error { return nil }
func (m *memRepoPoll) FindByMessage(context.Context, string) ([]*dwatch.Watch, error) {
	return nil, nil
}
func (m *memRepoPoll) RemoveByListing(context.Context, dlisting.Market, string) error {
	return m.rmListingErr
}

func TestPollingWorker_processGroup_endingNotifyErr_noMarkReminded(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{}
	w := NewPollingWorker(repo, &stubFetch{}, &errN{}, 60, 1)
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, testListingData("a", 1, true, &end))
	if len(repo.markedReminded) != 0 {
		t.Fatalf("expected no MarkReminded, got %v", repo.markedReminded)
	}
}

func TestPollingWorker_processGroup_fallbackItemEndTime(t *testing.T) {
	end := time.Now().Add(5 * time.Minute)
	repo := &memRepoPoll{}
	notifier := &trackEndingN{}
	w := NewPollingWorker(repo, &stubFetch{}, notifier, 60, 1)
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, testListingData("a", 1, true, nil))
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
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, testListingData("a", 1, true, &end))
	if !notifier.called {
		t.Fatal("expected ending soon within extended trigger window")
	}
}

func TestPollingWorker_processGroup_noEarlyTrigger(t *testing.T) {
	end := time.Now().Add(20 * time.Minute)
	repo := &memRepoPoll{}
	notifier := &trackEndingN{}
	w := NewPollingWorker(repo, &stubFetch{}, notifier, 60, 1, WithPollInterval(15*time.Minute))
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, testListingData("a", 1, true, &end))
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
	items := []*dwatch.Watch{{ID: 1, Market: testMarket, ListingID: "a", LastKnownPrice: 1, Reminded: false, EndTime: &end}}
	w.processGroup(context.Background(), items, testListingData("a", 1, true, &end))
	if !notifier.called {
		t.Fatal("expected ending soon notification")
	}
	if len(repo.markedReminded) != 3 {
		t.Fatalf("expected 3 MarkReminded attempts, got %d", len(repo.markedReminded))
	}
}

func TestShouldSendEndingReminder(t *testing.T) {
	threshold := 10 * time.Minute
	interval := 15 * time.Minute
	cases := []struct {
		remaining time.Duration
		want      bool
	}{
		{5 * time.Minute, true},
		{10 * time.Minute, true},
		{12 * time.Minute, true},
		{20 * time.Minute, false},
		{0, false},
		{-1 * time.Minute, false},
	}
	for _, tc := range cases {
		got := shouldSendEndingReminder(tc.remaining, interval, threshold)
		if got != tc.want {
			t.Errorf("remaining=%v: got %v, want %v", tc.remaining, got, tc.want)
		}
	}
}

type trackEndingN struct {
	called bool
}

func (trackEndingN) NotifyPriceAlert(context.Context, *dwatch.Watch, int64, int64, string) error {
	return nil
}
func (t *trackEndingN) NotifyEndingReminder(context.Context, *dwatch.Watch, int64, string, time.Duration) error {
	t.called = true
	return nil
}
