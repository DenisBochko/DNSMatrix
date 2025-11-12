-- 000007_add_request_table.up.sql

CREATE SCHEMA IF NOT EXISTS domain;

CREATE TYPE domain.check_status AS ENUM ('PENDING','RUNNING','DONE','FAILED','TIMEOUT');

CREATE TABLE IF NOT EXISTS domain.requests (
    id               UUID PRIMARY KEY,
    target           TEXT NOT NULL,
    timeout_seconds  INTEGER NOT NULL,
    broadcast        BOOLEAN NOT NULL DEFAULT FALSE,
    client_ip        INET,
    user_agent       TEXT,
    client_asn       INTEGER,
    client_cc        CHAR(2),
    client_region    TEXT,
    status           domain.check_status NOT NULL DEFAULT 'PENDING',
    checks_types     TEXT[] NOT NULL,
    request_json     JSONB NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS requests_target_idx  ON domain.requests(target);
CREATE INDEX IF NOT EXISTS requests_status_idx  ON domain.requests(status);
CREATE INDEX IF NOT EXISTS requests_region_idx  ON domain.requests(client_region);
CREATE INDEX IF NOT EXISTS requests_created_idx ON domain.requests(created_at);
