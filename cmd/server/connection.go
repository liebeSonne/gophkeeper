package main

import (
	"context"
	"fmt"

	"github.com/liebeSonne/gophkeeper/internal/storage"

	iocloser "github.com/liebeSonne/gophkeeper/internal/io/closer"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/server/config"
	"github.com/liebeSonne/gophkeeper/internal/server/repository/db"
)

type connectionContainer struct {
	Database    *db.DB
	MinIOClient storage.MinIOClient
}

func newConnectionContainer(
	ctx context.Context,
	cfg config.ServerConfig,
	closer *iocloser.MultiCloser,
	logger intlogger.Logger,
) (*connectionContainer, error) {
	database, err := db.New(ctx, cfg.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	if closer != nil {
		closer.AddCloser(iocloser.Func(
			func() error {
				return database.Close()
			},
		))
	}

	minioClient, err := createMinIOClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	if minioClient != nil {
		err = minioClient.EnsureBucket(ctx, cfg.StorageBucket)
		if err != nil {
			return nil, fmt.Errorf("ensure minio bucket: %w", err)
		}
	}

	return &connectionContainer{
		Database:    database,
		MinIOClient: minioClient,
	}, nil
}

func createMinIOClient(cfg config.ServerConfig, logger intlogger.Logger) (storage.MinIOClient, error) {
	if cfg.StorageEndpoint == "" {
		return nil, nil
	}

	return storage.NewMinIOClient(cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey, cfg.StorageSecure, logger)
}
