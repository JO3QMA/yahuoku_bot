package rqlite

const schema = `
DROP TABLE IF EXISTS watch_items;
CREATE TABLE watch_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    market TEXT NOT NULL,
    listing_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    last_known_price INTEGER NOT NULL DEFAULT 0,
    end_time DATETIME,
    reminded INTEGER NOT NULL DEFAULT 0,
    thread_id TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(market, listing_id, user_id, message_id)
);
CREATE INDEX IF NOT EXISTS idx_watch_items_message ON watch_items(message_id);
CREATE INDEX IF NOT EXISTS idx_watch_items_listing ON watch_items(market, listing_id);
`
