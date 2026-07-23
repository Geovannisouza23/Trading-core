package snapshot

const columns = `id, balance, equity, positions, open_orders, daily_pnl, drawdown, "timestamp"`

const latestQuery = `SELECT ` + columns + ` FROM account_snapshots ORDER BY "timestamp" DESC LIMIT 1`

const insertQuery = `
INSERT INTO account_snapshots (id, balance, equity, positions, open_orders, daily_pnl, drawdown, "timestamp")
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`
