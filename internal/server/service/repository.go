package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/liebeSonne/gophkeeper/internal/model"
	"github.com/liebeSonne/gophkeeper/internal/repository/db"
)

type UserRepository interface {
	NextID(ctx context.Context) uuid.UUID
	Store(ctx context.Context, user model.User) error
	GetByLogin(ctx context.Context, login string) (model.User, error)
}

type TokenRepository interface {
	NextID(ctx context.Context) uuid.UUID
	Store(ctx context.Context, token model.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (model.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type DataRepository interface {
	NextID(ctx context.Context) uuid.UUID
	Store(ctx context.Context, data model.Data) error
	GetByID(ctx context.Context, id uuid.UUID) (model.Data, error)
	List(ctx context.Context, spec db.ListSpec) ([]model.Data, error)
	Count(ctx context.Context, spec db.ListSpec) (int, error)
	Delete(ctx context.Context, ids []uuid.UUID) error
}
