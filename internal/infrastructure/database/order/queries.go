package order

const columns = `id, client_order_id, broker_order_id, symbol, side, type, quantity, filled_quantity,
	requested_price, average_execution_price, stop_price, target_price, status, strategy_name,
	signal_id, risk_decision_id, failure_reason, version, created_at, updated_at`

const getByIDQuery = `SELECT ` + columns + ` FROM orders WHERE id = $1`

const getByClientOrderIDQuery = `SELECT ` + columns + ` FROM orders WHERE client_order_id = $1`

// Terminal statuses mirror domain/order.Status.IsTerminal(); kept as SQL
// literals here since a repository query is the appropriate place for a
// stable, storage-facing constant list.
const listOpenQuery = `SELECT ` + columns + ` FROM orders
	WHERE status NOT IN ('FILLED', 'CANCELLED', 'REJECTED', 'FAILED')
	ORDER BY created_at DESC`

const listBySignalIDQuery = `SELECT ` + columns + ` FROM orders WHERE signal_id = $1 ORDER BY created_at DESC`

const listQuery = `SELECT ` + columns + ` FROM orders ORDER BY created_at DESC LIMIT $1`

const upsertQuery = `
INSERT INTO orders (
	id, client_order_id, broker_order_id, symbol, side, type, quantity, filled_quantity,
	requested_price, average_execution_price, stop_price, target_price, status, strategy_name,
	signal_id, risk_decision_id, failure_reason, version, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
ON CONFLICT (id) DO UPDATE SET
	broker_order_id = EXCLUDED.broker_order_id,
	filled_quantity = EXCLUDED.filled_quantity,
	average_execution_price = EXCLUDED.average_execution_price,
	status = EXCLUDED.status,
	failure_reason = EXCLUDED.failure_reason,
	version = EXCLUDED.version,
	updated_at = EXCLUDED.updated_at
WHERE orders.version = $21
`

const insertExecutionQuery = `
INSERT INTO order_executions (id, order_id, quantity, price, executed_at)
VALUES ($1, $2, $3, $4, $5)
`
