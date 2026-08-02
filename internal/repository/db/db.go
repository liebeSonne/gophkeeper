package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/liebeSonne/gophkeeper/migrations"
)

var migrationsFS = migrations.FS

type DB struct {
	pool    *pgxpool.Pool
	migrate *migrate.Migrate
}

func New(ctx context.Context, databaseURI string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinIdleConns = 5
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	m, err := setupMigrations(sqlDB)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("setup migrations: %w", err)
	}

	return &DB{
		pool:    pool,
		migrate: m,
	}, nil
}

func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *DB) Close() error {
	d.pool.Close()
	return nil
}

func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *DB) Migrate() error {
	err := d.migrate.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func setupMigrations(sqlDB *sql.DB) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("migrations fs: %w", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return m, nil
}
