CREATE TABLE system_incidents (
    id                UUID PRIMARY KEY,
    type              TEXT NOT NULL,
    severity          TEXT NOT NULL,
    description       TEXT NOT NULL,
    source            TEXT NOT NULL,
    related_entity_id TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL,
    detected_at       TIMESTAMPTZ NOT NULL,
    resolved_at       TIMESTAMPTZ,
    resolution        TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL
);

CREATE INDEX system_incidents_status_idx ON system_incidents (status);
CREATE INDEX system_incidents_detected_at_idx ON system_incidents (detected_at DESC);
