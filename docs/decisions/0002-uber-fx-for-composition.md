# ADR 0002: Uber Fx for Composition, Nowhere Else

## Status

Accepted.

## Context

With ~11 repositories, 9 use cases, 5 mode-selected external adapters, and
lifecycle-managed resources (a pgx pool, an HTTP server, three background
loops), manual wiring in `main.go` would be a very long, error-prone
function, and every new dependency would mean touching that function by
hand. We also do not want dependency-injection concerns leaking into
`domain` or `application`, where they would add an import neither layer
needs to do its job.

## Decision

Use `go.uber.org/fx` for composition, confined to `cmd/*/main.go`,
`internal/app/*.go`, and exactly one explicitly allow-listed composition
file per adapter category where Fx-managed lifecycle genuinely belongs next
to construction (`internal/infrastructure/database/module.go`, which owns
the pgx pool's connect/close hooks). `tests/architecture` enforces the
allow-list; anything provided outside it fails.

## Consequences

- Adding a new repository or use case is one `fx.Provide` line in
  `internal/app/modules.go`, not a hand-edited constructor chain.
- Wiring mistakes (a missing provider, a type mismatch) surface as a clear
  Fx graph error at process start, not a runtime nil-pointer three calls
  deep — confirmed directly: running the built `api` binary without a
  reachable database resolves the entire ~50-provider graph and fails
  exactly where expected (the pool's `Ping`), which is strong evidence the
  wiring itself is correct.
- `domain` and `application` never see `go.uber.org/fx` in their import
  graphs, so they remain framework-agnostic and fast to test.
