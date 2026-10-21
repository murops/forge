package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	rm, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	res, err := rm.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("river migrations: %w", err)
	}
	for _, v := range res.Versions {
		log.Info("applied river migration", "version", v.Version)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys)
	if err != nil {
		return err
	}
	results, err := p.Up(ctx)
	if err != nil {
		return fmt.Errorf("app migrations: %w", err)
	}
	for _, r := range results {
		log.Info("applied migration", "file", r.Source.Path)
	}
	return nil
}
