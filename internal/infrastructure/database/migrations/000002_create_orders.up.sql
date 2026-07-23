CREATE TABLE orders (
    id                      UUID PRIMARY KEY,
    client_order_id         TEXT NOT NULL,
    broker_order_id         TEXT NOT NULL DEFAULT '',
    symbol                  TEXT NOT NULL,
    side                    TEXT NOT NULL,
    type                    TEXT NOT NULL,
    quantity                NUMERIC(24,8) NOT NULL,
    filled_quantity         NUMERIC(24,8) NOT NULL DEFAULT 0,
    requested_price         NUMERIC(24,8),
    average_execution_price NUMERIC(24,8),
    stop_price              NUMERIC(24,8),
    target_price            NUMERIC(24,8),
    status                  TEXT NOT NULL,
    strategy_name           TEXT NOT NULL,
    signal_id               UUID NOT NULL,
    risk_decision_id        UUID NOT NULL,
    failure_reason          TEXT NOT NULL DEFAULT '',
    version                 INTEGER NOT NULL DEFAULT 1,
    created_at              TIMESTAMPTZ NOT NULL,
    updated_at              TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX orders_client_order_id_idx ON orders (client_order_id);
CREATE INDEX orders_status_idx ON orders (status);
CREATE INDEX orders_signal_id_idx ON orders (signal_id);
CREATE INDEX orders_created_at_idx ON orders (created_at DESC);
