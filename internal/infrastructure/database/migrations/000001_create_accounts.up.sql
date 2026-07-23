CREATE TABLE accounts (
    id                UUID PRIMARY KEY,
    balance           NUMERIC(24,8) NOT NULL,
    equity            NUMERIC(24,8) NOT NULL,
    available_balance NUMERIC(24,8) NOT NULL,
    peak_equity       NUMERIC(24,8) NOT NULL,
    daily_pnl         NUMERIC(24,8) NOT NULL DEFAULT 0,
    weekly_pnl        NUMERIC(24,8) NOT NULL DEFAULT 0,
    current_drawdown  NUMERIC(10,6) NOT NULL DEFAULT 0,
    operational_mode  TEXT NOT NULL,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    version           INTEGER NOT NULL DEFAULT 1,
    updated_at        TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX accounts_single_active_idx ON accounts (is_active) WHERE is_active;
