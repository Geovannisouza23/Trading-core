package incident

const columns = `id, type, severity, description, source, related_entity_id, status, detected_at, resolved_at, resolution, created_at, updated_at`

const getByIDQuery = `SELECT ` + columns + ` FROM system_incidents WHERE id = $1`

const listOpenQuery = `SELECT ` + columns + ` FROM system_incidents WHERE status = 'OPEN' ORDER BY detected_at DESC`

const listQuery = `SELECT ` + columns + ` FROM system_incidents ORDER BY detected_at DESC LIMIT $1`

const upsertQuery = `
INSERT INTO system_incidents (id, type, severity, description, source, related_entity_id, status, detected_at, resolved_at, resolution, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (id) DO UPDATE SET
	status = EXCLUDED.status,
	resolved_at = EXCLUDED.resolved_at,
	resolution = EXCLUDED.resolution,
	updated_at = EXCLUDED.updated_at
`
