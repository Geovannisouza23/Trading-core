package risk

const columns = `id, signal_id, allowed, original_position_size, approved_position_size, reason_codes, applied_rules, event_restrictions, created_at`

const getByIDQuery = `SELECT ` + columns + ` FROM risk_decisions WHERE id = $1`

const listQuery = `SELECT ` + columns + ` FROM risk_decisions ORDER BY created_at DESC LIMIT $1`

const insertQuery = `
INSERT INTO risk_decisions (id, signal_id, allowed, original_position_size, approved_position_size, reason_codes, applied_rules, event_restrictions, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO NOTHING
`
