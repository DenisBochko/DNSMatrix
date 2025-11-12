CREATE TABLE sso.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES sso.users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NULL,
    revoked BOOLEAN DEFAULT FALSE,

    CONSTRAINT uk_api_key_hash UNIQUE (key_hash)
);

CREATE INDEX idx_api_keys_user_id ON sso.api_keys(user_id);
CREATE INDEX idx_api_keys_revoked ON sso.api_keys(revoked);
CREATE INDEX idx_api_keys_expires_at ON sso.api_keys(expires_at);