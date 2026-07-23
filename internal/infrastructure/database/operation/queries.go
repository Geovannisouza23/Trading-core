package operation

const getQuery = `SELECT current_mode, changed_by, origin, reason, changed_at, version FROM operational_modes WHERE id = 1`

const upsertQuery = `
INSERT INTO operational_modes (id, current_mode, changed_by, origin, reason, changed_at, version)
VALUES (1, $1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
	current_mode = EXCLUDED.current_mode,
	changed_by = EXCLUDED.changed_by,
	origin = EXCLUDED.origin,
	reason = EXCLUDED.reason,
	changed_at = EXCLUDED.changed_at,
	version = EXCLUDED.version
WHERE operational_modes.version = $7
`
