package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/crypto"
	"github.com/liebeSonne/gophkeeper/internal/jwt"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/server/config"
	"github.com/liebeSonne/gophkeeper/internal/server/handler"
	"github.com/liebeSonne/gophkeeper/internal/server/handler/auth"
	"github.com/liebeSonne/gophkeeper/internal/server/repository/db"
	"github.com/liebeSonne/gophkeeper/internal/server/service"
)

const vaultClientTimeout = 10 * time.Second

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
	dataRepo := db.NewDataRepo(pool, logger)
	fileRepo := db.NewFileRepo(pool, logger)

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
	httpServerHandler := createHTTPServerHandler(serverHandler, authMiddleware)

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
		vaultHTTPClient := &http.Client{
			Timeout: vaultClientTimeout,
		}
		return crypto.NewVaultEncryptor(ctx, cfg.VaultAddress, cfg.VaultToken, cfg.VaultKeyPath, vaultHTTPClient, logger)
	}

	key, err := base64.StdEncoding.DecodeString(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}

	logger.Info("using encryption key from config")
	return crypto.NewAESGCM(key)
}

func createHTTPServerHandler(serverHandler server.ServerInterface, authMiddleware *auth.Middleware) http.Handler {
	r := chi.NewMux()
	r.Use(middleware.AllowContentEncoding("deflate", "gzip"))

	wrapper := &server.ServerInterfaceWrapper{
		Handler: serverHandler,
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			// System
			r.Get("/health", wrapper.HealthCheck)
			// Auth
			r.Post("/auth/login", wrapper.LoginUser)
			r.Post("/auth/refresh", wrapper.RefreshToken)
			r.Post("/auth/register", wrapper.RegisterUser)
		})
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.ToMiddlewareFunc())
			// Data
			r.Post("/data", wrapper.CreateData)
			r.Get("/data/list", wrapper.ListData)
			r.Get("/data/{id}", wrapper.GetData)
			r.Put("/data/{id}", wrapper.UpdateData)
			r.Delete("/data/{id}", wrapper.DeleteData)
			// File
			r.Get("/file/list", wrapper.ListFiles)
			r.Post("/file/upload/init", wrapper.InitUpload)
			r.Post("/file/upload/chunk", wrapper.UploadChunk)
			r.Post("/file/upload/complete", wrapper.CompleteUpload)
			r.Get("/file/{id}/download", wrapper.DownloadFile)
			r.Delete("/file/{id}", wrapper.DeleteFile)
		})
	})

	return r
}
