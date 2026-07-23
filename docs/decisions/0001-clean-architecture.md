# ADR 0001: Clean Architecture with Ports and Adapters

## Status

Accepted.

## Context

This service ultimately decides whether to send an order to a real
exchange. Every dependency the decision logic (risk, sizing, state
transitions) has on a database, a broker SDK, or a web framework is a
dependency that makes that logic harder to test in isolation and easier to
get subtly wrong under a library upgrade. The system also has three
concrete adapters per external concern that must be swappable at runtime
(Broker: paper/testnet/real; LLM: noop/gemini; Quant: fake/grpc).

## Decision

Structure the codebase as `domain → application → {interfaces,
infrastructure}`, with the direction enforced two ways: code review
convention, and `tests/architecture/layering_test.go`, which fails the
build if `domain` or `application` ever import a framework or a concrete
adapter. Output ports (`application/ports/output`) are the only contract
infrastructure implements; input ports (`application/ports/input`) are the
only contract interfaces call.

## Consequences

- `domain` and `application` compile and test with zero external
  dependencies beyond `decimal`/`uuid` and the standard library — fast
  feedback, no Docker required for the majority of the test suite.
- Swapping an adapter (PaperBroker → Testnet, in-memory EventBus → Pub/Sub)
  is a one-function change in `internal/app/providers.go`.
- The cost is more files and more explicit wiring than a framework-coupled
  "fat handler" style would need. For a system whose mistakes cost real
  money, that verbosity buys testability we consider worth it.
