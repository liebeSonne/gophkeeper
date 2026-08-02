package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/config"
	jwt2 "github.com/liebeSonne/gophkeeper/internal/jwt"
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
	con *connectionContainer,
	cfg config.ServerConfig,
	logger intlogger.Logger,
) *dependencyContainer {
	pool := con.Database.Pool()

	userRepo := db.NewUserRepo(pool)
	tokenRepo := db.NewTokenRepo(pool)
	_ = db.NewDataRepo(pool)

	jwt := jwt2.NewJWT(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authService := service.NewAuthService(userRepo, tokenRepo, jwt)

	// HTTP Server
	authMiddleware := auth.NewAuthMiddleware(authService)
	serverHandler := handler.NewServerHandler(authService, logger)

	r := chi.NewMux()
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: logger}))
	r.Use(middleware.AllowContentEncoding("deflate", "gzip"))
	r.Use(authMiddleware.ToMiddlewareFunc())
	httpServerHandler := server.HandlerFromMux(serverHandler, r)

	return &dependencyContainer{
		HTTPServerHandler: httpServerHandler,
	}
}
