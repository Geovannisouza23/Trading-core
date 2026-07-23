-- Optional manual seed for local development (the API also bootstraps this
-- automatically on first startup if no active account exists). Safe to
-- re-run: both inserts are no-ops once a row already exists.
INSERT INTO accounts (
    id, balance, equity, available_balance, peak_equity,
    daily_pnl, weekly_pnl, current_drawdown, operational_mode,
    is_active, version, updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    10000, 10000, 10000, 10000,
    0, 0, 0, 'PAPER',
    TRUE, 1, now()
)
ON CONFLICT (is_active) WHERE is_active DO NOTHING;

INSERT INTO operational_modes (id, current_mode, changed_by, origin, reason, changed_at, version)
VALUES (1, 'PAPER', 'system', 'seed', 'initial seed', now(), 1)
ON CONFLICT (id) DO NOTHING;
