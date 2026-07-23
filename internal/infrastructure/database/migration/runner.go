// Package migration implements a small, dependency-free SQL migration
// runner reading the embedded files in
// internal/infrastructure/database/migrations.
//
// PostgreSQL DDL is transactional, so unlike migration tools built to
// support databases without transactional DDL, a single transaction per
// step is enough to guarantee there is never a partially-applied
// migration: either the whole step (schema_migrations bookkeeping + SQL)
// commits, or none of it does.
package migration

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var fileNamePattern = regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

type step struct {
	version int64
	name    string
	upSQL   string
	downSQL string
}

// Runner applies or reverts embedded SQL migrations against a Postgres
// database, tracking progress in a schema_migrations table.
type Runner struct {
	pool  *pgxpool.Pool
	steps []step
}

// NewRunner parses every *.sql file in fsys into ordered migration steps.
func NewRunner(pool *pgxpool.Pool, fsys embed.FS) (*Runner, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("reading migrations directory: %w", err)
	}

	byVersion := map[int64]*step{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := fileNamePattern.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		version, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing migration version from %s: %w", entry.Name(), err)
		}
		content, err := fsys.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading migration file %s: %w", entry.Name(), err)
		}
		s, ok := byVersion[version]
		if !ok {
			s = &step{version: version, name: match[2]}
			byVersion[version] = s
		}
		if match[3] == "up" {
			s.upSQL = string(content)
		} else {
			s.downSQL = string(content)
		}
	}

	steps := make([]step, 0, len(byVersion))
	for _, s := range byVersion {
		steps = append(steps, *s)
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].version < steps[j].version })

	return &Runner{pool: pool, steps: steps}, nil
}

func (r *Runner) ensureVersionTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}

// Version reports the highest applied migration version, or 0 if no
// migration has run yet.
func (r *Runner) Version(ctx context.Context) (int64, error) {
	if err := r.ensureVersionTable(ctx); err != nil {
		return 0, err
	}
	var version int64
	row := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`)
	if err := row.Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

// Force replaces the entire tracked history with a single row at the given
// version, without running any SQL. Used to recover manually after an
// out-of-band schema fix.
func (r *Runner) Force(ctx context.Context, version int64) error {
	if err := r.ensureVersionTable(ctx); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations`); err != nil {
		return err
	}
	if version > 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Up applies every migration with a version greater than the current one,
// in order, each inside its own transaction.
func (r *Runner) Up(ctx context.Context) error {
	current, err := r.Version(ctx)
	if err != nil {
		return err
	}
	for _, s := range r.steps {
		if s.version <= current {
			continue
		}
		if s.upSQL == "" {
			return fmt.Errorf("migration %d_%s has no up script", s.version, s.name)
		}
		if err := r.runInTx(ctx, s.upSQL, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, s.version)
			return err
		}); err != nil {
			return fmt.Errorf("applying migration %d_%s: %w", s.version, s.name, err)
		}
	}
	return nil
}

// Down reverts the `count` most recently applied migrations, most recent
// first.
func (r *Runner) Down(ctx context.Context, count int) error {
	current, err := r.Version(ctx)
	if err != nil {
		return err
	}
	reverted := 0
	for i := len(r.steps) - 1; i >= 0 && reverted < count && current > 0; i-- {
		s := r.steps[i]
		if s.version != current {
			continue
		}
		if s.downSQL == "" {
			return fmt.Errorf("migration %d_%s has no down script", s.version, s.name)
		}
		if err := r.runInTx(ctx, s.downSQL, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, s.version)
			return err
		}); err != nil {
			return fmt.Errorf("reverting migration %d_%s: %w", s.version, s.name, err)
		}
		reverted++
		current, err = r.previousVersion(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) previousVersion(currentIndex int) (int64, error) {
	if currentIndex <= 0 {
		return 0, nil
	}
	return r.steps[currentIndex-1].version, nil
}

func (r *Runner) runInTx(ctx context.Context, migrationSQL string, bookkeeping func(tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, migrationSQL); err != nil {
		return err
	}
	if err := bookkeeping(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
