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

// openCfg は Open の環境依存部分をまとめる（テストで差し替え可能）。
type openCfg struct {
	getenv      func(string) string
	mkdirAll    func(string, os.FileMode) error
	filepathAbs func(string) (string, error)
	remove      func(string) error
	sqlOpen     func(driverName, dataSourceName string) (*sql.DB, error)
}

func defaultOpenCfg() openCfg {
	return openCfg{
		getenv:      os.Getenv,
		mkdirAll:    os.MkdirAll,
		filepathAbs: filepath.Abs,
		remove:      os.Remove,
		sqlOpen:     sql.Open,
	}
}

// OpenOption は Open の挙動を上書きする。
type OpenOption func(*openCfg)

// WithSQLOpen は sql.Open の代替を注入する（テスト用）。
func WithSQLOpen(fn func(driverName, dataSourceName string) (*sql.DB, error)) OpenOption {
	return func(c *openCfg) {
		c.sqlOpen = fn
	}
}

// WithEnv は os.Getenv の代替を注入する。
func WithEnv(fn func(string) string) OpenOption {
	return func(c *openCfg) {
		c.getenv = fn
	}
}

// WithMkdirAll はディレクトリ作成関数を注入する。
func WithMkdirAll(fn func(string, os.FileMode) error) OpenOption {
	return func(c *openCfg) {
		c.mkdirAll = fn
	}
}

// WithFilepathAbs は filepath.Abs の代替を注入する。
func WithFilepathAbs(fn func(string) (string, error)) OpenOption {
	return func(c *openCfg) {
		c.filepathAbs = fn
	}
}

// WithRemove はファイル削除の代替を注入する。
func WithRemove(fn func(string) error) OpenOption {
	return func(c *openCfg) {
		c.remove = fn
	}
}

// Open はSQLiteデータベースを開き、スキーマを初期化する。
func Open(dbPath string, opts ...OpenOption) (*sql.DB, error) {
	cfg := defaultOpenCfg()
	for _, o := range opts {
		o(&cfg)
	}
	return openWithCfg(dbPath, cfg)
}

func openWithCfg(dbPath string, cfg openCfg) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := cfg.mkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	openPath := dbPath
	if !filepath.IsAbs(dbPath) {
		abs, err := cfg.filepathAbs(dbPath)
		if err == nil {
			openPath = abs
		}
	}
	db, err := cfg.sqlOpen("sqlite", openPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	useDeleteJournal := cfg.getenv("SQLITE_JOURNAL_MODE") == "DELETE"
	if useDeleteJournal {
		if _, err := db.Exec("PRAGMA mmap_size=0"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set mmap_size=0: %w", err)
		}
		if _, err := db.Exec("PRAGMA journal_mode=DELETE"); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set journal_mode=DELETE: %w", err)
		}
	} else if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("[sqlite] WAL mode failed (%v), removing WAL/shm and reopening", err)
		_ = db.Close()
		for _, suffix := range []string{"-wal", "-shm"} {
			_ = cfg.remove(openPath + suffix)
		}
		db, err = cfg.sqlOpen("sqlite", openPath)
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
