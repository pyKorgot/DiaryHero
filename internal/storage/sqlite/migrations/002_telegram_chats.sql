CREATE TABLE IF NOT EXISTS telegram_chats (
    chat_id TEXT PRIMARY KEY,
    chat_type TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    username TEXT NOT NULL DEFAULT '',
    is_default INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'unknown',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
) STRICT;

CREATE INDEX IF NOT EXISTS idx_telegram_chats_default ON telegram_chats(is_default, updated_at);
