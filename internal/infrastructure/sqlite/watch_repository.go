package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"
)

// WatchRepository はSQLiteによるwatch.Repositoryの実装。
type WatchRepository struct {
	db *sql.DB
}

// NewWatchRepository はWatchRepositoryを生成する。
func NewWatchRepository(db *sql.DB) *WatchRepository {
	return &WatchRepository{db: db}
}

func (r *WatchRepository) Add(ctx context.Context, item *watch.WatchItem) error {
	query := `
		INSERT INTO watch_items (auction_id, user_id, guild_id, channel_id, message_id, last_known_price, end_time, reminded, thread_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, '')
		ON CONFLICT(auction_id, user_id, message_id) DO UPDATE SET
			last_known_price = excluded.last_known_price,
			end_time = excluded.end_time
	`
	var endTime *string
	if item.EndTime != nil {
		s := item.EndTime.UTC().Format(time.RFC3339)
		endTime = &s
	}
	_, err := r.db.ExecContext(ctx, query,
		item.AuctionID, item.UserID, item.GuildID, item.ChannelID, item.MessageID,
		item.LastKnownPrice, endTime,
	)
	if err != nil {
		return fmt.Errorf("insert watch item: %w", err)
	}
	return nil
}

func (r *WatchRepository) Remove(ctx context.Context, auctionID, userID, messageID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM watch_items WHERE auction_id = ? AND user_id = ? AND message_id = ?`,
		auctionID, userID, messageID,
	)
	if err != nil {
		return fmt.Errorf("delete watch item: %w", err)
	}
	return nil
}

func (r *WatchRepository) RemoveByAuctionID(ctx context.Context, auctionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM watch_items WHERE auction_id = ?`, auctionID)
	if err != nil {
		return fmt.Errorf("delete watch items by auction: %w", err)
	}
	return nil
}

func (r *WatchRepository) ListActive(ctx context.Context) ([]*watch.WatchItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, auction_id, user_id, guild_id, channel_id, message_id, last_known_price, end_time, reminded, thread_id, created_at FROM watch_items`)
	if err != nil {
		return nil, fmt.Errorf("list watch items: %w", err)
	}
	defer rows.Close()

	return scanWatchItems(rows)
}

func (r *WatchRepository) UpdatePrice(ctx context.Context, id int64, newPrice int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE watch_items SET last_known_price = ? WHERE id = ?`, newPrice, id)
	if err != nil {
		return fmt.Errorf("update price: %w", err)
	}
	return nil
}

func (r *WatchRepository) MarkReminded(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE watch_items SET reminded = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("mark reminded: %w", err)
	}
	return nil
}

func (r *WatchRepository) UpdateThreadID(ctx context.Context, messageID, threadID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE watch_items SET thread_id = ? WHERE message_id = ?`, threadID, messageID)
	if err != nil {
		return fmt.Errorf("update thread id: %w", err)
	}
	return nil
}

func (r *WatchRepository) FindByMessage(ctx context.Context, messageID string) ([]*watch.WatchItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, auction_id, user_id, guild_id, channel_id, message_id, last_known_price, end_time, reminded, thread_id, created_at FROM watch_items WHERE message_id = ?`,
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("find by message: %w", err)
	}
	defer rows.Close()

	return scanWatchItems(rows)
}

func scanWatchItems(rows *sql.Rows) ([]*watch.WatchItem, error) {
	var items []*watch.WatchItem
	for rows.Next() {
		var item watch.WatchItem
		var endTime sql.NullString
		var reminded int
		err := rows.Scan(
			&item.ID, &item.AuctionID, &item.UserID, &item.GuildID,
			&item.ChannelID, &item.MessageID, &item.LastKnownPrice,
			&endTime, &reminded, &item.ThreadID, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan watch item: %w", err)
		}
		item.Reminded = reminded != 0
		if endTime.Valid {
			t, err := time.Parse(time.RFC3339, endTime.String)
			if err == nil {
				item.EndTime = &t
			}
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}
