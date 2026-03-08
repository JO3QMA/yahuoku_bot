package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS watch_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auction_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    last_known_price INTEGER NOT NULL DEFAULT 0,
    end_time DATETIME,
    reminded INTEGER NOT NULL DEFAULT 0,
    thread_id TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(auction_id, user_id, message_id)
);
CREATE INDEX IF NOT EXISTS idx_watch_items_message ON watch_items(message_id);
CREATE INDEX IF NOT EXISTS idx_watch_items_auction ON watch_items(auction_id);
`

// Open はSQLiteデータベースを開き、スキーマを初期化する。
func Open(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return db, nil
}
