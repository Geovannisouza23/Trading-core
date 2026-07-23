// Package migrations embeds every versioned SQL migration file so the
// compiled binary can run migrations without needing the source tree on
// disk (used by cmd/migrate and by the API's optional auto-migrate step).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
