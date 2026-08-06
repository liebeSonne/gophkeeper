package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apiClient "github.com/liebeSonne/gophkeeper/internal/client/adapter/gophkeeper"
	"github.com/liebeSonne/gophkeeper/internal/client/model"
	"github.com/liebeSonne/gophkeeper/internal/client/storage"
	"github.com/liebeSonne/gophkeeper/internal/logger"
	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

func setupSyncTestStore(t *testing.T) (store *storage.Store, cleanup func()) {
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

func newSyncTestLogger(t *testing.T) *logger.MockLogger {
	t.Helper()
	l := logger.NewMockLogger(t)
	l.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	l.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	l.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	return l
}

func TestService_SyncPushEntry_CreateNew(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	entryID := uuid.New()
	remoteID := uuid.New()
	entry := model.DataEntry{
		ID:         entryID,
		Type:       model.EntryTypeText,
		Payload:    `{"text":"test data"}`,
		SyncStatus: model.SyncStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, store.DataPut(entry))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"data":{"id":"` + remoteID.String() + `","type":"TEXT","text":"test data","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPush(ctx)
	require.NoError(t, err)

	got, err := store.DataGet(entryID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.SyncStatusSynced, got.SyncStatus)
	assert.NotNil(t, got.RemoteID)
	assert.Equal(t, remoteID, *got.RemoteID)
}

func TestService_SyncPushEntry_UpdateExisting(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	entryID := uuid.New()
	remoteID := uuid.New()
	entry := model.DataEntry{
		ID:         entryID,
		RemoteID:   &remoteID,
		Type:       model.EntryTypeText,
		Payload:    `{"text":"updated data"}`,
		SyncStatus: model.SyncStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, store.DataPut(entry))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"data":{"id":"` + remoteID.String() + `","type":"TEXT","text":"updated data","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPush(ctx)
	require.NoError(t, err)

	got, err := store.DataGet(entryID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.SyncStatusSynced, got.SyncStatus)
}

func TestService_SyncDeleteEntry_WithRemoteID(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	entryID := uuid.New()
	remoteID := uuid.New()
	entry := model.DataEntry{
		ID:         entryID,
		RemoteID:   &remoteID,
		Type:       model.EntryTypeText,
		Payload:    `{"text":"test"}`,
		SyncStatus: model.SyncStatusDeleting,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, store.DataPut(entry))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPush(ctx)
	require.NoError(t, err)

	got, err := store.DataGet(entryID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_SyncDeleteEntry_NoRemoteID(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	entryID := uuid.New()
	entry := model.DataEntry{
		ID:         entryID,
		Type:       model.EntryTypeText,
		Payload:    `{"text":"test"}`,
		SyncStatus: model.SyncStatusDeleting,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, store.DataPut(entry))

	l := newSyncTestLogger(t)
	svc := NewService(store, nil, l, time.Hour)

	ctx := context.Background()
	err := svc.syncPush(ctx)
	assert.Error(t, err)
}

func TestService_SyncPull_EmptyStore(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"items":[{"id":"550e8400-e29b-41d4-a716-446655440000","type":"TEXT","text":"test","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"page":1,"page_size":100,"total":1,"total_pages":1}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPull(ctx)
	require.NoError(t, err)

	count, err := store.DataCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestService_SyncPull_NonEmptyStore(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	entry := model.DataEntry{
		ID:         uuid.New(),
		Type:       model.EntryTypeText,
		Payload:    `{"text":"existing"}`,
		SyncStatus: model.SyncStatusSynced,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, store.DataPut(entry))

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("should not be called")
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPull(ctx)
	require.NoError(t, err)

	count, err := store.DataCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestService_SyncPull_ContextCancelled(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = svc.syncPull(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestBuildEntryFromData_LoginPassword(t *testing.T) {
	remoteID := uuid.New()
	now := time.Now()
	data := gophkeeper.LoginPasswordDataInfo{
		Id:        remoteID,
		Type:      gophkeeper.LoginPasswordDataInfoTypeLOGINPASSWORD,
		Login:     "testuser",
		Password:  "testpass",
		CreatedAt: now,
		UpdatedAt: now,
	}

	item := gophkeeper.DataInfo{}
	require.NoError(t, item.FromLoginPasswordDataInfo(data))

	entry, err := buildEntryFromData(&item)
	require.NoError(t, err)
	assert.Equal(t, model.EntryTypeLoginPassword, entry.Type)
	assert.Equal(t, remoteID, *entry.RemoteID)
	assert.Contains(t, entry.Payload, "testuser")
	assert.Contains(t, entry.Payload, "testpass")
}

func TestBuildEntryFromData_BankCard(t *testing.T) {
	data := gophkeeper.BankCardDataInfo{
		Id:         uuid.New(),
		Type:       gophkeeper.BankCardDataInfoTypeBANKCARD,
		CardNumber: "1234567890123456",
		CardHolder: "John Doe",
		CardExpiry: "12/25",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	item := gophkeeper.DataInfo{}
	require.NoError(t, item.FromBankCardDataInfo(data))

	entry, err := buildEntryFromData(&item)
	require.NoError(t, err)
	assert.Equal(t, model.EntryTypeBankCard, entry.Type)
	assert.Contains(t, entry.Payload, "1234567890123456")
}

func TestBuildEntryFromData_Text(t *testing.T) {
	data := gophkeeper.TextDataInfo{
		Id:        uuid.New(),
		Type:      gophkeeper.TextDataInfoTypeTEXT,
		Text:      "secret text",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	item := gophkeeper.DataInfo{}
	require.NoError(t, item.FromTextDataInfo(data))

	entry, err := buildEntryFromData(&item)
	require.NoError(t, err)
	assert.Equal(t, model.EntryTypeText, entry.Type)
	assert.Contains(t, entry.Payload, "secret text")
}

func TestBuildEntryFromData_File(t *testing.T) {
	fileID := uuid.New()
	data := gophkeeper.FileDataInfo{
		Id:        uuid.New(),
		Type:      gophkeeper.FileDataInfoTypeFILE,
		FileIds:   []uuid.UUID{fileID},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	item := gophkeeper.DataInfo{}
	require.NoError(t, item.FromFileDataInfo(data))

	entry, err := buildEntryFromData(&item)
	require.NoError(t, err)
	assert.Equal(t, model.EntryTypeFile, entry.Type)
	assert.Contains(t, entry.Payload, fileID.String())
}

func TestBuildDataFromEntry_LoginPassword(t *testing.T) {
	entry := model.DataEntry{
		Type:    model.EntryTypeLoginPassword,
		Payload: `{"login":"user","password":"pass"}`,
	}

	data, err := buildDataFromEntry(entry)
	require.NoError(t, err)
	require.NotNil(t, data)

	discriminator, err := data.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, string(gophkeeper.DataTypeLOGINPASSWORD), discriminator)
}

func TestBuildDataFromEntry_BankCard(t *testing.T) {
	entry := model.DataEntry{
		Type:    model.EntryTypeBankCard,
		Payload: `{"cardNumber":"1234","cardHolder":"John","cardExpiry":"12/25"}`,
	}

	data, err := buildDataFromEntry(entry)
	require.NoError(t, err)
	require.NotNil(t, data)

	discriminator, err := data.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, string(gophkeeper.DataTypeBANKCARD), discriminator)
}

func TestBuildDataFromEntry_Text(t *testing.T) {
	entry := model.DataEntry{
		Type:    model.EntryTypeText,
		Payload: `{"text":"secret"}`,
	}

	data, err := buildDataFromEntry(entry)
	require.NoError(t, err)
	require.NotNil(t, data)

	discriminator, err := data.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, string(gophkeeper.DataTypeTEXT), discriminator)
}

func TestBuildDataFromEntry_File(t *testing.T) {
	fileID := uuid.New()
	entry := model.DataEntry{
		Type:    model.EntryTypeFile,
		Payload: `{"fileIds":["` + fileID.String() + `"]}`,
	}

	data, err := buildDataFromEntry(entry)
	require.NoError(t, err)
	require.NotNil(t, data)

	discriminator, err := data.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, string(gophkeeper.DataTypeFILE), discriminator)
}

func TestBuildDataFromEntry_InvalidPayload(t *testing.T) {
	entry := model.DataEntry{
		Type:    model.EntryTypeLoginPassword,
		Payload: `invalid json`,
	}

	_, err := buildDataFromEntry(entry)
	assert.Error(t, err)
}

func TestBuildDataFromEntry_UnknownType(t *testing.T) {
	entry := model.DataEntry{
		Type:    "UNKNOWN",
		Payload: `{}`,
	}

	_, err := buildDataFromEntry(entry)
	assert.Error(t, err)
}

func TestExtractDataID(t *testing.T) {
	testCases := []struct {
		name       string
		item       *gophkeeper.DataInfo
		expectUUID bool
		expectErr  bool
	}{
		{
			name: "login password",
			item: func() *gophkeeper.DataInfo {
				item := &gophkeeper.DataInfo{}
				_ = item.FromLoginPasswordDataInfo(gophkeeper.LoginPasswordDataInfo{
					Id:   uuid.New(),
					Type: gophkeeper.LoginPasswordDataInfoTypeLOGINPASSWORD,
				})
				return item
			}(),
			expectUUID: true,
		},
		{
			name: "bank card",
			item: func() *gophkeeper.DataInfo {
				item := &gophkeeper.DataInfo{}
				_ = item.FromBankCardDataInfo(gophkeeper.BankCardDataInfo{
					Id:   uuid.New(),
					Type: gophkeeper.BankCardDataInfoTypeBANKCARD,
				})
				return item
			}(),
			expectUUID: true,
		},
		{
			name: "text",
			item: func() *gophkeeper.DataInfo {
				item := &gophkeeper.DataInfo{}
				_ = item.FromTextDataInfo(gophkeeper.TextDataInfo{
					Id:   uuid.New(),
					Type: gophkeeper.TextDataInfoTypeTEXT,
				})
				return item
			}(),
			expectUUID: true,
		},
		{
			name: "file",
			item: func() *gophkeeper.DataInfo {
				item := &gophkeeper.DataInfo{}
				_ = item.FromFileDataInfo(gophkeeper.FileDataInfo{
					Id:   uuid.New(),
					Type: gophkeeper.FileDataInfoTypeFILE,
				})
				return item
			}(),
			expectUUID: true,
		},
		{
			name:      "nil item",
			item:      nil,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := extractDataID(tc.item)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.expectUUID {
					assert.NotEqual(t, uuid.Nil, id)
				}
			}
		})
	}
}

func TestService_Sync_PushPull(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	tokens := model.Token{
		AccessToken:          "test-token",
		RefreshToken:         "test-refresh",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, store.SaveTokens(tokens))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"items":[],"page":1,"page_size":100,"total":0,"total_pages":1}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.Sync(ctx)
	assert.NoError(t, err)
}

func TestService_SyncPush_EntryError(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	entryID := uuid.New()
	entry := model.DataEntry{
		ID:         entryID,
		Type:       model.EntryTypeText,
		Payload:    `{"text":"test"}`,
		SyncStatus: model.SyncStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, store.DataPut(entry))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPush(ctx)
	assert.NoError(t, err)

	got, err := store.DataGet(entryID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.SyncStatusPending, got.SyncStatus)
	assert.NotNil(t, got.ErrorMessage)
}

func TestService_SyncPush_MultipleEntries(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	callCount := 0
	remoteIDs := []uuid.UUID{uuid.New(), uuid.New()}

	for i := 0; i < 2; i++ {
		entry := model.DataEntry{
			ID:         uuid.New(),
			Type:       model.EntryTypeText,
			Payload:    `{"text":"test"}`,
			SyncStatus: model.SyncStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(entry))
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, err := w.Write([]byte(`{"data":{"id":"` + remoteIDs[callCount-1].String() + `","type":"TEXT","text":"test","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}}`))
			assert.NoError(t, err)
		} else {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"items":[],"page":1,"page_size":100,"total":0,"total_pages":1}`))
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPush(ctx)
	require.NoError(t, err)

	pending, err := store.DataGetPending()
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestBuildEntryFromData_UnknownType(t *testing.T) {
	item := &gophkeeper.DataInfo{}
	err := item.FromTextDataInfo(gophkeeper.TextDataInfo{
		Id:   uuid.New(),
		Type: gophkeeper.TextDataInfoTypeTEXT,
		Text: "test",
	})
	require.NoError(t, err)

	entry, err := buildEntryFromData(item)
	require.NoError(t, err)
	assert.Equal(t, model.EntryTypeText, entry.Type)
}

func TestStrPtr(t *testing.T) {
	s := "test"
	ptr := strPtr(s)
	assert.NotNil(t, ptr)
	assert.Equal(t, s, *ptr)
}

func TestService_SyncPull_MultiplePages(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	pageCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		pageCalls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if pageCalls == 1 {
			_, err := w.Write([]byte(`{"items":[{"id":"550e8400-e29b-41d4-a716-446655440000","type":"TEXT","text":"test1","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"page":1,"page_size":1,"total":2,"total_pages":2}`))
			assert.NoError(t, err)
		} else {
			_, err := w.Write([]byte(`{"items":[{"id":"650e8400-e29b-41d4-a716-446655440000","type":"TEXT","text":"test2","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"page":2,"page_size":1,"total":2,"total_pages":2}`))
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx := context.Background()
	err = svc.syncPull(ctx)
	require.NoError(t, err)

	count, err := store.DataCount()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestService_SyncPush_ContextCancelled(t *testing.T) {
	store, cleanup := setupSyncTestStore(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		entry := model.DataEntry{
			ID:         uuid.New(),
			Type:       model.EntryTypeText,
			Payload:    `{"text":"test"}`,
			SyncStatus: model.SyncStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, store.DataPut(entry))
	}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	api, err := apiClient.NewClient(server.URL)
	require.NoError(t, err)
	api.SetAuthToken("test-token")

	l := newSyncTestLogger(t)
	svc := NewService(store, api, l, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err = svc.syncPush(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}
