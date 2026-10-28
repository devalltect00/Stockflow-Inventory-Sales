package database

import (
	"context"
	"fmt"

	"stockflow/backend/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Open verifies PostgreSQL is reachable before the HTTP server accepts requests.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// Migrate applies each embedded SQL migration once, using Goose's version ledger and session lock.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("configure migration lock: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.Files, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("configure migrations: %w", err)
	}
	_, err = provider.Up(ctx)
	return err
}
