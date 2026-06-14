package rqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/watch"

	rqlitehttp "github.com/rqlite/rqlite-go-http"
)

// WatchRepository は rqlite による watch.Repository の実装。
type WatchRepository struct {
	client *Client
}

// NewWatchRepository は WatchRepository を返す。
func NewWatchRepository(client *Client) *WatchRepository {
	return &WatchRepository{client: client}
}

func (r *WatchRepository) Add(ctx context.Context, item *watch.WatchItem) error {
	// reminded = 0: 再登録時にリマインドを再送可能にする
	query := `
		INSERT INTO watch_items (auction_id, user_id, guild_id, channel_id, message_id, last_known_price, end_time, reminded, thread_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0, '')
		ON CONFLICT(auction_id, user_id, message_id) DO UPDATE SET
			last_known_price = excluded.last_known_price,
			end_time = excluded.end_time,
			reminded = 0
	`
	var endTime any
	if item.EndTime != nil {
		endTime = item.EndTime.UTC().Format(time.RFC3339)
	} else {
		endTime = nil
	}
	_, err := r.client.h.ExecuteSingle(ctx, query,
		item.AuctionID, item.UserID, item.GuildID, item.ChannelID, item.MessageID,
		item.LastKnownPrice, endTime,
	)
	if err != nil {
		return fmt.Errorf("insert watch item: %w", err)
	}
	return nil
}

func (r *WatchRepository) Remove(ctx context.Context, auctionID, userID, messageID string) error {
	_, err := r.client.h.ExecuteSingle(ctx,
		`DELETE FROM watch_items WHERE auction_id = ? AND user_id = ? AND message_id = ?`,
		auctionID, userID, messageID,
	)
	if err != nil {
		return fmt.Errorf("delete watch item: %w", err)
	}
	return nil
}

func (r *WatchRepository) RemoveByAuctionID(ctx context.Context, auctionID string) error {
	_, err := r.client.h.ExecuteSingle(ctx, `DELETE FROM watch_items WHERE auction_id = ?`, auctionID)
	if err != nil {
		return fmt.Errorf("delete watch items by auction: %w", err)
	}
	return nil
}

func (r *WatchRepository) ListActive(ctx context.Context) ([]*watch.WatchItem, error) {
	q := `SELECT id, auction_id, user_id, guild_id, channel_id, message_id, last_known_price, end_time, reminded, thread_id, created_at FROM watch_items`
	resp, err := r.client.h.QuerySingle(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list watch items: %w", err)
	}
	return parseQueryResults(resp)
}

func (r *WatchRepository) UpdatePrice(ctx context.Context, id int64, newPrice int64) error {
	_, err := r.client.h.ExecuteSingle(ctx, `UPDATE watch_items SET last_known_price = ? WHERE id = ?`, newPrice, id)
	if err != nil {
		return fmt.Errorf("update price: %w", err)
	}
	return nil
}

func (r *WatchRepository) MarkReminded(ctx context.Context, id int64) error {
	_, err := r.client.h.ExecuteSingle(ctx, `UPDATE watch_items SET reminded = 1 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("mark reminded: %w", err)
	}
	return nil
}

func (r *WatchRepository) UpdateThreadID(ctx context.Context, messageID, threadID string) error {
	_, err := r.client.h.ExecuteSingle(ctx, `UPDATE watch_items SET thread_id = ? WHERE message_id = ?`, threadID, messageID)
	if err != nil {
		return fmt.Errorf("update thread id: %w", err)
	}
	return nil
}

func (r *WatchRepository) FindByMessage(ctx context.Context, messageID string) ([]*watch.WatchItem, error) {
	q := `SELECT id, auction_id, user_id, guild_id, channel_id, message_id, last_known_price, end_time, reminded, thread_id, created_at FROM watch_items WHERE message_id = ?`
	resp, err := r.client.h.QuerySingle(ctx, q, messageID)
	if err != nil {
		return nil, fmt.Errorf("find by message: %w", err)
	}
	return parseQueryResults(resp)
}

func parseQueryResults(resp *rqlitehttp.QueryResponse) ([]*watch.WatchItem, error) {
	if ok, i, msg := resp.HasError(); ok {
		return nil, fmt.Errorf("query error at %d: %s", i, msg)
	}
	results := resp.GetQueryResults()
	if len(results) == 0 {
		return nil, nil
	}
	// 単一 SELECT の結果は results[0] に入る
	qr := results[0]
	var items []*watch.WatchItem
	for _, row := range qr.Values {
		item, err := rowToWatchItem(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// row は id, auction_id, user_id, guild_id, channel_id, message_id, last_known_price, end_time, reminded, thread_id, created_at の順
func rowToWatchItem(row []any) (*watch.WatchItem, error) {
	if len(row) < 11 {
		return nil, fmt.Errorf("row has %d columns, want 11", len(row))
	}
	item := &watch.WatchItem{}
	if v, err := toInt64(row[0]); err == nil {
		item.ID = v
	}
	item.AuctionID = toString(row[1])
	item.UserID = toString(row[2])
	item.GuildID = toString(row[3])
	item.ChannelID = toString(row[4])
	item.MessageID = toString(row[5])
	if v, err := toInt64(row[6]); err == nil {
		item.LastKnownPrice = v
	}
	if t := parseTime(row[7]); t != nil {
		item.EndTime = t
	}
	if v, err := toInt64(row[8]); err == nil {
		item.Reminded = v != 0
	}
	item.ThreadID = toString(row[9])
	if t := parseTime(row[10]); t != nil {
		item.CreatedAt = *t
	}
	return item, nil
}

func toInt64(v any) (int64, error) {
	switch x := v.(type) {
	case json.Number:
		return x.Int64()
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func parseTime(v any) *time.Time {
	if v == nil {
		return nil
	}
	s := toString(v)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
