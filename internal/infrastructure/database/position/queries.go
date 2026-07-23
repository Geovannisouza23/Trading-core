package position

const columns = `id, symbol, side, quantity, entry_price, current_price, stop_price, target_price,
	unrealized_pnl, realized_pnl, status, version, opened_at, closed_at`

const getByIDQuery = `SELECT ` + columns + ` FROM positions WHERE id = $1`

const listOpenQuery = `SELECT ` + columns + ` FROM positions WHERE status = 'OPEN' ORDER BY opened_at DESC`

const getOpenBySymbolQuery = `SELECT ` + columns + ` FROM positions WHERE symbol = $1 AND status = 'OPEN'`

const upsertQuery = `
INSERT INTO positions (
	id, symbol, side, quantity, entry_price, current_price, stop_price, target_price,
	unrealized_pnl, realized_pnl, status, version, opened_at, closed_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (id) DO UPDATE SET
	quantity = EXCLUDED.quantity,
	current_price = EXCLUDED.current_price,
	stop_price = EXCLUDED.stop_price,
	target_price = EXCLUDED.target_price,
	unrealized_pnl = EXCLUDED.unrealized_pnl,
	realized_pnl = EXCLUDED.realized_pnl,
	status = EXCLUDED.status,
	version = EXCLUDED.version,
	closed_at = EXCLUDED.closed_at
WHERE positions.version = $15
`
