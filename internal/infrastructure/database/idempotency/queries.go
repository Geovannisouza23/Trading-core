package idempotency

const reserveQuery = `
INSERT INTO idempotency_keys (scope, key, created_at, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (scope, key) DO UPDATE
    SET created_at = EXCLUDED.created_at, expires_at = EXCLUDED.expires_at
    WHERE idempotency_keys.expires_at < EXCLUDED.created_at
`
