package sqlite_test

import (
	"context"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
	"jo3qma.com/yahoo_auctions_bot/internal/infrastructure/sqlite"
)

func setupTestDB(t *testing.T) *sqlite.WatchRepository {
	t.Helper()
	db, err := sqlite.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlite.NewWatchRepository(db)
}

func TestWatchRepository_AddAndList(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	endTime := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second)
	item := &watch.WatchItem{
		AuctionID:      "abc12345678",
		UserID:         "user1",
		GuildID:        "guild1",
		ChannelID:      "chan1",
		MessageID:      "msg1",
		LastKnownPrice: 5000,
		EndTime:        &endTime,
	}

	if err := repo.Add(ctx, item); err != nil {
		t.Fatalf("add: %v", err)
	}

	items, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	got := items[0]
	if got.AuctionID != "abc12345678" {
		t.Errorf("auction_id = %s, want abc12345678", got.AuctionID)
	}
	if got.LastKnownPrice != 5000 {
		t.Errorf("price = %d, want 5000", got.LastKnownPrice)
	}
	if got.Reminded {
		t.Error("expected reminded = false")
	}
}

func TestWatchRepository_UpsertOnConflict(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	item := &watch.WatchItem{
		AuctionID: "abc12345678", UserID: "user1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000,
	}
	if err := repo.Add(ctx, item); err != nil {
		t.Fatalf("first add: %v", err)
	}

	item.LastKnownPrice = 2000
	if err := repo.Add(ctx, item); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	items, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after upsert, got %d", len(items))
	}
	if items[0].LastKnownPrice != 2000 {
		t.Errorf("price after upsert = %d, want 2000", items[0].LastKnownPrice)
	}
}

func TestWatchRepository_Remove(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	item := &watch.WatchItem{
		AuctionID: "abc12345678", UserID: "user1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000,
	}
	if err := repo.Add(ctx, item); err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := repo.Remove(ctx, "abc12345678", "user1", "m1"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	items, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items after remove, got %d", len(items))
	}
}

func TestWatchRepository_RemoveByAuctionID(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	for _, uid := range []string{"user1", "user2", "user3"} {
		item := &watch.WatchItem{
			AuctionID: "abc12345678", UserID: uid, GuildID: "g1",
			ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000,
		}
		if err := repo.Add(ctx, item); err != nil {
			t.Fatalf("add %s: %v", uid, err)
		}
	}

	if err := repo.RemoveByAuctionID(ctx, "abc12345678"); err != nil {
		t.Fatalf("remove by auction: %v", err)
	}

	items, err := repo.ListActive(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestWatchRepository_UpdatePrice(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	item := &watch.WatchItem{
		AuctionID: "abc12345678", UserID: "user1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000,
	}
	if err := repo.Add(ctx, item); err != nil {
		t.Fatalf("add: %v", err)
	}

	items, _ := repo.ListActive(ctx)
	if err := repo.UpdatePrice(ctx, items[0].ID, 3000); err != nil {
		t.Fatalf("update price: %v", err)
	}

	items, _ = repo.ListActive(ctx)
	if items[0].LastKnownPrice != 3000 {
		t.Errorf("price = %d, want 3000", items[0].LastKnownPrice)
	}
}

func TestWatchRepository_MarkReminded(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	item := &watch.WatchItem{
		AuctionID: "abc12345678", UserID: "user1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000,
	}
	if err := repo.Add(ctx, item); err != nil {
		t.Fatalf("add: %v", err)
	}

	items, _ := repo.ListActive(ctx)
	if err := repo.MarkReminded(ctx, items[0].ID); err != nil {
		t.Fatalf("mark reminded: %v", err)
	}

	items, _ = repo.ListActive(ctx)
	if !items[0].Reminded {
		t.Error("expected reminded = true")
	}
}

func TestWatchRepository_UpdateThreadID(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	for _, uid := range []string{"user1", "user2"} {
		item := &watch.WatchItem{
			AuctionID: "abc12345678", UserID: uid, GuildID: "g1",
			ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000,
		}
		if err := repo.Add(ctx, item); err != nil {
			t.Fatalf("add: %v", err)
		}
	}

	if err := repo.UpdateThreadID(ctx, "m1", "thread123"); err != nil {
		t.Fatalf("update thread: %v", err)
	}

	items, err := repo.FindByMessage(ctx, "m1")
	if err != nil {
		t.Fatalf("find by message: %v", err)
	}
	for _, it := range items {
		if it.ThreadID != "thread123" {
			t.Errorf("user %s thread_id = %s, want thread123", it.UserID, it.ThreadID)
		}
	}
}

func TestWatchRepository_FindByMessage(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	item1 := &watch.WatchItem{
		AuctionID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m1", LastKnownPrice: 1000,
	}
	item2 := &watch.WatchItem{
		AuctionID: "auc1", UserID: "u1", GuildID: "g1",
		ChannelID: "c1", MessageID: "m2", LastKnownPrice: 2000,
	}
	if err := repo.Add(ctx, item1); err != nil {
		t.Fatalf("repo.Add item1: %v", err)
	}
	if err := repo.Add(ctx, item2); err != nil {
		t.Fatalf("repo.Add item2: %v", err)
	}

	items, err := repo.FindByMessage(ctx, "m1")
	if err != nil {
		t.Fatalf("find by message: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for m1, got %d", len(items))
	}
}
