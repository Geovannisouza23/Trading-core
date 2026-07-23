package event

const columns = `id, event_type, direction, severity, confidence, affected_assets, action, source_count, has_official_source, published_at, detected_at, expires_at, status`

const getByIDQuery = `SELECT ` + columns + ` FROM market_events WHERE id = $1`

const listActiveQuery = `SELECT ` + columns + ` FROM market_events WHERE status = 'ACTIVE' ORDER BY detected_at DESC`

const listQuery = `SELECT ` + columns + ` FROM market_events ORDER BY detected_at DESC LIMIT $1`

const insertQuery = `
INSERT INTO market_events (id, event_type, direction, severity, confidence, affected_assets, action, source_count, has_official_source, published_at, detected_at, expires_at, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status
`
