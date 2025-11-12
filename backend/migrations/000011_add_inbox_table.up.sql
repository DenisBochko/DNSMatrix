-- 000011_add_inbox_table.up.sql

CREATE TABLE IF NOT EXISTS messages.inbox_messages (
    id           UUID PRIMARY KEY,
    topic        TEXT NOT NULL,
    payload      BYTEA NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    processed    BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_inbox_messages_unprocessed
    ON messages.inbox_messages (processed, topic);
