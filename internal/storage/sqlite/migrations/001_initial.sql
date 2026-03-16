PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS heroes (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    archetype TEXT NOT NULL,
    traits_json TEXT NOT NULL DEFAULT '[]',
    voice_style TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
) STRICT;

CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    danger_level INTEGER NOT NULL DEFAULT 0,
    tags_json TEXT NOT NULL DEFAULT '[]'
) STRICT;

CREATE TABLE IF NOT EXISTS hero_state (
    hero_id INTEGER PRIMARY KEY,
    location_id INTEGER NOT NULL,
    health INTEGER NOT NULL,
    energy INTEGER NOT NULL,
    stress INTEGER NOT NULL,
    gold INTEGER NOT NULL,
    current_time TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hero_id) REFERENCES heroes(id),
    FOREIGN KEY (location_id) REFERENCES locations(id)
) STRICT;

CREATE TABLE IF NOT EXISTS event_types (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    base_weight INTEGER NOT NULL DEFAULT 1,
    cooldown_ticks INTEGER NOT NULL DEFAULT 0,
    rules_json TEXT NOT NULL DEFAULT '{}'
) STRICT;

CREATE TABLE IF NOT EXISTS ticks (
    id INTEGER PRIMARY KEY,
    hero_id INTEGER NOT NULL,
    scheduled_for TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    status TEXT NOT NULL,
    error_text TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hero_id) REFERENCES heroes(id)
) STRICT;

CREATE TABLE IF NOT EXISTS world_events (
    id INTEGER PRIMARY KEY,
    hero_id INTEGER NOT NULL,
    tick_id INTEGER NOT NULL,
    event_type_id INTEGER NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    outcome_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hero_id) REFERENCES heroes(id),
    FOREIGN KEY (tick_id) REFERENCES ticks(id),
    FOREIGN KEY (event_type_id) REFERENCES event_types(id)
) STRICT;

CREATE TABLE IF NOT EXISTS journal_entries (
    id INTEGER PRIMARY KEY,
    hero_id INTEGER NOT NULL,
    world_event_id INTEGER NOT NULL,
    channel TEXT NOT NULL,
    external_message_id TEXT,
    text TEXT NOT NULL,
    status TEXT NOT NULL,
    published_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (hero_id) REFERENCES heroes(id),
    FOREIGN KEY (world_event_id) REFERENCES world_events(id)
) STRICT;

CREATE TABLE IF NOT EXISTS outbox (
    id INTEGER PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id INTEGER NOT NULL,
    destination TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    status TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TEXT,
    last_error TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at TEXT
) STRICT;

CREATE INDEX IF NOT EXISTS idx_ticks_hero_status ON ticks(hero_id, status, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_outbox_status_next_retry ON outbox(status, next_retry_at, created_at);
