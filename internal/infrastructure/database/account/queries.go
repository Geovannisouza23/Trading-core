package account

const columns = `id, balance, equity, available_balance, peak_equity, daily_pnl, weekly_pnl, current_drawdown, operational_mode, version, updated_at`

const getByIDQuery = `SELECT ` + columns + ` FROM accounts WHERE id = $1`

const getActiveQuery = `SELECT ` + columns + ` FROM accounts WHERE is_active LIMIT 1`

const upsertQuery = `
INSERT INTO accounts (id, balance, equity, available_balance, peak_equity, daily_pnl, weekly_pnl, current_drawdown, operational_mode, is_active, version, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, $10, $11)
ON CONFLICT (id) DO UPDATE SET
    balance = EXCLUDED.balance,
    equity = EXCLUDED.equity,
    available_balance = EXCLUDED.available_balance,
    peak_equity = EXCLUDED.peak_equity,
    daily_pnl = EXCLUDED.daily_pnl,
    weekly_pnl = EXCLUDED.weekly_pnl,
    current_drawdown = EXCLUDED.current_drawdown,
    operational_mode = EXCLUDED.operational_mode,
    version = EXCLUDED.version,
    updated_at = EXCLUDED.updated_at
WHERE accounts.version = $12
`
