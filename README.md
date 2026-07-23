# trading-core

Go operational core of an automated trading robot. It owns market data
ingestion, signal evaluation (via an external Quant Engine port), a fully
deterministic risk manager, order execution, position/account state,
broker reconciliation, and the HTTP/WebSocket API a dashboard would consume
— all in Clean Architecture with the dependency direction
`interfaces → application → domain` and `infrastructure → application/ports
/output → domain`.

The Rust quantitative engine, the TypeScript dashboard, and cloud
infrastructure are **out of scope** for this delivery and are represented
only by ports, a `.proto` contract, and fake/paper adapters. Everything
under this directory is real, compiled, tested Go.

**Start here:** [`docs/architecture.md`](docs/architecture.md) explains the
layers, the trading pipeline, idempotency/concurrency guarantees, and how
to extend each adapter. [`docs/decisions/`](docs/decisions) has one ADR per
major architectural choice. This README covers running and testing it.

## Quick start (Docker)

```bash
cp .env.example .env   # optional — docker-compose.yml already has working PAPER defaults
make docker-up         # postgres + migrate (one-shot) + api + worker
curl localhost:8080/health
curl localhost:8080/ready
```

`docker compose up` starts, in order: `postgres` (with a health check),
`migrate` (runs `migrate up` once and exits), then `api` and `worker`,
both waiting on `migrate` to finish successfully. The system starts in
**PAPER mode** with a $10,000 simulated account, exactly as required.

To stop and wipe the database volume: `make docker-down`.

## Quick start (local, no Docker)

Requires a running PostgreSQL 16 reachable with the credentials in
`.env.example` (or your own — export the `DATABASE_*` variables to match).

```bash
make migrate-up   # applies every migration in internal/infrastructure/database/migrations
make run          # starts the API on :8080
make worker       # (optional, separate terminal) outbox drain + reconciliation
```

> **Environment note:** `go build`/`go run` on this machine may fail with
> `error obtaining VCS status: exit status 128` if this checkout happens to
> sit inside an unrelated or partially-initialized git repository higher up
> the directory tree. Every Makefile target already passes
> `-buildvcs=false` to sidestep this; if you invoke `go` directly, do the
> same.

## Calling the authenticated API

Every `/v1/*` endpoint requires an `Authorization: Bearer <JWT>` header —
an HS256 token signed with `SECURITY_JWT_SIGNING_SECRET`, with a `roles`
claim (`["viewer"]` is enough for the `GET` endpoints; `POST
/v1/system/*`, `/v1/paper/reset`, and `/v1/backtest/request` need
`"operator"` or `"admin"`). Mint one locally with any JWT library, or with
`jwt-cli`:

```bash
# using https://github.com/mike-engel/jwt-cli, matching the default dev secret
jwt encode --secret local-development-secret-change-me-32chars \
  --sub "local-operator" --payload roles='["admin"]'
```

```bash
curl -H "Authorization: Bearer $TOKEN" localhost:8080/v1/dashboard
```

`/health`, `/ready`, `/metrics`, and the dashboard WebSocket at `/ws` are
intentionally unauthenticated (see
[`docs/architecture.md`](docs/architecture.md) and the WebSocket handler's
comment for why).

## Configuration

Every setting is an environment variable, loaded and validated exactly
once at startup in `internal/config` — no other package reads `os.Getenv`
(enforced by `tests/architecture`). See `.env.example` for the most common
ones and their PAPER-safe defaults, and `internal/config/*.go` for the
full, authoritative list and validation rules.

## Testing

```bash
make test-unit          # domain + application logic — no external dependencies
make test-architecture  # dependency-direction/layering rules, via go list -json
make test-contract      # Binance request signing, Quant fake output contract
make test-integration   # real Postgres via Testcontainers — needs Docker
make test-failure       # unreachable DB, broker timeout, invalid LLM JSON, concurrent execution races
make test               # all of the above
```

`test-unit`, `test-architecture`, and `test-contract` need nothing beyond
the Go toolchain. `test-integration` and part of `test-failure` (anything
tagged `//go:build integration`) start a disposable PostgreSQL container
via Testcontainers and therefore need a working Docker daemon.

> This delivery was built and verified in a sandboxed environment without
> an accessible Docker daemon. Every package builds
> (`go build ./...`), every layering rule is enforced
> (`go vet` + `go test ./tests/architecture/...`), and the full Uber Fx
> dependency graph was confirmed to resolve correctly by running the
> compiled `api` binary against no database — it failed at exactly the
> expected point (`pool.Ping`), with **zero** dependency-injection errors
> across all ~50 provided types. The Postgres-backed integration and
> failure tests are written and type-checked (`go vet -tags=integration`)
> but have not been executed live; run `make test-integration
> test-failure` yourself with Docker available to confirm them end to end.

## Migrations

```bash
make migrate-up                # apply everything pending
make migrate-down N=1          # revert the last N migrations (default 1)
make migrate-version           # print the current schema version
make migrate-force V=3         # stamp the tracked version without running SQL (recovery only)
```

Migration files live only at
`internal/infrastructure/database/migrations/*.{up,down}.sql`, embedded
into the binary via `go:embed` — `cmd/migrate` needs no filesystem access
to the source tree to run.

## Safety checklist before any real capital

This codebase gives you the guardrails (deterministic risk manager,
multi-layer REAL-mode gating, reconciliation, kill switch, idempotent
execution) — it does not decide *for* you that real money is ready to be
at risk. Before setting `BROKER_MODE=REAL`:

1. Run PAPER for long enough to trust the Quant Engine you actually plug
   in (the shipped fake is a placeholder, not a strategy).
2. Review every default in `internal/config/risk.go` against your own risk
   tolerance — they match the spec's example values, not necessarily
   yours.
3. Run TESTNET first, with the exact strategy and risk config you intend
   for REAL, and watch `ReconcileBrokerState` for a real trading cycle.
4. Confirm your Binance API key has **no withdrawal permission**.
5. Only then set `BROKER_REAL_TRADING_ENABLED=true` and
   `BROKER_REAL_TRADING_CONFIRMATION=I_UNDERSTAND_THE_RISK` — and note
   there is deliberately no HTTP endpoint that can flip this for you; it's
   a deploy-time decision, not a runtime one.

## Known gaps in this delivery

- **`POST /v1/backtest/request`** returns `501 Not Implemented`. No
  backtest engine or use case is defined anywhere in the originating spec
  beyond this one endpoint; faking a "queued" response would have been
  dishonest about what actually happens.
- **News ingestion isn't scheduled.** `ProcessMarketEvent`, the RSS
  provider (`infrastructure/external/news/rss`), and the event consumer
  (`interfaces/consumer/event`) are all implemented and unit/failure
  tested, but nothing in `internal/app` polls a feed on a timer yet — no
  default feed URLs were specified. Wiring a scheduler is a small addition
  to `internal/app/lifecycle.go` once you have feed URLs to poll.
- **`output.Broker.GetOrder`/`FindOrderByClientOrderID`** on the Binance
  adapter need a symbol Binance's API requires but the given port
  signature doesn't carry; the adapter keeps an in-process id→symbol
  cache populated by `PlaceOrder`/`PlaceStop`, documented directly in
  `internal/infrastructure/external/broker/binance/broker.go`.
- **Quant Engine and LLM health** on the dashboard report `true`
  unconditionally (see `usecase.QueryService`) — both active adapters
  (fake, noop) are always available by construction; a real gRPC/Gemini
  health probe would replace that constant.

## Repository layout

See [`docs/architecture.md`](docs/architecture.md) for the annotated tour;
the short version:

```
cmd/               three entrypoints: api, worker, migrate
internal/
  app/              Uber Fx composition + lifecycle (the only other Fx-importing code besides cmd/)
  config/           env loading + validation — the only os.Getenv callers
  contracts/        the Quant Engine .proto contract (uncompiled — see ADR 0008)
  domain/           entities, value objects, specifications, state machines — framework-free
  application/      use cases, ports, DTOs, mappers
  interfaces/       HTTP API, dashboard WebSocket, consumers
  infrastructure/   Postgres, PaperBroker + Binance client, Quant fake, LLM noop/Gemini, observability
tests/              unit, architecture, contract, integration, failure, fixtures
docs/               architecture.md, ADRs, openapi.yaml
deployments/        docker/compose variants beyond the root Dockerfile/docker-compose.yml
```
