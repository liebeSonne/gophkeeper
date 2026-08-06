package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/liebeSonne/gophkeeper/internal/client/model"
	"github.com/liebeSonne/gophkeeper/internal/logger"
)

func setupTestStore(t *testing.T) (store *Store, cleanup func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_store.db")
	l := logger.NewMockLogger(t)
	l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

	var err error
	store, err = NewStore(dbPath, l)
	require.NoError(t, err)

	cleanup = func() {
		_ = store.Close()
		_ = os.Remove(dbPath)
	}

	return store, cleanup
}

func TestStore_Tokens(t *testing.T) {
	t.Run("Save and get tokens", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		now := time.Now()
		tokens := model.Token{
			AccessToken:          "test_access_token",
			RefreshToken:         "test_refresh_token",
			AccessTokenExpiresAt: now.Add(time.Hour),
			RefreshExpiresAt:     now.Add(24 * time.Hour),
		}

		err := store.SaveTokens(tokens)
		require.NoError(t, err)

		got, err := store.GetTokens()
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, tokens.AccessToken, got.AccessToken)
		assert.Equal(t, tokens.RefreshToken, got.RefreshToken)
	})

	t.Run("Get tokens when none exist", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		got, err := store.GetTokens()
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Has valid access token", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		assert.False(t, store.HasValidAccessToken())

		tokens := model.Token{
			AccessToken:          "test_access_token",
			RefreshToken:         "test_refresh_token",
			AccessTokenExpiresAt: time.Now().Add(time.Hour),
			RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
		}

		err := store.SaveTokens(tokens)
		require.NoError(t, err)

		assert.True(t, store.HasValidAccessToken())
	})

	t.Run("Clear tokens", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		tokens := model.Token{
			AccessToken:          "test_access_token",
			RefreshToken:         "test_refresh_token",
			AccessTokenExpiresAt: time.Now().Add(time.Hour),
			RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
		}

		err := store.SaveTokens(tokens)
		require.NoError(t, err)

		err = store.ClearTokens()
		require.NoError(t, err)

		got, err := store.GetTokens()
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("Overwrite tokens", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		tokens1 := model.Token{
			AccessToken:          "first_access_token",
			RefreshToken:         "first_refresh_token",
			AccessTokenExpiresAt: time.Now().Add(time.Hour),
			RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
		}

		err := store.SaveTokens(tokens1)
		require.NoError(t, err)

		tokens2 := model.Token{
			AccessToken:          "second_access_token",
			RefreshToken:         "second_refresh_token",
			AccessTokenExpiresAt: time.Now().Add(2 * time.Hour),
			RefreshExpiresAt:     time.Now().Add(48 * time.Hour),
		}

		err = store.SaveTokens(tokens2)
		require.NoError(t, err)

		got, err := store.GetTokens()
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, tokens2.AccessToken, got.AccessToken)
		assert.Equal(t, tokens2.RefreshToken, got.RefreshToken)
	})
}

func TestStore_DataPut(t *testing.T) {
	t.Run("Put new data entry", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		id := uuid.New()
		entry := model.DataEntry{
			ID:         id,
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"user","password":"pass"}`,
			SyncStatus: model.SyncStatusSynced,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := store.DataPut(entry)
		require.NoError(t, err)

		got, err := store.DataGet(id)
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, id, got.ID)
		assert.Equal(t, model.EntryTypeLoginPassword, got.Type)
		assert.Equal(t, model.SyncStatusSynced, got.SyncStatus)
	})

	t.Run("Update existing data entry", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		id := uuid.New()
		entry := model.DataEntry{
			ID:         id,
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"user","password":"pass"}`,
			SyncStatus: model.SyncStatusSynced,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := store.DataPut(entry)
		require.NoError(t, err)

		entry.Payload = `{"login":"user","password":"updated_pass"}`
		entry.UpdatedAt = time.Now()

		err = store.DataPut(entry)
		require.NoError(t, err)

		got, err := store.DataGet(id)
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, `{"login":"user","password":"updated_pass"}`, got.Payload)
	})

	t.Run("Put entry with remote ID", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		id := uuid.New()
		remoteID := uuid.New()
		etag := "2024-01-01T00:00:00Z"
		entry := model.DataEntry{
			ID:         id,
			RemoteID:   &remoteID,
			Type:       model.EntryTypeText,
			Payload:    `{"text":"secret"}`,
			SyncStatus: model.SyncStatusSynced,
			ServerEtag: &etag,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := store.DataPut(entry)
		require.NoError(t, err)

		got, err := store.DataGet(id)
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, remoteID, *got.RemoteID)
		assert.Equal(t, etag, *got.ServerEtag)
	})
}

func TestStore_DataGet(t *testing.T) {
	t.Run("Get non-existent entry", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		id := uuid.New()
		got, err := store.DataGet(id)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestStore_DataList(t *testing.T) {
	t.Run("List empty", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		entries, err := store.DataList(model.DataFilter{})
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("List with entries", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		for i := 0; i < 3; i++ {
			entry := model.DataEntry{
				ID:         uuid.New(),
				Type:       model.EntryTypeLoginPassword,
				Payload:    `{"login":"user","password":"pass"}`,
				SyncStatus: model.SyncStatusSynced,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			require.NoError(t, store.DataPut(entry))
		}

		entries, err := store.DataList(model.DataFilter{})
		require.NoError(t, err)
		assert.Len(t, entries, 3)
	})

	t.Run("Filter by type", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		for i := 0; i < 2; i++ {
			entry := model.DataEntry{
				ID:         uuid.New(),
				Type:       model.EntryTypeLoginPassword,
				Payload:    `{"login":"user","password":"pass"}`,
				SyncStatus: model.SyncStatusSynced,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			require.NoError(t, store.DataPut(entry))
		}

		for i := 0; i < 2; i++ {
			entry := model.DataEntry{
				ID:         uuid.New(),
				Type:       model.EntryTypeText,
				Payload:    `{"text":"secret"}`,
				SyncStatus: model.SyncStatusSynced,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			require.NoError(t, store.DataPut(entry))
		}

		entries, err := store.DataList(model.DataFilter{Types: []string{"LOGIN_PASSWORD"}})
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("Filter by query", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		entry1 := model.DataEntry{
			ID:         uuid.New(),
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"admin","password":"pass"}`,
			SyncStatus: model.SyncStatusSynced,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(entry1))

		entry2 := model.DataEntry{
			ID:         uuid.New(),
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"user","password":"pass"}`,
			SyncStatus: model.SyncStatusSynced,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(entry2))

		entries, err := store.DataList(model.DataFilter{Query: "admin"})
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})

	t.Run("Limit and offset", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		for i := 0; i < 5; i++ {
			entry := model.DataEntry{
				ID:         uuid.New(),
				Type:       model.EntryTypeLoginPassword,
				Payload:    `{"login":"user","password":"pass"}`,
				SyncStatus: model.SyncStatusSynced,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			require.NoError(t, store.DataPut(entry))
		}

		entries, err := store.DataList(model.DataFilter{Limit: 2, Offset: 1})
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})
}

func TestStore_DataGetPending(t *testing.T) {
	t.Run("Get pending entries", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		syncedEntry := model.DataEntry{
			ID:         uuid.New(),
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"user","password":"pass"}`,
			SyncStatus: model.SyncStatusSynced,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(syncedEntry))

		pendingEntry := model.DataEntry{
			ID:         uuid.New(),
			Type:       model.EntryTypeText,
			Payload:    `{"text":"secret"}`,
			SyncStatus: model.SyncStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(pendingEntry))

		deletingEntry := model.DataEntry{
			ID:         uuid.New(),
			Type:       model.EntryTypeBankCard,
			Payload:    `{"cardNumber":"1234"}`,
			SyncStatus: model.SyncStatusDeleting,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(deletingEntry))

		pending, err := store.DataGetPending()
		require.NoError(t, err)
		assert.Len(t, pending, 2)
	})

	t.Run("No pending entries", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		pending, err := store.DataGetPending()
		require.NoError(t, err)
		assert.Empty(t, pending)
	})
}

func TestStore_DataUpdateSyncStatus(t *testing.T) {
	t.Run("Update sync status", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		id := uuid.New()
		entry := model.DataEntry{
			ID:         id,
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"user","password":"pass"}`,
			SyncStatus: model.SyncStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(entry))

		remoteID := uuid.New()
		etag := "2024-01-01T00:00:00Z"
		err := store.DataUpdateSyncStatus(id, model.SyncStatusSynced, &remoteID, &etag, nil)
		require.NoError(t, err)

		got, err := store.DataGet(id)
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, model.SyncStatusSynced, got.SyncStatus)
		assert.Equal(t, remoteID, *got.RemoteID)
		assert.Equal(t, etag, *got.ServerEtag)
		assert.NotNil(t, got.LastSyncAt)
	})

	t.Run("Update with error message", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		id := uuid.New()
		entry := model.DataEntry{
			ID:         id,
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"user","password":"pass"}`,
			SyncStatus: model.SyncStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(entry))

		errMsg := "connection failed"
		err := store.DataUpdateSyncStatus(id, model.SyncStatusPending, nil, nil, &errMsg)
		require.NoError(t, err)

		got, err := store.DataGet(id)
		require.NoError(t, err)
		require.NotNil(t, got)

		assert.Equal(t, errMsg, *got.ErrorMessage)
	})
}

func TestStore_DataDelete(t *testing.T) {
	t.Run("Delete entry", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		id := uuid.New()
		entry := model.DataEntry{
			ID:         id,
			Type:       model.EntryTypeLoginPassword,
			Payload:    `{"login":"user","password":"pass"}`,
			SyncStatus: model.SyncStatusSynced,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(entry))

		err := store.DataDelete(id)
		require.NoError(t, err)

		got, err := store.DataGet(id)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestStore_DataCount(t *testing.T) {
	t.Run("Count entries", func(t *testing.T) {
		store, cleanup := setupTestStore(t)
		defer cleanup()

		count, err := store.DataCount()
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		for i := 0; i < 3; i++ {
			entry := model.DataEntry{
				ID:         uuid.New(),
				Type:       model.EntryTypeLoginPassword,
				Payload:    `{"login":"user","password":"pass"}`,
				SyncStatus: model.SyncStatusSynced,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			require.NoError(t, store.DataPut(entry))
		}

		count, err = store.DataCount()
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}
