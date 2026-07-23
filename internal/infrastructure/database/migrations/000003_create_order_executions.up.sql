CREATE TABLE order_executions (
    id          UUID PRIMARY KEY,
    order_id    UUID NOT NULL REFERENCES orders (id),
    quantity    NUMERIC(24,8) NOT NULL,
    price       NUMERIC(24,8) NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX order_executions_order_id_idx ON order_executions (order_id);
