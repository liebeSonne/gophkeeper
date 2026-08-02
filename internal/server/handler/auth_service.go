package handler

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/liebeSonne/gophkeeper/internal/model"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService interface {
	Register(ctx context.Context, login, password string) (model.Token, error)
	Login(ctx context.Context, login, password string) (model.Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (model.Token, error)
	RevokeToken(ctx context.Context, refreshToken string) error
	GetUserIDFromToken(tokenString string) (uuid.UUID, error)
}
