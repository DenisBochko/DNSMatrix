-- 000003_add_verification_token_table.up.sql

CREATE TABLE IF NOT EXISTS sso.verification_tokens (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES sso.users(id) ON DELETE CASCADE,
    token BYTEA UNIQUE NOT NULL,
    code VARCHAR(4) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_verification_tokens_user_id ON sso.verification_tokens(user_id);
