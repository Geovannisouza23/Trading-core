CREATE TABLE risk_decisions (
    id                     UUID PRIMARY KEY,
    signal_id              UUID NOT NULL REFERENCES trade_signals (id),
    allowed                BOOLEAN NOT NULL,
    original_position_size NUMERIC(24,8) NOT NULL,
    approved_position_size NUMERIC(24,8) NOT NULL,
    reason_codes           TEXT[] NOT NULL DEFAULT '{}',
    applied_rules          TEXT[] NOT NULL DEFAULT '{}',
    event_restrictions     TEXT[] NOT NULL DEFAULT '{}',
    created_at             TIMESTAMPTZ NOT NULL
);

CREATE INDEX risk_decisions_signal_id_idx ON risk_decisions (signal_id);
CREATE INDEX risk_decisions_created_at_idx ON risk_decisions (created_at DESC);
