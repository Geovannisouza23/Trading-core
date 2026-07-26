# ADR 0010: Real gRPC Client Against quant-engine, All 15 RPCs

## Status

Accepted. Supersedes ADR 0008's proto-source assumption.

## Context

ADR 0008 anticipated exactly this moment ("when the Rust engine exists,
connecting it is: run `make proto`, write a gRPC client...") but pointed
at a local placeholder proto (`internal/contracts/grpc/quant/quant_engine.proto`,
2 RPCs, a 3-state `MarketRegime`) written before `quant-engine` existed.
`quant-engine` now exists, is fully implemented and tested, and its own
proto (`proto/quant/v1/quant_engine.proto`, `QuantEngineService`, 15 RPCs,
a 10-state `MarketRegime`) deliberately supersedes and is **not**
byte-compatible with that placeholder — a redesign `quant-engine`'s own
ADR 0001 states was authorized by this repository's ADR 0008.

Two questions this ADR answers: where the generated Go code's source
proto now comes from, and how far "connect the gRPC client" goes — just
the 2 RPCs the realtime signal path already used, or all 15.

## Decisions

**Proto source moves to a vendored copy of quant-engine's canonical
contract.** `internal/contracts/grpc/quant/v1/quant_engine.proto` is now
a byte-for-byte copy of `../quant-engine/proto/quant/v1/quant_engine.proto`,
with one intentional deviation: the `go_package` option is rewritten to
`trading-core/internal/contracts/grpc/quant/v1` (Go tooling metadata, not
part of the wire contract — the source file's own header comment already
names this exact path as "reference for a future Go-side migration"). The
old placeholder (`internal/contracts/grpc/quant/quant_engine.proto` +
the empty `generated/` package) is deleted, not kept alongside. `make
proto` regenerates from the vendored copy; re-sync it manually if the
Rust side changes the contract (poly-repo, no submodule linkage).

**All 15 RPCs get a real client, not just the 2 the realtime path uses.**
`EvaluateSignal`/`AnalyzeMarketRegime` continue satisfying
`output.QuantEngine` exactly as ADR 0008 described. The other 13
(backtest, optimization, dataset export, strategy validation, direct
feature/model access, decision-outcome feedback, model-registry
introspection/reload) get their own segregated output ports
(`output.BacktestService`, `output.OptimizationService`,
`output.DatasetExportService`, `output.QuantDiagnostics` —
`internal/application/ports/output/quant_{backtest,optimization,dataset,diagnostics}.go`),
new use cases delegating straight through to them
(`internal/application/usecase/quant_*.go`), and new HTTP endpoints under
`/v1` (`backtest`, `optimization`, `dataset`, `strategy`, `quant/*`,
`decisions/{id}/outcome` — see `docs/openapi.yaml`), gated by the same
`operator`/`admin` role the existing `/v1/backtest/request` stub already
required. `/v1/backtest/request` itself goes from a deliberate 501 to a
real submission — the same handler package, the same route, now backed
by a real call instead of an honest "not implemented."

**One shared gRPC connection, not five.** All five quant output ports are
backed by the same `*grpc.Client` instance when `QUANT_MODE=grpc`
(`internal/app/providers.go::provideQuantGrpcClient`, `fx.Lifecycle`-
managed) — never five separate `grpc.ClientConn`s for one logical
dependency.

**No fake for the 13 new capabilities.** The realtime path keeps its
deterministic in-process fake (`fake.go::Engine`) — trivial to fake
convincingly, and every prior test suite already depends on it working
offline. There is no equally honest way to fake "run a backtest" or
"reload a model," so the 13 new capabilities are backed by
`quantfake.Unavailable` in every mode except `grpc`: every method returns
a clear `fmt.Errorf` naming the capability and how to enable it, rather
than a fabricated response. This mirrors the exact precedent the
pre-integration `/v1/backtest/request` handler already set (a clean 501,
never a fake "queued" response) — extended to all 13 rather than
special-cased for one.

**`RegisterDecisionOutcome` stays caller-supplied, not auto-triggered.**
The wire RPC exists to feed realized trade outcomes back into
quant-engine's training data. Auto-triggering it from `trading-core`'s
own order reconciliation would require threading quant-engine's
`signal_id`/`decision_id` through `output.QuantSignal` →
`signal.TradeSignal` → the orders table (a domain change and a migration)
— out of scope here. `POST /v1/decisions/{decision_id}/outcome` accepts
the outcome fields directly from the caller instead.

**`QUANT_MODE` default stays `fake`.** Enabling `grpc` mode is a deploy-
time operator decision (`QUANT_MODE=grpc`, `QUANT_GRPC_TARGET=...`), not
a new default — flipping the default would break every existing local
PAPER-mode run that doesn't have `quant-engine` running alongside it.

**`MarketRegime` (10→4) and `Action`→`Side`/`HasSignal` mappings** are
unchanged from what ADR 0008 anticipated in spirit, now concretely
implemented in `internal/infrastructure/external/quant/grpc/mapper.go`:
`TRENDING_UP`/`TRENDING_DOWN`/`BREAKOUT` → `Trending`; `RANGING` →
`RangeBound`; `HIGH_VOLATILITY`/`LOW_VOLATILITY`/`PANIC` → `Volatile`;
`ILLIQUID`/`NEWS_EVENT`/`UNSPECIFIED`/`UNKNOWN` → `Unknown`. `HOLD`/
`UNSPECIFIED` action → `HasSignal=false`; `BUY`/`SELL` → `Side` + parsed
prices.

## Consequences

- `google.golang.org/grpc` is now a direct dependency (previously only
  pulled in indirectly).
- `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` are now required to
  run `make proto` — installed without `sudo` in this delivery (the
  `protoc` release binary into a user-local bin directory,
  `protoc-gen-go`/`protoc-gen-go-grpc` via `go install`).
- Bindings exist for all 15 RPCs' messages (protoc generates the whole
  service client regardless of what's wired), but only 15/15 are actually
  called from application code now — none are dead weight.
- Backtest/optimization/dataset-export/feature-calculation requests
  accept `candles[]` directly in the HTTP request body — `trading-core`
  does not fetch historical candle data on the caller's behalf, mirroring
  quant-engine's own gRPC contract exactly (which also takes `candles[]`
  as an explicit request field). A candle-history repository/backfill
  capability is a separate, larger feature this ADR does not build.
- `StreamBacktestProgress` (the one server-streaming RPC) is exposed over
  HTTP as Server-Sent Events (`GET /v1/backtest/{id}/stream`), closing
  after the first terminal status — the same semantics the underlying RPC
  already has.
