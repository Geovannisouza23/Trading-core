# ADR 0006: Exactly One `database` Folder

## Status

Accepted.

## Context

It is common, over time, for a second "just this one query" data-access
path to appear near whatever code needs it — a repository imported
straight into a handler, or a one-off SQL file living next to a feature
instead of next to the schema it modifies. Each instance makes "how does
this system talk to Postgres" a question with more than one answer, which
is exactly the kind of drift that makes a financial system's audit trail
untrustworthy.

## Decision

Every PostgreSQL concern — the pool (`postgres.go`), health check
(`health.go`), the Fx composition module (`module.go`), migrations
(`migrations/`, embedded via `go:embed`), the migration runner
(`migration/`), the transaction manager (`transaction/`), and one
repository package per aggregate — lives under exactly one directory:
`internal/infrastructure/database`. No `internal/database`, no
`persistence` anywhere, no second folder named `database` anywhere else
under `internal/`.

## Decision enforcement

`tests/architecture/layering_test.go::TestNoForbiddenOrDuplicateDirectoryNames`
walks the entire `internal/` tree and fails if it finds a second directory
literally named `database`, or one named `persistence` or `bootstrap`
anywhere.

## Consequences

- "Where is the SQL for X" has exactly one answer, always.
- Schema migrations, the runner that applies them, and the code that
  queries the resulting tables are impossible to accidentally split across
  two places that drift out of sync.
