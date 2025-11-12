-- 000010_add_outbox_table.down.sql

DROP INDEX IF EXISTS idx_outbox_messages_unsent;

DROP TABLE IF EXISTS messages.outbox_messages;

DROP SCHEMA IF EXISTS messages;
