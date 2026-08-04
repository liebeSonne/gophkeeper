package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/config"
	"github.com/liebeSonne/gophkeeper/internal/crypto"
	"github.com/liebeSonne/gophkeeper/internal/jwt"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/repository/db"
	"github.com/liebeSonne/gophkeeper/internal/server/auth"
	"github.com/liebeSonne/gophkeeper/internal/server/handler"
	"github.com/liebeSonne/gophkeeper/internal/server/service"
)

type dependencyContainer struct {
	HTTPServerHandler http.Handler
}

func newDependencyContainer(
	ctx context.Context,
	con *connectionContainer,
	cfg config.ServerConfig,
	logger intlogger.Logger,
) (*dependencyContainer, error) {
	pool := con.Database.Pool()

	userRepo := db.NewUserRepo(pool)
	tokenRepo := db.NewTokenRepo(pool)
	dataRepo := db.NewDataRepo(pool)
	fileRepo := db.NewFileRepo(pool)

	jwtService := jwt.NewJWT(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtService)

	encryptor, err := createEncryptor(ctx, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("create encryptor: %w", err)
	}

	dataService := service.NewDataService(dataRepo, encryptor, fileRepo)
	fileService := service.NewFileService(fileRepo, encryptor, con.MinIOClient, cfg.StorageBucket, logger)

	// HTTP Server
	authMiddleware := auth.NewAuthMiddleware(authService)
	serverHandler := handler.NewServerHandler(authService, dataService, fileService, logger)

	r := chi.NewMux()
	r.Use(middleware.AllowContentEncoding("deflate", "gzip"))
	r.Use(authMiddleware.ToMiddlewareFunc())
	httpServerHandler := server.HandlerFromMux(serverHandler, r)

	return &dependencyContainer{
		HTTPServerHandler: httpServerHandler,
	}, nil
}

func createEncryptor(
	ctx context.Context,
	cfg config.ServerConfig,
	logger intlogger.Logger,
) (crypto.Encryptor, error) {
	if cfg.EnableVault {
		logger.Info("loading encryption key from Vault",
			"address", cfg.VaultAddress,
			"key_path", cfg.VaultKeyPath,
		)
		return crypto.NewVaultEncryptor(ctx, cfg.VaultAddress, cfg.VaultToken, cfg.VaultKeyPath, logger)
	}

	key, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}

	logger.Info("using encryption key from config")
	return crypto.NewAESGCM(key)
}
