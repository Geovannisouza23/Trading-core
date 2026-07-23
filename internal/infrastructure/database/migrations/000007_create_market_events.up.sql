CREATE TABLE market_events (
    id                  UUID PRIMARY KEY,
    event_type          TEXT NOT NULL,
    direction           TEXT NOT NULL,
    severity            TEXT NOT NULL,
    confidence          NUMERIC(5,4) NOT NULL,
    affected_assets     TEXT[] NOT NULL DEFAULT '{}',
    action              TEXT NOT NULL,
    source_count        INTEGER NOT NULL DEFAULT 1,
    has_official_source BOOLEAN NOT NULL DEFAULT FALSE,
    published_at        TIMESTAMPTZ NOT NULL,
    detected_at         TIMESTAMPTZ NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    status              TEXT NOT NULL
);

CREATE INDEX market_events_status_idx ON market_events (status);
CREATE INDEX market_events_expires_at_idx ON market_events (expires_at);
