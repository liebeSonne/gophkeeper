package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/liebeSonne/gophkeeper/internal/model"
	"github.com/liebeSonne/gophkeeper/internal/repository"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) NextID(_ context.Context) uuid.UUID {
	return uuid.New()
}

func (r *UserRepo) Store(ctx context.Context, user *model.User) error {
	const query = `
		INSERT INTO users (id, login, password, created_at) 
		VALUES ($1, $2, $3, $4) 
		ON CONFLICT (id) DO UPDATE SET 
			login = EXCLUDED.login, 
			password = EXCLUDED.password
	`
	_, err := r.pool.Exec(ctx, query, user.ID, user.Login, user.Password, user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.UniqueViolation == pgErr.Code {
			return repository.NewErrConflictUserLogin(user.Login, err)
		}
		return fmt.Errorf("store user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	const query = `
		SELECT id, login, password, created_at 
		FROM users 
		WHERE id = $1
	`
	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Login, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	const query = `
		SELECT id, login, password, created_at 
		FROM users 
		WHERE login = $1
	`
	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, login).Scan(&user.ID, &user.Login, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get user by login: %w", err)
	}
	return user, nil
}

func (r *UserRepo) Delete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	const query = `DELETE FROM users WHERE id = ANY($1)`
	_, err := r.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("delete users: %w", err)
	}
	return nil
}
