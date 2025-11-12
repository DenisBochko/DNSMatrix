-- 000011_add_inbox_table.down.sql

DROP INDEX IF EXISTS idx_outbox_messages_unsent;

DROP TABLE IF EXISTS outbox_messages;
