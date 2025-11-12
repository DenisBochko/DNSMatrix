-- 000010_add_outbox_table.up.sql

CREATE SCHEMA IF NOT EXISTS messages;

CREATE TABLE IF NOT EXISTS messages.outbox_messages (
    id         UUID PRIMARY KEY,
    topic      TEXT NOT NULL,
    payload    BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    sent       BOOLEAN DEFAULT FALSE,
    sent_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_messages_unsent
    ON messages.outbox_messages (sent, topic);
