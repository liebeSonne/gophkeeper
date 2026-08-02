package main

import (
	"context"
	"fmt"

	"github.com/liebeSonne/gophkeeper/internal/config"
	iocloser "github.com/liebeSonne/gophkeeper/internal/io/closer"
	"github.com/liebeSonne/gophkeeper/internal/repository/db"
)

type connectionContainer struct {
	Database *db.DB
}

func newConnectionContainer(
	ctx context.Context,
	cfg config.ServerConfig,
	closer *iocloser.MultiCloser,
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

	return &connectionContainer{
		Database: database,
	}, nil
}
