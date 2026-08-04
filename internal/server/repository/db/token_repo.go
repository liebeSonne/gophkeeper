package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

type TokenRepo struct {
	pool *pgxpool.Pool
}

func NewTokenRepo(pool *pgxpool.Pool) *TokenRepo {
	return &TokenRepo{pool: pool}
}

func (r *TokenRepo) NextID(_ context.Context) uuid.UUID {
	return uuid.New()
}

func (r *TokenRepo) Store(ctx context.Context, token model.RefreshToken) error {
	const query = `
		INSERT INTO refresh_token (id, user_id, token_hash, expires_at, revoked_at, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		ON CONFLICT (id) DO UPDATE SET 
			user_id = EXCLUDED.user_id, 
			token_hash = EXCLUDED.token_hash, 
			expires_at = EXCLUDED.expires_at, 
			revoked_at = EXCLUDED.revoked_at
	`
	_, err := r.pool.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.RevokedAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (r *TokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (model.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at 
		FROM refresh_token 
		WHERE token_hash = $1
	`
	token := model.RefreshToken{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RefreshToken{}, repository.ErrNotFound
		}
		return model.RefreshToken{}, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return token, nil
}

func (r *TokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE refresh_token SET revoked_at = $1 
		 WHERE id = $2 AND revoked_at IS NULL
	`
	result, err := r.pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *TokenRepo) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE refresh_token SET revoked_at = $1 
		WHERE user_id = $2 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("revoke refresh tokens by user id: %w", err)
	}
	return nil
}

func (r *TokenRepo) Delete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	const query = `DELETE FROM refresh_token WHERE id = ANY($1)`
	_, err := r.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("delete refresh tokens: %w", err)
	}
	return nil
}

func (r *TokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	const query = `DELETE FROM refresh_token WHERE expires_at < NOW() AND revoked_at IS NULL`
	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired refresh tokens: %w", err)
	}
	return result.RowsAffected(), nil
}
