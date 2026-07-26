# ADR 0011: Real NATS Client + Redis (Market Data Cache, Cross-Service Idempotency)

## Status

Accepted. Partially supersedes [ADR 0009](0009-in-memory-event-bus-for-mvp.md):
the in-memory `EventBus` is no longer the only implementation, but it
remains the fallback whenever NATS is unreachable and `NATS_REQUIRED=false`
(the default) — ADR 0009's reasoning for why that fallback is safe still
holds.

## Context

quant-engine (Rust) already runs real NATS JetStream: it publishes 6
event families and runs 2 durable consumers
(`quant.decision.outcome.received`, `quant.model.approved`) — see its own
[0006-consumer-scope.md](../../../quant-engine/docs/decisions/0006-consumer-scope.md).
That ADR explicitly notes both consumers' subjects were "proposed
contracts... no publisher for either is confirmed to exist yet on the
trading-core side" — trading-core's own messaging package
(`internal/infrastructure/external/messaging/pubsub`) was a dead Google
Cloud Pub/Sub placeholder that was never NATS and never wired to
anything.

Separately, Redis existed in trading-core as an unwired `go-redis` client
(`internal/infrastructure/external/cache/redis/client.go`) with no
application port, no config, no docker-compose service, and no call
sites — and did not exist at all on the quant-engine side.

## Decision

**NATS**: `internal/infrastructure/external/messaging/pubsub` is deleted.
A real `nats.go`/JetStream client
(`internal/infrastructure/external/messaging/nats`) replaces it,
mirroring quant-engine's own optionality contract exactly:
`NATS_REQUIRED=false` by default, an unreachable broker at startup falls
back to `messaging/memory.NewEventBus()` (a warning, not a boot failure),
`NATS_REQUIRED=true` makes it a hard startup dependency instead. The
`{prefix}_EVENTS` JetStream stream (`CreateOrUpdateStream`, subject
filter `{prefix}.>`) is provisioned idempotently by whichever side
connects first.

Every message this package publishes uses the same `EventEnvelope` shape
quant-engine's `contracts::events::EventEnvelope<T>` defines
(`event_id`, `event_type`, `event_version`, `aggregate_id`,
`aggregate_type`, `aggregate_version`, `correlation_id`, `causation_id`,
`idempotency_key`, `occurred_at`, `producer`, `payload`) — see
`internal/infrastructure/external/messaging/nats/envelope.go`.

Two fixed-subject publishers close the gap ADR 0006 (quant-engine) flags:
`Publisher.PublishDecisionOutcome`/`PublishModelApproved` write to
`quant.decision.outcome.received`/`quant.model.approved`, reusing the
existing `RegisterDecisionOutcomeInput`/`ReloadApprovedModelInput` types
from the gRPC integration (ADR 0010) rather than duplicating DTOs — this
is the same operation over a different transport, exposed as two new
`POST .../publish` endpoints alongside the existing synchronous RPC
endpoints.

A new consumer, `internal/interfaces/consumer/quantevents`, subscribes
(via a durable pull-consumer, `FilterSubjects` on
`{prefix}.backtest.*.events` / `{prefix}.optimization.*.events`,
`DeliverNewPolicy` so a fresh consumer doesn't replay 30 days of
backlog) to quant-engine's own `BacktestCompleted`/`BacktestFailed`/
`OptimizationCompleted` events and forwards them to the existing
WebSocket hub — real-time job-completion notification in place of
dashboard polling. It is wired only into the API process
(`internal/app/application.go`'s `Module()`), not the worker, for the
same reason the market data feed is API-only: it needs the hub, and
running a second puller against the same durable consumer without a hub
to broadcast to would silently steal messages from the instance that can
actually use them.

**Redis**: two concrete, scoped uses — not a general-purpose cache.

1. `output.MarketDataCache` (`GetCandles`/`SetCandles`), backed by
   `internal/infrastructure/external/cache/redis/market_data_cache.go`.
   Wired into `runMarketDataFeed` (`internal/app/lifecycle.go`): every
   closed candle is written through after being handed to the consumer
   pipeline, and a fresh process warm-starts its `recentCandles` window
   from the cache instead of an empty slice — covers a process
   restart/crash, which the pre-existing in-memory-only window did not.
2. A second, Redis-backed binding of the existing
   `output.IdempotencyRepository` interface
   (`internal/infrastructure/external/cache/redis/idempotency_repository.go`,
   atomic `SET NX EX`), scoped to `"nats-publish"` and injected only into
   `QuantEventPublisherService` via a named Fx binding
   (`natsPublishIdempotency`) — it does **not** replace the
   Postgres-backed `IdempotencyRepository` used by order execution and
   market-event processing.

Both Redis uses are optional (`REDIS_REQUIRED=false` by default,
mirroring the NATS/database pattern): an unreachable Redis degrades the
market feed to its pre-existing empty-start, no-cache behavior, and
disables the extra publish-side dedup guard — it never blocks the
synchronous request path.

## Topology

Both repositories keep their own `docker-compose.yml` and their own
isolated Postgres (different databases, users, ports — unchanged). A new
external Docker network, `aegis-net`, is the only thing shared: both
compose files declare it as `external: true` and attach the services
that need it (quant-engine's `nats`/`redis`/workers; trading-core's
`api`/`worker`). One-time setup: `docker network create aegis-net`
before bringing up either stack — see each repo's `docker-compose.yml`
comments. quant-engine remains the "owner" of the shared NATS/Redis
containers (it already ran NATS); trading-core's `NATS_URL`/`REDIS_ADDR`
point at quant-engine's service names (`nats:4222`, `redis:6379`),
resolvable across the two Compose projects via Docker's embedded DNS
because they share the network.

## Consequences

- quant-engine's `decision_outcome_received`/`model_approved` consumers
  now have a real publisher — ADR 0006 (quant-engine)'s "proposed
  contract" caveat for those two subjects is resolved from the
  trading-core side.
- The in-memory `EventBus` (ADR 0009) is still exactly what runs when
  nobody has configured NATS — local development and tests are
  unaffected.
- Cross-service idempotency is deliberately narrow: only the NATS-publish
  path gets the Redis-backed guard. Extending it to other call sites is
  a separate decision, not implied by this one.
- Not addressed here (tracked separately, same boundaries ADR 0010
  already drew): auto-triggering `quant.decision.outcome.received` from
  order reconciliation, and publishing quant-engine's other event
  families that trading-core has no consumer for today.
