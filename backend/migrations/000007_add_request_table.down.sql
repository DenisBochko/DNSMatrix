-- 000007_add_request_table.down.sql

DROP TABLE IF EXISTS domain.requests;

DROP TYPE IF EXISTS domain.check_status;

DROP SCHEMA IF EXISTS domain;
