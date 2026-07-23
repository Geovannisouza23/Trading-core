# ADR 0008: gRPC Contract for the Future Rust Quant Engine, Uncompiled

## Status

Accepted.

## Context

The quantitative model is explicitly out of scope for this delivery and
will eventually be a separate Rust service. Go needs a stable contract to
code against today, and the future Rust side needs the same contract to
implement against later, without either side blocking on the other's
existence. This build environment does not have `protoc` installed, so
generating and committing Go stubs from the `.proto` file is not possible
right now — and would create a codegen artifact nobody can regenerate
without extra tooling anyway.

## Decision

Write the real contract at
`internal/contracts/grpc/quant/quant_engine.proto` (the `EvaluateSignal`
and `AnalyzeMarketRegime` RPCs, decimal-as-string fields, never a float).
Leave `internal/contracts/grpc/quant/generated/` with a `doc.go` explaining
exactly how to run `make proto` once `protoc` +
`protoc-gen-go` + `protoc-gen-go-grpc` are available. The Go application
depends on `application/ports/output.QuantEngine` — a plain Go interface,
not a protobuf-generated one — currently satisfied by an in-process
deterministic fake
(`internal/infrastructure/external/quant/grpc`, same package name the
future gRPC client will occupy).

## Consequences

- The build has zero dependency on `protoc` or generated protobuf code.
- When the Rust engine exists, connecting it is: run `make proto`, write a
  gRPC client in the same package implementing the same
  `output.QuantEngine` interface the fake does today, and flip
  `provideQuantEngine`'s `cfg.Quant.Mode == "grpc"` branch in
  `internal/app/providers.go` — no use case, risk logic, or execution code
  changes.
- The `.proto` file is a real, reviewable contract today even though
  nothing generates code from it yet — the Rust team can start
  implementing against it immediately.
