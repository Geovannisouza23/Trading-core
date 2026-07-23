# ADR 0003: pgx Without an ORM

## Status

Accepted.

## Context

The persistence layer needs `SELECT ... FOR UPDATE SKIP LOCKED` (the
outbox), `WHERE version = $n` optimistic-locking updates, and array/JSONB
columns — all things ORMs typically fight rather than help with, often by
generating suboptimal SQL or requiring escape hatches that end up looking
like raw SQL anyway. Financial values must never round-trip through a
type an ORM's reflection-based mapper might silently coerce to `float64`.

## Decision

Use `github.com/jackc/pgx/v5` directly. Every repository follows the same
shape: `model.go` (a plain struct matching the row), `queries.go` (SQL
string constants), `mapper.go` (`model ⇄ domain aggregate`, including
`decimal.Decimal ⇄ shared.*` value objects), `repository.go` (the
`output.XRepository` implementation). `shopspring/decimal` implements
`database/sql.Scanner`/`driver.Valuer`, which `pgx`'s type map picks up
automatically for `NUMERIC` columns — no manual string parsing needed.

## Consequences

- Every query is visible, reviewable SQL — no generated-query surprises.
- The four-file-per-aggregate pattern is repetitive by design: once you've
  read `internal/infrastructure/database/account`, every other aggregate
  package looks the same, which is the point.
- We give up ORM conveniences like automatic migration-from-struct-tags;
  migrations are hand-written SQL (see ADR 0004's sibling: the migration
  runner in `internal/infrastructure/database/migration`).
