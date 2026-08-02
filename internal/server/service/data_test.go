package service

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/liebeSonne/gophkeeper/internal/crypto"
	apperrors "github.com/liebeSonne/gophkeeper/internal/errors"
	"github.com/liebeSonne/gophkeeper/internal/model"
	"github.com/liebeSonne/gophkeeper/internal/repository"
)

type mockEncryptor struct {
	mock.Mock
}

func (m *mockEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	ret := m.Called(plaintext)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]byte), ret.Error(1)
}

func (m *mockEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	ret := m.Called(ciphertext)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]byte), ret.Error(1)
}

func makeTestPayload() []byte {
	p := model.LoginPasswordPayload{Login: "testuser", Password: "testpass"}
	b, _ := json.Marshal(p)
	return b
}

func makeTestData(userID, id uuid.UUID) *model.Data {
	return &model.Data{
		ID:       id,
		UserID:   userID,
		Type:     model.DataTypeLoginPassword,
		Payload:  makeTestPayload(),
		Metadata: "test metadata",
	}
}

func TestDataService_CreateData(t *testing.T) {
	testCases := []struct {
		name        string
		setupMocks  func(*MockDataRepository, *mockEncryptor)
		userID      uuid.UUID
		payloadJSON []byte
		dataType    model.DataType
		metadata    string
		expectError bool
		expectData  bool
	}{
		{
			name: "successful create",
			setupMocks: func(repo *MockDataRepository, enc *mockEncryptor) {
				enc.On("Encrypt", makeTestPayload()).Return([]byte("encrypted"), nil)
				repo.On("NextID", mock.Anything).Return(uuid.New())
				repo.On("Store", mock.Anything, mock.AnythingOfType("model.Data")).Return(nil)
			},
			userID:      uuid.New(),
			payloadJSON: makeTestPayload(),
			dataType:    model.DataTypeLoginPassword,
			metadata:    "test metadata",
			expectData:  true,
		},
		{
			name: "encrypt fails",
			setupMocks: func(_ *MockDataRepository, enc *mockEncryptor) {
				enc.On("Encrypt", makeTestPayload()).Return(nil, crypto.ErrDecryptionFailed)
			},
			userID:      uuid.New(),
			payloadJSON: makeTestPayload(),
			dataType:    model.DataTypeLoginPassword,
			expectError: true,
		},
		{
			name: "store fails",
			setupMocks: func(repo *MockDataRepository, enc *mockEncryptor) {
				enc.On("Encrypt", makeTestPayload()).Return([]byte("encrypted"), nil)
				repo.On("NextID", mock.Anything).Return(uuid.New())
				repo.On("Store", mock.Anything, mock.AnythingOfType("model.Data")).Return(errors.New("db error"))
			},
			userID:      uuid.New(),
			payloadJSON: makeTestPayload(),
			dataType:    model.DataTypeLoginPassword,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := &mockEncryptor{}
			tc.setupMocks(mockRepo, mockEnc)

			svc := NewDataService(mockRepo, mockEnc)
			result, err := svc.CreateData(t.Context(), tc.userID, tc.payloadJSON, tc.dataType, tc.metadata)

			if tc.expectError {
				require.Error(t, err)
				assert.Equal(t, model.Data{}, result)
			} else if tc.expectData {
				require.NoError(t, err)
				assert.Equal(t, tc.userID, result.UserID)
				assert.Equal(t, tc.dataType, result.Type)
				assert.Equal(t, tc.metadata, result.Metadata)
			}

			mockEnc.AssertExpectations(t)
		})
	}
}

func TestDataService_GetData(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	dataID := uuid.New()
	payload := makeTestPayload()

	testCases := []struct {
		name        string
		setupMocks  func(*MockDataRepository, *mockEncryptor)
		requestID   uuid.UUID
		requestUser uuid.UUID
		expectError bool
		expectErr   error
		expectData  bool
	}{
		{
			name: "successful get",
			setupMocks: func(repo *MockDataRepository, enc *mockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
				enc.On("Decrypt", mock.Anything).Return(payload, nil)
			},
			requestID:   dataID,
			requestUser: userID,
			expectData:  true,
		},
		{
			name: "not found",
			setupMocks: func(repo *MockDataRepository, _ *mockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{}, repository.ErrNotFound)
			},
			requestID:   dataID,
			requestUser: userID,
			expectError: true,
			expectErr:   apperrors.ErrDataNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockDataRepository, _ *mockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
			},
			requestID:   dataID,
			requestUser: otherUserID,
			expectError: true,
			expectErr:   apperrors.ErrDataAccessDenied,
		},
		{
			name: "decrypt fails",
			setupMocks: func(repo *MockDataRepository, enc *mockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
				enc.On("Decrypt", mock.Anything).Return(nil, crypto.ErrDecryptionFailed)
			},
			requestID:   dataID,
			requestUser: userID,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := &mockEncryptor{}
			tc.setupMocks(mockRepo, mockEnc)

			svc := NewDataService(mockRepo, mockEnc)
			result, err := svc.GetData(t.Context(), tc.requestID, tc.requestUser)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectErr != nil {
					assert.ErrorIs(t, err, tc.expectErr)
				}
			} else if tc.expectData {
				require.NoError(t, err)
				assert.Equal(t, dataID, result.ID)
			}

			mockEnc.AssertExpectations(t)
		})
	}
}

func TestDataService_ListData(t *testing.T) {
	userID := uuid.New()
	payload := makeTestPayload()

	testCases := []struct {
		name        string
		setupMocks  func(*MockDataRepository, *mockEncryptor)
		page        int
		pageSize    int
		dataTypes   []model.DataType
		query       *string
		expectError bool
		expectCount int
		expectTotal int
	}{
		{
			name: "successful list single page",
			setupMocks: func(repo *MockDataRepository, enc *mockEncryptor) {
				repo.On("Count", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return(3, nil)
				repo.On("List", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return([]model.Data{
					*makeTestData(userID, uuid.New()),
					*makeTestData(userID, uuid.New()),
					*makeTestData(userID, uuid.New()),
				}, nil)
				enc.On("Decrypt", mock.Anything).Return(payload, nil).Times(3)
			},
			page:        1,
			pageSize:    20,
			expectCount: 3,
			expectTotal: 3,
		},
		{
			name: "pagination page 2",
			setupMocks: func(repo *MockDataRepository, enc *mockEncryptor) {
				repo.On("Count", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return(5, nil)
				repo.On("List", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return([]model.Data{
					*makeTestData(userID, uuid.New()),
					*makeTestData(userID, uuid.New()),
				}, nil)
				enc.On("Decrypt", mock.Anything).Return(payload, nil).Times(2)
			},
			page:        2,
			pageSize:    3,
			expectCount: 2,
			expectTotal: 5,
		},
		{
			name: "empty list",
			setupMocks: func(repo *MockDataRepository, _ *mockEncryptor) {
				repo.On("Count", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return(0, nil)
				repo.On("List", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return([]model.Data{}, nil)
			},
			page:        1,
			pageSize:    20,
			expectCount: 0,
			expectTotal: 0,
		},
		{
			name: "count fails",
			setupMocks: func(repo *MockDataRepository, _ *mockEncryptor) {
				repo.On("Count", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return(0, errors.New("db error"))
			},
			page:        1,
			pageSize:    20,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := &mockEncryptor{}
			tc.setupMocks(mockRepo, mockEnc)

			svc := NewDataService(mockRepo, mockEnc)
			items, total, err := svc.ListData(t.Context(), userID, tc.page, tc.pageSize, tc.dataTypes, tc.query)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectTotal, total)
				assert.Len(t, items, tc.expectCount)
			}

			mockEnc.AssertExpectations(t)
		})
	}
}

func TestDataService_UpdateData(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	dataID := uuid.New()
	newPayload := []byte(`{"login":"newlogin","password":"newpass"}`)

	testCases := []struct {
		name        string
		setupMocks  func(*MockDataRepository, *mockEncryptor)
		expectError bool
		expectErr   error
		expectData  bool
	}{
		{
			name: "successful update",
			setupMocks: func(repo *MockDataRepository, enc *mockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
				enc.On("Encrypt", newPayload).Return([]byte("encrypted"), nil)
				repo.On("Store", mock.Anything, mock.AnythingOfType("model.Data")).Return(nil)
			},
			expectData: true,
		},
		{
			name: "not found",
			setupMocks: func(repo *MockDataRepository, _ *mockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{}, repository.ErrNotFound)
			},
			expectError: true,
			expectErr:   apperrors.ErrDataNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockDataRepository, _ *mockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
			},
			expectError: true,
			expectErr:   apperrors.ErrDataAccessDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := &mockEncryptor{}
			tc.setupMocks(mockRepo, mockEnc)

			var requestUser uuid.UUID
			if tc.name == "access denied" {
				requestUser = otherUserID
			} else {
				requestUser = userID
			}

			svc := NewDataService(mockRepo, mockEnc)
			result, err := svc.UpdateData(t.Context(), dataID, requestUser, newPayload, model.DataTypeLoginPassword, "updated")

			if tc.expectError {
				require.Error(t, err)
				if tc.expectErr != nil {
					assert.ErrorIs(t, err, tc.expectErr)
				}
			} else if tc.expectData {
				require.NoError(t, err)
				assert.Equal(t, "updated", result.Metadata)
			}

			mockEnc.AssertExpectations(t)
		})
	}
}

func TestDataService_DeleteData(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	dataID := uuid.New()

	testCases := []struct {
		name        string
		setupMocks  func(*MockDataRepository)
		requestUser uuid.UUID
		expectError bool
		expectErr   error
	}{
		{
			name: "successful delete",
			setupMocks: func(repo *MockDataRepository) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
				repo.On("Delete", mock.Anything, []uuid.UUID{dataID}).Return(nil)
			},
			requestUser: userID,
		},
		{
			name: "not found",
			setupMocks: func(repo *MockDataRepository) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{}, repository.ErrNotFound)
			},
			requestUser: userID,
			expectError: true,
			expectErr:   apperrors.ErrDataNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockDataRepository) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
			},
			requestUser: otherUserID,
			expectError: true,
			expectErr:   apperrors.ErrDataAccessDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			tc.setupMocks(mockRepo)

			svc := NewDataService(mockRepo, &mockEncryptor{})
			err := svc.DeleteData(t.Context(), dataID, tc.requestUser)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectErr != nil {
					assert.ErrorIs(t, err, tc.expectErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
