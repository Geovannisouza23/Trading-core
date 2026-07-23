package outbox

const insertQuery = `
INSERT INTO outbox_events (id, event_type, payload, status, attempts, created_at)
VALUES ($1, $2, $3, $4, 0, $5)
`

// fetchPendingBatchQuery atomically claims up to $1 pending rows by moving
// them straight to PROCESSING inside a single statement. Because the SELECT
// (with FOR UPDATE SKIP LOCKED) and the UPDATE happen in the same implicit
// transaction, there is no window in which two concurrent workers could
// both see the same row as PENDING.
const fetchPendingBatchQuery = `
WITH claimed AS (
    SELECT id FROM outbox_events
    WHERE status = 'PENDING'
    ORDER BY created_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events
SET status = 'PROCESSING'
FROM claimed
WHERE outbox_events.id = claimed.id
RETURNING outbox_events.id, outbox_events.event_type, outbox_events.payload,
          outbox_events.status, outbox_events.attempts, outbox_events.created_at,
          outbox_events.processed_at, outbox_events.last_error
`

const markProcessedQuery = `
UPDATE outbox_events SET status = 'PROCESSED', processed_at = $2 WHERE id = $1
`

const markFailedQuery = `
UPDATE outbox_events
SET attempts = attempts + 1,
    last_error = $2,
    status = CASE WHEN attempts + 1 >= $3 THEN 'FAILED' ELSE 'PENDING' END
WHERE id = $1
`
