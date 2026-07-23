CREATE TABLE positions (
    id             UUID PRIMARY KEY,
    symbol         TEXT NOT NULL,
    side           TEXT NOT NULL,
    quantity       NUMERIC(24,8) NOT NULL,
    entry_price    NUMERIC(24,8) NOT NULL,
    current_price  NUMERIC(24,8) NOT NULL,
    stop_price     NUMERIC(24,8),
    target_price   NUMERIC(24,8),
    unrealized_pnl NUMERIC(24,8) NOT NULL DEFAULT 0,
    realized_pnl   NUMERIC(24,8) NOT NULL DEFAULT 0,
    status         TEXT NOT NULL,
    version        INTEGER NOT NULL DEFAULT 1,
    opened_at      TIMESTAMPTZ NOT NULL,
    closed_at      TIMESTAMPTZ
);

CREATE INDEX positions_status_idx ON positions (status);
CREATE UNIQUE INDEX positions_open_symbol_idx ON positions (symbol) WHERE status = 'OPEN';
