# Architecture

`trading-core` is the Go operational core of an automated trading robot. It
owns every decision that can touch money — signal evaluation, risk, order
execution, reconciliation — while the quantitative model (Rust), the
dashboard (TypeScript) and cloud infrastructure are out of scope for this
delivery and are represented only by ports, contracts and adapters.

## Layers and dependency direction

```
interfaces  →  application  →  domain
infrastructure  →  application/ports/output  →  domain
```

- **`internal/domain`** — entities, aggregates, value objects, domain
  events, specifications, the risk policy, and the order/operational-mode
  state machines. Pure Go plus `github.com/shopspring/decimal` and
  `github.com/google/uuid`. It imports nothing from any other layer and
  nothing from a framework — enforced by
  `tests/architecture/layering_test.go`, not just convention.
- **`internal/application`** — use cases (`usecase/`), their input ports
  (`ports/input`, one interface per use case) and the output ports
  (`ports/output`) infrastructure must implement, plus `command/`, `query/`,
  `dto/`, `mapper/`. Depends only on `domain`. Never imports a concrete
  adapter or a framework.
- **`internal/interfaces`** — inbound adapters: the HTTP API
  (`http/{handler,middleware,router,presenter,request,response}`), the
  public dashboard WebSocket (`websocket/{hub,connection,handler}`), and the
  consumers (`consumer/{market,order,event,outbox}`). Depends on
  `application` only, through its ports.
- **`internal/infrastructure`** — outbound adapters implementing
  `application/ports/output`: PostgreSQL (`database/`), external services
  (`external/{broker,quant,llm,marketdata,news,notification,cache,storage,
  messaging}`), and cross-cutting concerns (`observability/`, `clock/`).
- **`internal/app`** — the only other place (besides `cmd/`) allowed to
  import Uber Fx. It wires every concrete adapter to the port it satisfies
  and owns the process lifecycle (HTTP server, background workers, initial
  DB bootstrap).

### Why `interfaces` is a separate top-level package from `external`

They are both "adapters" in the hexagonal sense, but for opposite
directions of traffic. `interfaces` is what the outside world calls *into*
the system (HTTP, the dashboard WebSocket, consumers). `external` is what
the system calls *out* to (a broker, an LLM, an exchange's own WebSocket
feed). Conflating them would blur a genuinely useful rule: nothing that
receives inbound traffic is allowed to also hold broker/DB credentials or
open a transaction directly — `tests/architecture` enforces exactly that
boundary for HTTP handlers and consumers.

### Why there is exactly one `database` folder

Every PostgreSQL concern — the pool, migrations, the transaction manager,
and one repository package per aggregate — lives under
`internal/infrastructure/database`. A second "database" or "persistence"
folder anywhere else would mean two independent connection strategies, two
places to look up how a query is built, and no single owner of schema
evolution. `tests/architecture/layering_test.go` fails the build if a
second `database`, a `persistence`, or a `bootstrap` directory ever
reappears under `internal/`.

### How Uber Fx is confined to composition

`domain` and `application` never import `go.uber.org/fx` — verified by the
architecture tests. Fx appears in exactly three places: `cmd/*/main.go`
(two lines: build the module, call `.Run()`), `internal/app/*.go` (every
provider and lifecycle hook), and one explicitly allow-listed composition
file, `internal/infrastructure/database/module.go`, which owns the pgx
pool's Fx lifecycle because a pool's connect/close hooks are tightly
coupled to how it's constructed. Nothing else needs Fx to compile or run in
isolation, which is what keeps `go test ./internal/domain/...` and
`./internal/application/...` fast and dependency-free.

## The PAPER trade pipeline, end to end

```
market data feed (Binance public kline WebSocket, no credentials needed)
  → interfaces/consumer/market.Consumer.HandleCandle
      → usecase.EvaluateMarketSignal   (calls the Quant Engine port)
          → persists TradeSignal, publishes SignalCreated
      → usecase.EvaluateRisk           (CompositeRiskPolicy + PositionSizer)
          → persists RiskDecision, publishes RiskDecisionCreated
      → usecase.ExecuteApprovedOrder   (the ONLY use case allowed to call Broker.PlaceOrder)
          → Pending order + outbox row, same transaction
          → Broker.PlaceOrder (PaperBroker simulates the fill)
          → Filled order + Position opened + outbox rows, same transaction
          → Broker.PlaceStop (best-effort; failure raises an incident, never rolls back the fill)
```

The market and event consumers call use cases directly (sequencing, not
business logic — the business logic is entirely inside the use cases).
Order/position lifecycle events required to survive a crash
(`OrderCreated`, `OrderFilled`, `PositionOpened`, ...) go through the
transactional outbox; `SignalCreated`/`RiskDecisionCreated` are published
directly to the in-memory `EventBus` since they're not in the spec's
outbox-mandatory list and are consumed synchronously in-process. The outbox
worker (`internal/infrastructure/database/outbox/worker.go`) drains
pending rows on a 2s interval and republishes them onto the `EventBus`;
`interfaces/consumer/outbox` subscribes to that bus and forwards every
dashboard-relevant event to the WebSocket hub.

**Neither the LLM nor the Quant Engine can place an order.** The Quant
Engine (`output.QuantEngine`) only proposes a candidate signal.
`EvaluateRisk` is the only place a signal is approved or rejected, and
`ExecuteApprovedOrder` is the only place anything reaches
`output.Broker.PlaceOrder`.

## Risk

`internal/domain/risk` implements the twelve required specifications
(Specification pattern) combined by `CompositeRiskPolicy`, which never
short-circuits — every specification runs so operators see every reason a
decision was blocked, not just the first. Position sizing is a Strategy
(`PositionSizer`); `FixedRiskPositionSizing` and `ATRPositionSizing` both
ship. Thresholds arrive as domain value objects
(`risk.Thresholds`), built once at startup in `internal/app/providers.go`
from `internal/config.RiskConfig` — the domain package itself never reads
configuration.

## Idempotency and concurrency

- **Orders**: `ClientOrderID` is derived deterministically from the
  `RiskDecisionID` (a 36-character UUID string, which also satisfies the
  `ClientOrderID` format), so retrying `ExecuteApprovedOrder` for the same
  decision is naturally idempotent — `OrderRepository.GetByClientOrderID`
  short-circuits before anything is sent to a broker.
- **Concurrent execution of the same decision**: guarded by
  `IdempotencyRepository.Reserve`, backed by a `(scope, key)` primary key —
  the database, not application logic, decides who wins a race.
- **Optimistic locking**: `Account`, `Order`, `Position`, and the
  operational-mode singleton all carry a `version` column; every `UPDATE`
  is `WHERE version = <expected previous>`, and zero rows affected surfaces
  as `output.ErrOptimisticLock`.
- **The outbox**: `FetchPendingBatch` is a single atomic
  `UPDATE ... FROM (SELECT ... FOR UPDATE SKIP LOCKED)` statement, so two
  worker processes racing for the same pending rows always partition them
  — proven directly in
  `tests/integration/outbox_worker_test.go::TestOutboxConcurrentWorkersNeverClaimTheSameRow`.
- **Unknown broker state**: if `PlaceOrder` itself errors, the use case
  calls `FindOrderByClientOrderID` before doing anything else. Found → the
  order went through, proceed as submitted. Confirmed absent → safe to mark
  `Failed`. Anything else (a second network error, a timeout) → `Unknown`,
  plus a `SystemIncident`, and the order is left for
  `ReconcileBrokerState` to resolve on its next pass. It is never retried
  blindly.

## Operational modes

`PAPER → TESTNET → REAL → PAUSED → CLOSE_ONLY → KILL_SWITCH`, enforced by
`domain/operation.Mode.CanTransition`. Two properties matter most:

- **REAL is unreachable from CLOSE_ONLY or KILL_SWITCH**, and reaching it
  from anywhere requires a `RealModeConfirmation` whose token must equal an
  exact, non-configurable string (`operation.RealModeConfirmation.Validate`)
  — checked by the domain itself, not by a caller that could forget to
  check it.
- **REAL has no HTTP endpoint.** Section 15's endpoint list only exposes
  pause/resume/close-only/kill-switch; there is no `POST` that lets an HTTP
  caller pick an arbitrary target mode. The only caller allowed to request
  `REAL` is `internal/app`'s bootstrap, sourcing the confirmation from
  validated `internal/config` values — themselves gated by
  `BROKER_REAL_TRADING_ENABLED=true` plus a non-empty
  `BROKER_REAL_TRADING_CONFIRMATION`. Three independent layers (config
  validation, the domain state machine, and the `real` broker adapter's own
  constructor) all refuse to proceed without both.

## Testing strategy

| Suite | What it proves | Needs Docker? |
|---|---|---|
| `tests/unit` | Value object invariants, `Order`/`OperationalMode` state machines, risk specifications, position sizing, `PaperBroker` fill/slippage/idempotency logic | No |
| `tests/architecture` | The dependency-direction and folder rules above, checked via `go list -json`, not by convention | No |
| `tests/contract` | Binance request signing (HMAC-SHA256, replayed against a captured request), the Quant fake's output always satisfies the domain's `TradeSignal` invariants | No |
| `tests/integration` | Real PostgreSQL: optimistic locking, unique constraints, transaction rollback, outbox claim/retry semantics under concurrency | Yes (Testcontainers) |
| `tests/failure` | Unreachable database, a slow broker past its deadline, Gemini returning invalid/out-of-schema JSON, two goroutines racing `ExecuteApprovedOrder` for the same decision | Mixed — see each file's build tag |

`tests/fixtures` holds the one shared Testcontainers Postgres helper both
`integration` and `failure` (build-tag-gated) tests use.

## Extending the system

**Add a new broker.** Implement `application/ports/output.Broker` under
`internal/infrastructure/external/broker/<name>`, add a `case` to
`provideBroker` in `internal/app/providers.go`. Nothing in `application` or
`domain` changes.

**Connect the real Rust Quant Engine.** The contract already exists at
`internal/contracts/grpc/quant/quant_engine.proto`. Install `protoc`,
`protoc-gen-go`, `protoc-gen-go-grpc` and run `make proto` to populate
`internal/contracts/grpc/quant/generated`; write a client implementing
`output.QuantEngine` under `internal/infrastructure/external/quant/grpc`
(replacing the current in-process fake in that same package), and switch
`provideQuantEngine` on `cfg.Quant.Mode == "grpc"`.

**Replace the in-memory Event Bus.** Implement `output.EventBus` — see
`internal/infrastructure/external/messaging/pubsub` for the placeholder
shape — and swap the constructor `provideEventBus` uses in
`internal/app/providers.go`. No use case, consumer, or handler changes,
because all of them depend on the `output.EventBus` interface, never on
`messaging/memory` directly.

**Move from PaperBroker to Testnet.** Set `BROKER_MODE=TESTNET` plus
`BROKER_BINANCE_API_KEY`/`BROKER_BINANCE_API_SECRET` (a Binance Futures
*testnet* key, never a real one). See
`deployments/compose/docker-compose.testnet.yml` for a ready override.

**Enable REAL mode.** Read `docs/decisions/` first. At minimum:
`BROKER_MODE=REAL`, real `BROKER_BINANCE_API_KEY`/`_SECRET`,
`BROKER_REAL_TRADING_ENABLED=true`, and
`BROKER_REAL_TRADING_CONFIRMATION=I_UNDERSTAND_THE_RISK` — and only after
you've reviewed position sizing, the risk thresholds in
`internal/config/risk.go`'s defaults, and reconciliation behavior against
your own account. This codebase gives you the guardrails; it does not
decide for you that real capital is ready to be at risk.

## Running it

```
make docker-up      # postgres + migrate (one-shot) + api + worker
curl localhost:8080/health
curl localhost:8080/ready
```

Or locally without Docker: start a Postgres matching `.env.example`, then
`make migrate-up && make run` (and, optionally, `make worker` in a second
terminal). See the root `README.md` for the full walkthrough, including how
to mint a JWT for the authenticated endpoints.
