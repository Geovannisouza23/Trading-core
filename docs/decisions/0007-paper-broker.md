# ADR 0007: A Fully Functional PaperBroker as the Default Adapter

## Status

Accepted.

## Context

The system must run and be demonstrable end-to-end without exchange
credentials, and PAPER must be the safe default an operator falls into,
not TESTNET or REAL. A broker adapter that only returns
`ErrNotImplemented` would make the "paper trading flow works end-to-end"
acceptance criterion untestable without external network access.

## Decision

`internal/infrastructure/external/broker/paper` is a real, self-contained
`output.Broker` implementation: an in-memory ledger (balance, positions,
open orders) protected by a mutex, configurable fees and slippage, fill
simulation for market orders, resting (never auto-triggered — there is no
live price feed inside the simulator itself) stop orders, and
`ClientOrderID`-keyed idempotency identical in shape to what a real
exchange would need to provide. `BROKER_MODE` defaults to `PAPER`, and
`ResetPaperState`/`POST /v1/paper/reset` can only target this adapter
(enforced by a `PaperResettable` type assertion, not a config flag).

## Consequences

- `docker compose up` with zero credentials configured produces a system
  that can actually take a signal through risk evaluation, execution, and
  a filled position — not a stub that always returns "not implemented".
- Testnet and Real (`internal/infrastructure/external/broker/{testnet,
  real}`) share a real Binance Futures REST client
  (`.../broker/binance`) rather than duplicating HMAC signing and response
  parsing, but neither is exercised unless `BROKER_MODE` and the matching
  credentials are explicitly set — and Real additionally refuses to
  construct at all without `BROKER_REAL_TRADING_ENABLED=true` and a
  non-empty confirmation phrase, independent of the domain-level check on
  the same token.
