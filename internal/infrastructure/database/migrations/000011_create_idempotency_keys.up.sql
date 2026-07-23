CREATE TABLE idempotency_keys (
    scope      TEXT NOT NULL,
    key        TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (scope, key)
);

CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);
