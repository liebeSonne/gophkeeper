package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	// Register sqlite3 driver for database/sql.
	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/liebeSonne/gophkeeper/internal/client/model"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

type TokenStorage struct {
	db *sql.DB
}

func NewTokenStorage(storagePath string, logger intlogger.Logger) (*TokenStorage, error) {
	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	err = db.Ping()
	if err != nil {
		errClose := db.Close()
		if errClose != nil {
			logger.Warn("error on close db", "err", errClose)
		}
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	ts := &TokenStorage{db: db}

	err = ts.initTables()
	if err != nil {
		errClose := db.Close()
		if errClose != nil {
			logger.Warn("error on close db", "err", errClose)
		}
		return nil, err
	}

	return ts, nil
}

func (s *TokenStorage) initTables() error {
	createTable := `
		CREATE TABLE IF NOT EXISTS tokens (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			access_token_expires_at TEXT NOT NULL,
			refresh_token_expires_at TEXT NOT NULL
		);`

	_, err := s.db.Exec(createTable)
	return err
}

func (s *TokenStorage) SaveTokens(tokens model.Token) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(`
		INSERT INTO tokens (id, access_token, refresh_token, access_token_expires_at, refresh_token_expires_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			access_token_expires_at = excluded.access_token_expires_at,
			refresh_token_expires_at = excluded.refresh_token_expires_at
	`,
		tokens.AccessToken,
		tokens.RefreshToken,
		tokens.AccessTokenExpiresAt.Format(time.RFC3339),
		tokens.RefreshExpiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert tokens: %w", err)
	}

	return tx.Commit()
}

func (s *TokenStorage) GetTokens() (*model.Token, error) {
	row := s.db.QueryRow(`
		SELECT access_token, refresh_token, access_token_expires_at, refresh_token_expires_at
		FROM tokens
		WHERE id = 1
	`)

	var token model.Token
	var accessTokenExpiry, refreshExpiry string

	err := row.Scan(&token.AccessToken, &token.RefreshToken, &accessTokenExpiry, &refreshExpiry)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan tokens: %w", err)
	}

	token.AccessTokenExpiresAt, err = time.Parse(time.RFC3339, accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("parse access token expiry: %w", err)
	}

	token.RefreshExpiresAt, err = time.Parse(time.RFC3339, refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token expiry: %w", err)
	}

	return &token, nil
}

func (s *TokenStorage) HasValidAccessToken() bool {
	tokens, err := s.GetTokens()
	if err != nil || tokens == nil {
		return false
	}

	return time.Now().Before(tokens.AccessTokenExpiresAt)
}

func (s *TokenStorage) ClearTokens() error {
	_, err := s.db.Exec("DELETE FROM tokens WHERE id = 1")
	return err
}

func (s *TokenStorage) Close() error {
	return s.db.Close()
}
