# ADR 0009: In-Memory Event Bus for the MVP

## Status

Accepted.

## Context

The system needs to fan events out to the dashboard WebSocket, and
potentially to future consumers (metrics, notifications, a second
service), without every use case knowing who is listening. Standing up
Kafka, NATS, or Google Pub/Sub for a single-process MVP would add
operational surface (another service to run, another failure mode to
handle in `docker compose up`) with no present benefit — nothing in this
delivery runs as more than one API/worker pair talking to one Postgres.

## Decision

`application/ports/output.EventBus` is a two-method interface
(`Publish`/`Subscribe`) with exactly one implementation wired today:
`internal/infrastructure/external/messaging/memory`, a mutex-protected
map of event name to handler list, invoked synchronously and
in-process. `internal/infrastructure/external/messaging/pubsub` exists as
the documented placeholder for the future swap and returns a clear
"not configured" error rather than silently no-op-ing.

## Consequences

- Zero extra infrastructure to run this system locally or in
  `docker compose up`.
- Events do not survive a process restart and are not visible across
  multiple API replicas — acceptable because the events this bus carries
  (`SignalCreated`, `RiskDecisionCreated`, and the outbox-forwarded
  lifecycle events) are either re-derivable from Postgres or already
  durably persisted before being republished onto this bus by the outbox
  worker, so nothing is silently lost.
- Swapping in Pub/Sub, NATS, or Kafka later is exactly one adapter
  (implement `output.EventBus`) plus one line in
  `internal/app/providers.go`'s `provideEventBus` — no use case, consumer,
  or handler changes, because none of them import `messaging/memory`
  directly.
