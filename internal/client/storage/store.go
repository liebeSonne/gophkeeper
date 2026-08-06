package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/ncruces/go-sqlite3/driver" // register sqlite3 driver for database/sql

	"github.com/liebeSonne/gophkeeper/internal/client/model"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string, logger intlogger.Logger) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
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

	s := &Store{db: db}

	if err := s.initTokensTable(); err != nil {
		errClose := db.Close()
		if errClose != nil {
			logger.Warn("error on close db", "err", errClose)
		}
		return nil, fmt.Errorf("init tokens table: %w", err)
	}

	if err := s.initDataEntriesTable(); err != nil {
		errClose := db.Close()
		if errClose != nil {
			logger.Warn("error on close db", "err", errClose)
		}
		return nil, fmt.Errorf("init data entries table: %w", err)
	}

	return s, nil
}

func (s *Store) initTokensTable() error {
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

func (s *Store) initDataEntriesTable() error {
	createTable := `
		CREATE TABLE IF NOT EXISTS data_entries (
			id              TEXT PRIMARY KEY,
			remote_id       TEXT,
			type            TEXT NOT NULL,
			payload         TEXT NOT NULL,
			sync_status     TEXT NOT NULL DEFAULT 'synced',
			server_etag     TEXT,
			last_sync_at    TEXT,
			error_message   TEXT,
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL
		);`

	_, err := s.db.Exec(createTable)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_data_entries_sync_status ON data_entries(sync_status)")
	if err != nil {
		return err
	}

	_, err = s.db.Exec("CREATE INDEX IF NOT EXISTS idx_data_entries_remote_id ON data_entries(remote_id)")
	if err != nil {
		return err
	}

	return nil
}

// Token methods

func (s *Store) SaveTokens(tokens model.Token) error {
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

func (s *Store) GetTokens() (*model.Token, error) {
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

func (s *Store) HasValidAccessToken() bool {
	tokens, err := s.GetTokens()
	if err != nil || tokens == nil {
		return false
	}

	return time.Now().Before(tokens.AccessTokenExpiresAt)
}

func (s *Store) ClearTokens() error {
	_, err := s.db.Exec("DELETE FROM tokens WHERE id = 1")
	return err
}

// Data methods

func (s *Store) DataPut(entry model.DataEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	remoteID := nullUUID(entry.RemoteID)
	serverEtag := nullStr(entry.ServerEtag)
	lastSyncAt := nullTime(entry.LastSyncAt)
	errMsg := nullStr(entry.ErrorMessage)

	_, err = tx.Exec(`
		INSERT INTO data_entries (id, remote_id, type, payload, sync_status, server_etag, last_sync_at, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			remote_id = excluded.remote_id,
			type = excluded.type,
			payload = excluded.payload,
			sync_status = excluded.sync_status,
			server_etag = excluded.server_etag,
			last_sync_at = excluded.last_sync_at,
			error_message = excluded.error_message,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at
	`,
		entry.ID.String(),
		remoteID,
		entry.Type,
		entry.Payload,
		string(entry.SyncStatus),
		serverEtag,
		lastSyncAt,
		errMsg,
		entry.CreatedAt.Format(time.RFC3339),
		entry.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert data entry: %w", err)
	}

	return tx.Commit()
}

func (s *Store) DataGet(id uuid.UUID) (*model.DataEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, remote_id, type, payload, sync_status, server_etag, last_sync_at, error_message, created_at, updated_at
		FROM data_entries
		WHERE id = ?
	`, id.String())

	return scanDataEntry(row)
}

func (s *Store) DataList(filter model.DataFilter) ([]model.DataEntry, error) {
	query := "SELECT id, remote_id, type, payload, sync_status, server_etag, last_sync_at, error_message, created_at, updated_at FROM data_entries WHERE 1=1"
	args := []interface{}{}

	if len(filter.Types) > 0 {
		placeholder := ""
		for i := range filter.Types {
			if i > 0 {
				placeholder += ", "
			}
			placeholder += "?"
			args = append(args, filter.Types[i])
		}
		query += fmt.Sprintf(" AND type IN (%s)", placeholder) //nolint:gosec // placeholders are safe, values are parameterized
	}

	if filter.Query != "" {
		query += " AND (payload LIKE ? OR type LIKE ?)"
		likeQuery := "%" + filter.Query + "%"
		args = append(args, likeQuery, likeQuery)
	}

	query += " ORDER BY updated_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query data entries: %w", err)
	}
	defer rows.Close()

	var entries []model.DataEntry
	for rows.Next() {
		entry, errScan := scanDataEntryRow(rows)
		if errScan != nil {
			return nil, errScan
		}
		entries = append(entries, *entry)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate data entries: %w", err)
	}

	if entries == nil {
		return []model.DataEntry{}, nil
	}

	return entries, nil
}

func (s *Store) DataGetPending() ([]model.DataEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, remote_id, type, payload, sync_status, server_etag, last_sync_at, error_message, created_at, updated_at
		FROM data_entries
		WHERE sync_status IN ('pending', 'deleting')
		ORDER BY updated_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query pending entries: %w", err)
	}
	defer rows.Close()

	var entries []model.DataEntry
	for rows.Next() {
		entry, errScan := scanDataEntryRow(rows)
		if errScan != nil {
			return nil, errScan
		}
		entries = append(entries, *entry)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate pending entries: %w", err)
	}

	if entries == nil {
		return []model.DataEntry{}, nil
	}

	return entries, nil
}

func (s *Store) DataUpdateSyncStatus(id uuid.UUID, status model.SyncStatus, remoteID *uuid.UUID, etag, errMsg *string) error {
	now := time.Now().Format(time.RFC3339)
	remoteIDStr := nullUUID(remoteID)
	serverEtag := nullStr(etag)
	errMsgStr := nullStr(errMsg)

	_, err := s.db.Exec(`
		UPDATE data_entries
		SET sync_status = ?, remote_id = ?, server_etag = ?, last_sync_at = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`, string(status), remoteIDStr, serverEtag, now, errMsgStr, now, id.String())

	if err != nil {
		return fmt.Errorf("update sync status: %w", err)
	}

	return nil
}

func (s *Store) DataDelete(id uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM data_entries WHERE id = ?", id.String())
	return err
}

func (s *Store) DataCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM data_entries").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count data entries: %w", err)
	}
	return count, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func scanDataEntry(row *sql.Row) (*model.DataEntry, error) {
	entry, err := scanDataEntryRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return entry, nil
}

func scanDataEntryRow(row scanner) (*model.DataEntry, error) {
	var id string
	var remoteID, serverEtag, lastSyncAt, errMsg sql.NullString
	var entryType, payload, syncStatus, createdAt, updatedAt string

	err := row.Scan(&id, &remoteID, &entryType, &payload, &syncStatus, &serverEtag, &lastSyncAt, &errMsg, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	entry := &model.DataEntry{
		Type:       entryType,
		Payload:    payload,
		SyncStatus: model.SyncStatus(syncStatus),
	}

	if parsedID, err := uuid.Parse(id); err == nil {
		entry.ID = parsedID
	}

	if remoteID.Valid {
		if parsedID, err := uuid.Parse(remoteID.String); err == nil {
			entry.RemoteID = &parsedID
		}
	}

	if serverEtag.Valid {
		entry.ServerEtag = &serverEtag.String
	}

	if lastSyncAt.Valid {
		if t, err := time.Parse(time.RFC3339, lastSyncAt.String); err == nil {
			entry.LastSyncAt = &t
		}
	}

	if errMsg.Valid {
		entry.ErrorMessage = &errMsg.String
	}

	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		entry.CreatedAt = t
	}

	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		entry.UpdatedAt = t
	}

	return entry, nil
}

func nullStr(s *string) *string {
	return s
}

func nullTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func nullUUID(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

type scanner interface {
	Scan(dest ...interface{}) error
}
