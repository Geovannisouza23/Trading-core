package signal

const columns = `id, symbol, side, entry_price, stop_price, target_price, confidence, strategy_name, market_regime, created_at, valid_until`

const getByIDQuery = `SELECT ` + columns + ` FROM trade_signals WHERE id = $1`

const listQuery = `SELECT ` + columns + ` FROM trade_signals ORDER BY created_at DESC LIMIT $1`

const insertQuery = `
INSERT INTO trade_signals (id, symbol, side, entry_price, stop_price, target_price, confidence, strategy_name, market_regime, created_at, valid_until)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (id) DO NOTHING
`
