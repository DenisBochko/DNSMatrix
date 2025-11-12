-- 000012_add_assignments_table.up.sql

CREATE TABLE IF NOT EXISTS domain.assignments (
    id               UUID PRIMARY KEY,
    request_id       UUID NOT NULL REFERENCES domain.requests(id) ON DELETE CASCADE,
    agent_id         TEXT NOT NULL,
    agent_region     TEXT NOT NULL,
    status           domain.check_status NOT NULL DEFAULT 'PENDING',
    enqueued_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    error_text       TEXT,
    outbox_id        UUID
);

CREATE INDEX IF NOT EXISTS assignments_request_idx  ON domain.assignments(request_id);
CREATE INDEX IF NOT EXISTS assignments_agent_idx    ON domain.assignments(agent_id);
CREATE INDEX IF NOT EXISTS assignments_status_idx   ON domain.assignments(status);
