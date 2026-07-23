# ADR 0004: Transactional Outbox for Order/Position Lifecycle Events

## Status

Accepted.

## Context

`ExecuteApprovedOrder` must both change durable state (an `Order` row) and
notify the rest of the system (the dashboard WebSocket, metrics, future
notification rules) that it did. Publishing to an event bus and writing to
Postgres are two different systems; if either step happens without the
other — a crash between the `UPDATE` and the publish, or a publish that
succeeds right before a rollback — the dashboard silently drifts from
reality. That is exactly the kind of divergence `ReconcileBrokerState`
exists to catch, but catching it after the fact is strictly worse than not
creating it.

## Decision

`OrderCreated`, `OrderSubmitted`, `OrderPartiallyFilled`, `OrderFilled`,
`OrderCancelled`, `PositionOpened`, `PositionClosed`, `KillSwitchActivated`,
`CriticalEventDetected`, and `ReconciliationDivergenceDetected` (spec
section 20's exact list) are written to an `outbox_events` row in the
**same transaction** as the aggregate write that produced them
(`output.TransactionManager.WithinTransaction`). A separate worker
(`internal/infrastructure/database/outbox/worker.go`) polls for pending
rows and publishes them to `output.EventBus`, retrying up to 5 times before
marking a row permanently `FAILED`. `SignalCreated` and
`RiskDecisionCreated` are *not* outbox events — they are internal pipeline
signals consumed synchronously in the same process and are not in spec
section 20's mandatory list.

## Consequences

- The aggregate write and the "this happened" record can never diverge:
  either both commit or neither does.
- The dashboard can lag behind the database by up to the worker's poll
  interval (2s) but can never show something that didn't actually happen.
- `FetchPendingBatch` is a single atomic
  `UPDATE ... FROM (SELECT ... FOR UPDATE SKIP LOCKED)` statement
  specifically so that running two outbox workers concurrently (e.g. one in
  `cmd/api`, one in `cmd/worker`) is safe rather than merely "usually fine"
  — proven in `tests/integration/outbox_worker_test.go`.
