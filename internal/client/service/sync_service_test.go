package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/liebeSonne/gophkeeper/internal/client/model"
	"github.com/liebeSonne/gophkeeper/internal/client/storage"
	"github.com/liebeSonne/gophkeeper/internal/logger"
)

func setupTestStore(t *testing.T) (store *storage.Store, cleanup func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_store.db")
	l := logger.NewMockLogger(t)
	l.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	l.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	var err error
	store, err = storage.NewStore(dbPath, l)
	require.NoError(t, err)

	cleanup = func() {
		_ = store.Close()
		_ = os.Remove(dbPath)
	}

	return store, cleanup
}

func newTestLogger(t *testing.T) *logger.MockLogger {
	t.Helper()
	l := logger.NewMockLogger(t)
	l.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	l.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	return l
}

func TestService_StartStop(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	l := newTestLogger(t)
	svc := NewService(store, nil, l, 100*time.Millisecond)

	svc.Start()
	svc.Stop()
}

func TestService_Sync_NoTokens(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	l := newTestLogger(t)
	svc := NewService(store, nil, l, time.Hour)

	ctx := context.Background()
	err := svc.Sync(ctx)
	assert.NoError(t, err)
}

func TestService_SyncPush_EmptyPending_NoClient(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	l := newTestLogger(t)
	svc := NewService(store, nil, l, time.Hour)

	ctx := context.Background()
	err := svc.syncPush(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not configured")
}

func TestService_SyncPull_NoClient(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	l := newTestLogger(t)
	svc := NewService(store, nil, l, time.Hour)

	ctx := context.Background()
	err := svc.syncPull(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not configured")
}

func TestService_BackgroundSync(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	l := newTestLogger(t)
	svc := NewService(store, nil, l, 50*time.Millisecond)

	svc.Start()
	time.Sleep(150 * time.Millisecond)
	svc.Stop()
}

func TestService_SyncPush_DeletingEntry_NoRemoteID_NoClient(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	entry := model.DataEntry{
		ID:         uuid.New(),
		Type:       model.EntryTypeLoginPassword,
		Payload:    `{"login":"user","password":"pass"}`,
		SyncStatus: model.SyncStatusDeleting,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, store.DataPut(entry))

	l := newTestLogger(t)
	svc := NewService(store, nil, l, time.Hour)

	ctx := context.Background()
	err := svc.syncPush(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not configured")
}
