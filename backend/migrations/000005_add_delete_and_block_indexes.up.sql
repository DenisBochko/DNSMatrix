-- 000005_add_delete_and_block_indexes.up.sql

CREATE INDEX IF NOT EXISTS idx_user_deleted
    ON sso.users(deleted);

CREATE INDEX IF NOT EXISTS idx_user_blocked
    ON sso.users(blocked);
