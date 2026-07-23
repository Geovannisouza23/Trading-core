CREATE TABLE trade_signals (
    id            UUID PRIMARY KEY,
    symbol        TEXT NOT NULL,
    side          TEXT NOT NULL,
    entry_price   NUMERIC(24,8) NOT NULL,
    stop_price    NUMERIC(24,8) NOT NULL,
    target_price  NUMERIC(24,8) NOT NULL,
    confidence    NUMERIC(5,4) NOT NULL,
    strategy_name TEXT NOT NULL,
    market_regime TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL,
    valid_until   TIMESTAMPTZ NOT NULL
);

CREATE INDEX trade_signals_created_at_idx ON trade_signals (created_at DESC);
