package watch

import (
	"context"
	"errors"
	"testing"
	"time"

	dlisting "jo3qma.com/yahoo_auctions_bot/internal/domain/listing"
	domainwatch "jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

type memRepo struct {
	items  []*domainwatch.Watch
	addErr error
	remErr error
}

func (m *memRepo) Add(ctx context.Context, item *domainwatch.Watch) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.items = append(m.items, item)
	return nil
}

func (m *memRepo) Remove(ctx context.Context, market dlisting.Market, listingID, userID, messageID string) error {
	return m.remErr
}

func (m *memRepo) ListActive(ctx context.Context) ([]*domainwatch.Watch, error) {
	return nil, nil
}

func (m *memRepo) UpdatePrice(ctx context.Context, id int64, newPrice int64) error { return nil }
func (m *memRepo) MarkReminded(ctx context.Context, id int64) error              { return nil }
func (m *memRepo) UpdateThreadID(ctx context.Context, messageID, threadID string) error {
	return nil
}
func (m *memRepo) FindByMessage(ctx context.Context, messageID string) ([]*domainwatch.Watch, error) {
	return nil, nil
}
func (m *memRepo) RemoveByListing(ctx context.Context, market dlisting.Market, listingID string) error {
	return nil
}

func TestWatchUsecase_Register(t *testing.T) {
	r := &memRepo{}
	u := NewWatchUsecase(r)
	end := time.Now()
	if err := u.Register(context.Background(), dlisting.MarketYahooAuction, "a", "u", "g", "c", "m", 10, &end); err != nil {
		t.Fatal(err)
	}
	if len(r.items) != 1 || r.items[0].ListingID != "a" {
		t.Fatal("item not stored")
	}
}

func TestWatchUsecase_Register_error(t *testing.T) {
	r := &memRepo{addErr: errors.New("e")}
	u := NewWatchUsecase(r)
	if err := u.Register(context.Background(), dlisting.MarketYahooAuction, "a", "u", "g", "c", "m", 0, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchUsecase_Unregister(t *testing.T) {
	r := &memRepo{}
	u := NewWatchUsecase(r)
	if err := u.Unregister(context.Background(), dlisting.MarketYahooAuction, "a", "u", "m"); err != nil {
		t.Fatal(err)
	}
}

func TestWatchUsecase_Unregister_error(t *testing.T) {
	r := &memRepo{remErr: errors.New("e")}
	u := NewWatchUsecase(r)
	if err := u.Unregister(context.Background(), dlisting.MarketYahooAuction, "a", "u", "m"); err == nil {
		t.Fatal("expected error")
	}
}
