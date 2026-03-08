package sqlite

import (
	"database/sql"
	"fmt"
	"log"
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

	// 絶対パスにしておくとコンテナ等でパス解決の不具合を防ぎやすい
	openPath := dbPath
	if !filepath.IsAbs(dbPath) {
		abs, err := filepath.Abs(dbPath)
		if err == nil {
			openPath = abs
		}
	}
	db, err := sql.Open("sqlite", openPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	useDeleteJournal := os.Getenv("SQLITE_JOURNAL_MODE") == "DELETE"
	if useDeleteJournal {
		// コンテナで mmap が失敗して CANTOPEN/NOMEM になることがあるため、先に無効化する。
		if _, err := db.Exec("PRAGMA mmap_size=0"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set mmap_size=0: %w", err)
		}
		if _, err := db.Exec("PRAGMA journal_mode=DELETE"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set journal_mode=DELETE: %w", err)
		}
	} else if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		// メモリ制約環境でWALが失敗すると -wal/-shm が残り、開き直しでも書き込みが失敗する。
		// 残骸を削除してからプレーンなパスで開き直す（デフォルトは DELETE ジャーナル）。
		log.Printf("[sqlite] WAL mode failed (%v), removing WAL/shm and reopening", err)
		_ = db.Close()
		for _, suffix := range []string{"-wal", "-shm"} {
			_ = os.Remove(openPath + suffix)
		}
		db, err = sql.Open("sqlite", openPath)
		if err != nil {
			return nil, fmt.Errorf("open sqlite after WAL fallback: %w", err)
		}
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return db, nil
}
