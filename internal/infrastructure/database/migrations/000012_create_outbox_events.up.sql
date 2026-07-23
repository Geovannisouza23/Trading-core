CREATE TABLE outbox_events (
    id           UUID PRIMARY KEY,
    event_type   TEXT NOT NULL,
    payload      JSONB NOT NULL,
    status       TEXT NOT NULL DEFAULT 'PENDING',
    attempts     INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ,
    last_error   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX outbox_events_pending_idx ON outbox_events (created_at) WHERE status = 'PENDING';
