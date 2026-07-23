# ADR 0005: `internal/interfaces` Is Not `internal/infrastructure/external`

## Status

Accepted.

## Context

Both packages hold "adapters" in the hexagonal-architecture sense, and it
would be easy to collapse them into one `adapters/` folder. But they face
opposite directions: one receives traffic (HTTP requests, WebSocket
upgrades, consumer messages), the other initiates it (a broker call, an
LLM request, a Telegram message). A handler that also holds a database
connection or broker credentials — the natural failure mode of merging
these — undermines the "handlers never touch a repository or a broker
directly" rule that keeps request validation decoupled from business logic
and external I/O.

## Decision

`internal/interfaces` holds only inbound adapters: `http/*`,
`websocket/{hub,connection,handler}`, `consumer/*`. `internal/infrastructure
/external` holds only outbound adapters: brokers, the LLM client, the Quant
Engine client, market data clients, news providers, Telegram, Redis, Cloud
Storage, messaging backends. Neither imports the other.
`tests/architecture/layering_test.go` enforces both directions: interfaces
handlers/consumers cannot import `infrastructure/database` or
`infrastructure/external`, and `infrastructure` cannot import
`interfaces`.

## Consequences

- A handler that needs data always goes through an input port
  (`application/ports/input`), never around it — there is no shortcut path
  to a repository from inside `interfaces/http/handler`.
- The exchange's own WebSocket client
  (`infrastructure/external/marketdata/websocket`) and the dashboard's
  public WebSocket (`interfaces/websocket`) live in clearly different
  places despite both being "a WebSocket", which avoids the confusion of
  one file importing `gorilla/websocket` for two unrelated purposes.
