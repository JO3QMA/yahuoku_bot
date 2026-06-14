package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWatchRepository_Add_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("INSERT INTO watch_items").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	err = repo.Add(context.Background(), &watch.Watch{
		AuctionID: "a", UserID: "u", GuildID: "g", ChannelID: "c", MessageID: "m",
		LastKnownPrice: 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_Remove_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("DELETE FROM watch_items").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	err = repo.Remove(context.Background(), "a", "u", "m")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_RemoveByAuctionID_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("DELETE FROM watch_items WHERE auction_id").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	err = repo.RemoveByAuctionID(context.Background(), "a")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_ListActive_queryErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	_, err = repo.ListActive(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_ListActive_scanErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	rows := sqlmock.NewRows([]string{"id"}).AddRow("bad")
	mock.ExpectQuery("SELECT id").WillReturnRows(rows)
	repo := NewWatchRepository(db)
	_, err = repo.ListActive(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_UpdatePrice_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("UPDATE watch_items SET last_known_price").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	err = repo.UpdatePrice(context.Background(), 1, 2)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_MarkReminded_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("UPDATE watch_items SET reminded").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	err = repo.MarkReminded(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_UpdateThreadID_error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("UPDATE watch_items SET thread_id").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	err = repo.UpdateThreadID(context.Background(), "m", "t")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_FindByMessage_queryErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id").WillReturnError(errors.New("e"))
	repo := NewWatchRepository(db)
	_, err = repo.FindByMessage(context.Background(), "m")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWatchRepository_FindByMessage_scanErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	rows := sqlmock.NewRows([]string{"id"}).AddRow("bad")
	mock.ExpectQuery("SELECT id").WillReturnRows(rows)
	repo := NewWatchRepository(db)
	_, err = repo.FindByMessage(context.Background(), "m")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScanWatchs_rowsErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	rows := sqlmock.NewRows([]string{
		"id", "auction_id", "user_id", "guild_id", "channel_id", "message_id",
		"last_known_price", "end_time", "reminded", "thread_id", "created_at",
	}).AddRow(1, "a", "u", "g", "c", "m", 1, nil, 0, "", time.Now().UTC().Format(time.RFC3339)).RowError(0, errors.New("row"))
	mock.ExpectQuery("SELECT id").WillReturnRows(rows)
	repo := NewWatchRepository(db)
	_, err = repo.ListActive(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
