package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/liebeSonne/gophkeeper/internal/config"
	internalio "github.com/liebeSonne/gophkeeper/internal/io/closer"
	internallogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	err := runApp()
	if err != nil {
		log.Fatal(fmt.Errorf("error running app: %w", err))
	}
}

func runApp() error {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	closer := internalio.MultiCloser{}
	defer func() {
		closeErr := closer.Close()
		if closeErr != nil {
			fmt.Println(fmt.Errorf("error closing closer: %w", closeErr))
		}
	}()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	logger, err := initLogger(cfg)
	if err != nil {
		return fmt.Errorf("error initializing logger: %w", err)
	}
	defer func() {
		syncErr := logger.Sync()
		if syncErr != nil {
			fmt.Println(fmt.Errorf("error syncing logger: %w", syncErr))
			logger.Warn("error syncing logger: %v", syncErr)
		}
	}()

	logger.Info("GophKeeper-server starting",
		"server_address", cfg.ServerAddress,
		"https", cfg.EnableHTTPS,
	)

	connection, err := newConnectionContainer(ctx, cfg, &closer)
	if err != nil {
		logger.Fatal("error creating connection container", "err", err)
	}

	err = runMigrations(connection, logger)
	if err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}

	logger.Info("server ready", "address", cfg.ServerAddress)

	_ = newDependencyContainer(connection)

	<-ctx.Done()
	gracefulShutdown(logger)

	return nil
}

func runMigrations(connection *connectionContainer, logger internallogger.Logger) error {
	logger.Info("running migrations...")
	err := connection.Database.Migrate()
	if err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}
	logger.Info("migrations completed")
	return nil
}

func gracefulShutdown(logger internallogger.Logger) {
	logger.Info("starting server shutdown")

	logger.Info("server shutdown complete")
}
