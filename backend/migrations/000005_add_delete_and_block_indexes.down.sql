-- 000005_add_delete_and_block_indexes.down.sql

DROP INDEX IF EXISTS idx_user_deleted;
DROP INDEX IF EXISTS idx_user_blocked;
