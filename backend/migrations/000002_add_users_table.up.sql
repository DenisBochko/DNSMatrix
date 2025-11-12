-- 000002_add_users_table.up.sql

CREATE SCHEMA IF NOT EXISTS sso;

CREATE TABLE IF NOT EXISTS sso.users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password BYTEA NOT NULL,
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_confirmed ON sso.users(confirmed);
CREATE INDEX IF NOT EXISTS idx_users_deleted ON sso.users(deleted);
CREATE INDEX IF NOT EXISTS idx_users_blocked ON sso.users(blocked);
CREATE INDEX IF NOT EXISTS idx_users_role ON sso.users(role);
