-- 000014_add_check_agents_table.up.sql

CREATE TABLE IF NOT EXISTS domain.agents (
    id           UUID PRIMARY KEY,
    region       TEXT NOT NULL,
    asn          INTEGER,
    online       BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO domain.agents (id, region, asn, online, updated_at)
VALUES ('6d40a8b9-a135-4b67-b96b-0579c6ae0f76', 'APAC', 12345, true, NOW());

INSERT INTO domain.agents (id, region, asn, online, updated_at)
VALUES ('f022e955-1dff-4cf0-979a-80b118fa1126', 'US', 67890, true, NOW());

INSERT INTO domain.agents (id, region, asn, online, updated_at)
VALUES ('1832f867-3295-4cb5-8b9d-f34fe8723560', 'EU', 23456, true, NOW());
