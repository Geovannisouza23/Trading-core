CREATE TABLE operational_modes (
    id           SMALLINT PRIMARY KEY DEFAULT 1,
    current_mode TEXT NOT NULL,
    changed_by   TEXT NOT NULL,
    origin       TEXT NOT NULL,
    reason       TEXT NOT NULL,
    changed_at   TIMESTAMPTZ NOT NULL,
    version      INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT operational_modes_singleton CHECK (id = 1)
);
