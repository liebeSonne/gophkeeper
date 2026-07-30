// Package db provides database connection and migration management.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // for migration fs
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // for driver
	_ "github.com/lib/pq"              // for driver

	"github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/migrations"
)

var migrationsFS = migrations.FS

type DB struct {
	conn    *sql.DB
	migrate *migrate.Migrate
}

func New(ctx context.Context, databaseURI string, l logger.Logger) (*DB, error) {
	pool, err := sql.Open("pgx", databaseURI)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	err = pool.PingContext(ctx)
	if err != nil {
		errClose := pool.Close()
		if errClose != nil {
			l.Warn("db pool close on ping error", "err", errClose)
		}
		return nil, fmt.Errorf("ping database: %w", err)
	}

	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(time.Hour)

	m, err := setupMigrations(pool)
	if err != nil {
		errClose := pool.Close()
		if errClose != nil {
			l.Warn("db pool close on setup migrations error", "err", errClose)
		}
		return nil, fmt.Errorf("setup migrations: %w", err)
	}

	return &DB{
		conn:    pool,
		migrate: m,
	}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Ping() error {
	return d.conn.Ping()
}

func (d *DB) Migrate() error {
	err := d.migrate.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func setupMigrations(pool *sql.DB) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("migrations fs: %w", err)
	}

	driver, err := postgres.WithInstance(pool, &postgres.Config{})
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
