-- 000013_add_check_results_table.up.sql

CREATE TABLE IF NOT EXISTS domain.check_results (
    id               UUID PRIMARY KEY,
    assignment_id    UUID NOT NULL,
    type             TEXT NOT NULL, -- http|ping|tcp|traceroute|dns
    status           domain.check_status NOT NULL,
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    payload          JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS results_assignment_idx ON domain.check_results(assignment_id);
CREATE INDEX IF NOT EXISTS results_type_idx       ON domain.check_results(type);
