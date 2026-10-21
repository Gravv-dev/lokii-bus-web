package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create migration tracking table: %w", err)
	}

	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		sql, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(781903402)"); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("lock migration %s: %w", name, err)
		}
		var applied bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", name).Scan(&applied); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("finish migration %s check: %w", name, err)
			}
			continue
		}
		for _, statement := range strings.Split(string(sql), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := tx.Exec(ctx, statement); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("execute migration %s: %w", name, err)
			}
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}
