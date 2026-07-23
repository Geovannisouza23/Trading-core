CREATE TABLE account_snapshots (
    id          UUID PRIMARY KEY,
    balance     NUMERIC(24,8) NOT NULL,
    equity      NUMERIC(24,8) NOT NULL,
    positions   JSONB NOT NULL DEFAULT '[]',
    open_orders JSONB NOT NULL DEFAULT '[]',
    daily_pnl   NUMERIC(24,8) NOT NULL DEFAULT 0,
    drawdown    NUMERIC(10,6) NOT NULL DEFAULT 0,
    "timestamp" TIMESTAMPTZ NOT NULL
);

CREATE INDEX account_snapshots_timestamp_idx ON account_snapshots ("timestamp" DESC);
