package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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
			logger.Warn("error syncing logger", "err", syncErr)
		}
	}()

	logger.Info("GophKeeper-server starting",
		"server_address", cfg.ServerAddress,
		"https", cfg.EnableHTTPS,
	)

	connection, err := newConnectionContainer(ctx, cfg, &closer, logger)
	if err != nil {
		logger.Fatal("error creating connection container", "err", err)
	}

	err = runMigrations(connection, logger)
	if err != nil {
		return fmt.Errorf("error running migrations: %w", err)
	}

	deps, err := newDependencyContainer(ctx, connection, cfg, logger)
	if err != nil {
		return fmt.Errorf("error creating dependency container: %w", err)
	}

	srv := &http.Server{
		Addr:              cfg.ServerAddress,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		Handler:           deps.HTTPServerHandler,
	}

	go func() {
		logger.Info("server ready", "address", cfg.ServerAddress)
		var serveErr error
		if cfg.EnableHTTPS {
			serveErr = srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			serveErr = srv.ListenAndServe()
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("server error", "err", serveErr)
		}
	}()

	<-ctx.Done()
	gracefulShutdown(srv, logger)

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

func gracefulShutdown(srv *http.Server, logger internallogger.Logger) {
	logger.Info("starting server shutdown")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	err := srv.Shutdown(shutdownCtx)
	if err != nil {
		logger.Error("server shutdown error", "err", err)
	}

	logger.Info("server shutdown complete")
}
