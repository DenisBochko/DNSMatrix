-- 000001_add_test_table.up.sql

CREATE SCHEMA IF NOT EXISTS test;

CREATE TABLE IF NOT EXISTS test.init (
    id SERIAL PRIMARY KEY,
    data VARCHAR(64)
);
