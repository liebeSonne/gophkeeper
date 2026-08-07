// nolint:goconst
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
	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

//nolint:gosec
func makeTestPayload() []byte {
	p := model.LoginPasswordPayload{Login: testLogin, Password: testPassword}
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
		setupMocks  func(*MockDataRepository, *crypto.MockEncryptor)
		userID      uuid.UUID
		payloadJSON []byte
		dataType    model.DataType
		metadata    string
		expectError bool
		expectData  bool
	}{
		{
			name: "successful create",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor) {
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
			setupMocks: func(_ *MockDataRepository, enc *crypto.MockEncryptor) {
				enc.On("Encrypt", makeTestPayload()).Return(nil, crypto.ErrDecryptionFailed)
			},
			userID:      uuid.New(),
			payloadJSON: makeTestPayload(),
			dataType:    model.DataTypeLoginPassword,
			expectError: true,
		},
		{
			name: "store fails",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor) {
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
			mockEnc := crypto.NewMockEncryptor(t)
			mockFileProvider := NewMockFileProvider(t)
			tc.setupMocks(mockRepo, mockEnc)

			svc := NewDataService(mockRepo, mockEnc, mockFileProvider)
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
		setupMocks  func(*MockDataRepository, *crypto.MockEncryptor)
		requestID   uuid.UUID
		requestUser uuid.UUID
		expectError bool
		expectErr   error
		expectData  bool
	}{
		{
			name: "successful get",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
				enc.On("Decrypt", mock.Anything).Return(payload, nil)
			},
			requestID:   dataID,
			requestUser: userID,
			expectData:  true,
		},
		{
			name: "not found",
			setupMocks: func(repo *MockDataRepository, _ *crypto.MockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{}, repository.ErrNotFound)
			},
			requestID:   dataID,
			requestUser: userID,
			expectError: true,
			expectErr:   ErrDataNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockDataRepository, _ *crypto.MockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
			},
			requestID:   dataID,
			requestUser: otherUserID,
			expectError: true,
			expectErr:   ErrDataAccessDenied,
		},
		{
			name: "decrypt fails",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor) {
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
			mockEnc := crypto.NewMockEncryptor(t)
			mockFileProvider := NewMockFileProvider(t)
			tc.setupMocks(mockRepo, mockEnc)

			svc := NewDataService(mockRepo, mockEnc, mockFileProvider)
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
		setupMocks  func(*MockDataRepository, *crypto.MockEncryptor)
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
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor) {
				repo.On("ListWithCount", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return([]model.Data{
					*makeTestData(userID, uuid.New()),
					*makeTestData(userID, uuid.New()),
					*makeTestData(userID, uuid.New()),
				}, 3, nil)
				enc.On("Decrypt", mock.Anything).Return(payload, nil).Times(3)
			},
			page:        1,
			pageSize:    20,
			expectCount: 3,
			expectTotal: 3,
		},
		{
			name: "pagination page 2",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor) {
				repo.On("ListWithCount", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return([]model.Data{
					*makeTestData(userID, uuid.New()),
					*makeTestData(userID, uuid.New()),
				}, 5, nil)
				enc.On("Decrypt", mock.Anything).Return(payload, nil).Times(2)
			},
			page:        2,
			pageSize:    3,
			expectCount: 2,
			expectTotal: 5,
		},
		{
			name: "empty list",
			setupMocks: func(repo *MockDataRepository, _ *crypto.MockEncryptor) {
				repo.On("ListWithCount", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return([]model.Data{}, 0, nil)
			},
			page:        1,
			pageSize:    20,
			expectCount: 0,
			expectTotal: 0,
		},
		{
			name: "list with count fails",
			setupMocks: func(repo *MockDataRepository, _ *crypto.MockEncryptor) {
				repo.On("ListWithCount", mock.Anything, mock.AnythingOfType("db.ListSpec")).Return([]model.Data{}, 0, errors.New("db error"))
			},
			page:        1,
			pageSize:    20,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockFileProvider := NewMockFileProvider(t)
			tc.setupMocks(mockRepo, mockEnc)

			svc := NewDataService(mockRepo, mockEnc, mockFileProvider)
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
		setupMocks  func(*MockDataRepository, *crypto.MockEncryptor)
		expectError bool
		expectErr   error
		expectData  bool
	}{
		{
			name: "successful update",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
				enc.On("Encrypt", newPayload).Return([]byte("encrypted"), nil)
				repo.On("Store", mock.Anything, mock.AnythingOfType("model.Data")).Return(nil)
			},
			expectData: true,
		},
		{
			name: "not found",
			setupMocks: func(repo *MockDataRepository, _ *crypto.MockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{}, repository.ErrNotFound)
			},
			expectError: true,
			expectErr:   ErrDataNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockDataRepository, _ *crypto.MockEncryptor) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
			},
			expectError: true,
			expectErr:   ErrDataAccessDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockFileProvider := NewMockFileProvider(t)
			tc.setupMocks(mockRepo, mockEnc)

			var requestUser uuid.UUID
			if tc.name == "access denied" {
				requestUser = otherUserID
			} else {
				requestUser = userID
			}

			svc := NewDataService(mockRepo, mockEnc, mockFileProvider)
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
			expectErr:   ErrDataNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockDataRepository) {
				repo.On("GetByID", mock.Anything, dataID).Return(*makeTestData(userID, dataID), nil)
			},
			requestUser: otherUserID,
			expectError: true,
			expectErr:   ErrDataAccessDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			tc.setupMocks(mockRepo)
			mockEnc := crypto.NewMockEncryptor(t)
			mockFileProvider := NewMockFileProvider(t)

			svc := NewDataService(mockRepo, mockEnc, mockFileProvider)
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

func TestDataService_CreateData_FileType(t *testing.T) {
	userID := uuid.New()
	fileID1 := uuid.New()
	fileID2 := uuid.New()
	payloadJSON, _ := json.Marshal(model.FilePayload{FileIDs: []uuid.UUID{fileID1, fileID2}})

	testCases := []struct {
		name        string
		setupMocks  func(*MockDataRepository, *crypto.MockEncryptor, *MockFileProvider)
		expectError bool
		expectErr   error
	}{
		{
			name: "successful create with valid file IDs",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor, fp *MockFileProvider) {
				fp.On("GetExistingFilesByUserID", mock.Anything, userID, []uuid.UUID{fileID1, fileID2}).Return([]uuid.UUID{fileID1, fileID2}, nil)
				enc.On("Encrypt", payloadJSON).Return([]byte("encrypted"), nil)
				repo.On("NextID", mock.Anything).Return(uuid.New())
				repo.On("Store", mock.Anything, mock.AnythingOfType("model.Data")).Return(nil)
			},
		},
		{
			name: "empty file IDs",
			setupMocks: func(_ *MockDataRepository, _ *crypto.MockEncryptor, _ *MockFileProvider) {
			},
			expectError: true,
			expectErr:   ErrFileReferenceInvalid,
		},
		{
			name: "file not owned by user",
			setupMocks: func(_ *MockDataRepository, _ *crypto.MockEncryptor, fp *MockFileProvider) {
				fp.On("GetExistingFilesByUserID", mock.Anything, userID, []uuid.UUID{fileID1, fileID2}).Return([]uuid.UUID{fileID1}, nil)
			},
			expectError: true,
			expectErr:   ErrFileReferenceInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockFP := NewMockFileProvider(t)
			tc.setupMocks(mockRepo, mockEnc, mockFP)

			var payload []byte
			if tc.name == "empty file IDs" {
				payload, _ = json.Marshal(model.FilePayload{FileIDs: []uuid.UUID{}})
			} else {
				payload = payloadJSON
			}

			svc := NewDataService(mockRepo, mockEnc, mockFP)
			_, err := svc.CreateData(t.Context(), userID, payload, model.DataTypeFile, "test file")

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

func TestDataService_UpdateData_FileType(t *testing.T) {
	userID := uuid.New()
	dataID := uuid.New()
	fileID1 := uuid.New()
	fileID2 := uuid.New()
	fileID3 := uuid.New()
	oldPayload, _ := json.Marshal(model.FilePayload{FileIDs: []uuid.UUID{fileID1, fileID2}})

	testCases := []struct {
		name        string
		setupMocks  func(*MockDataRepository, *crypto.MockEncryptor, *MockFileProvider)
		newPayload  []byte
		expectError bool
		expectErr   error
	}{
		{
			name: "file IDs unchanged - skip validation",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor, _ *MockFileProvider) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{ID: dataID, UserID: userID, Type: model.DataTypeFile, Payload: []byte("encrypted-old")}, nil)
				enc.On("Decrypt", []byte("encrypted-old")).Return(oldPayload, nil)
				enc.On("Encrypt", mock.Anything).Return([]byte("encrypted-new"), nil)
				repo.On("Store", mock.Anything, mock.AnythingOfType("model.Data")).Return(nil)
			},
			newPayload: oldPayload,
		},
		{
			name: "file IDs changed - validate new",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor, fp *MockFileProvider) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{ID: dataID, UserID: userID, Type: model.DataTypeFile, Payload: []byte("encrypted-old")}, nil)
				enc.On("Decrypt", []byte("encrypted-old")).Return(oldPayload, nil)
				fp.On("GetExistingFilesByUserID", mock.Anything, userID, []uuid.UUID{fileID2, fileID3}).Return([]uuid.UUID{fileID2, fileID3}, nil)
				enc.On("Encrypt", mock.Anything).Return([]byte("encrypted-new"), nil)
				repo.On("Store", mock.Anything, mock.AnythingOfType("model.Data")).Return(nil)
			},
			newPayload: func() []byte {
				b, _ := json.Marshal(model.FilePayload{FileIDs: []uuid.UUID{fileID2, fileID3}})
				return b
			}(),
		},
		{
			name: "file IDs changed - validation fails",
			setupMocks: func(repo *MockDataRepository, enc *crypto.MockEncryptor, fp *MockFileProvider) {
				repo.On("GetByID", mock.Anything, dataID).Return(model.Data{ID: dataID, UserID: userID, Type: model.DataTypeFile, Payload: []byte("encrypted-old")}, nil)
				enc.On("Decrypt", []byte("encrypted-old")).Return(oldPayload, nil)
				fp.On("GetExistingFilesByUserID", mock.Anything, userID, []uuid.UUID{fileID2, fileID3}).Return([]uuid.UUID{fileID2}, nil)
			},
			newPayload: func() []byte {
				b, _ := json.Marshal(model.FilePayload{FileIDs: []uuid.UUID{fileID2, fileID3}})
				return b
			}(),
			expectError: true,
			expectErr:   ErrFileReferenceInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockDataRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockFP := NewMockFileProvider(t)
			tc.setupMocks(mockRepo, mockEnc, mockFP)

			svc := NewDataService(mockRepo, mockEnc, mockFP)
			_, err := svc.UpdateData(t.Context(), dataID, userID, tc.newPayload, model.DataTypeFile, "updated")

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

func TestUnmarshalPayload(t *testing.T) {
	testCases := []struct {
		name      string
		payload   []byte
		dataType  model.DataType
		wantType  interface{}
		expectErr bool
	}{
		{
			name:     "login password payload",
			payload:  []byte(`{"login":"testuser","password":"testpass"}`),
			dataType: model.DataTypeLoginPassword,
			wantType: model.LoginPasswordPayload{},
		},
		{
			name:     "bank card payload",
			payload:  []byte(`{"cardNumber":"1234567890123456","cardHolder":"John Doe","cardExpiry":"12/25"}`),
			dataType: model.DataTypeBankCard,
			wantType: model.BankCardPayload{},
		},
		{
			name:     "text payload",
			payload:  []byte(`{"text":"secret text"}`),
			dataType: model.DataTypeText,
			wantType: model.TextPayload{},
		},
		{
			name:     "file payload",
			payload:  []byte(`{"fileIds":["550e8400-e29b-41d4-a716-446655440000"]}`),
			dataType: model.DataTypeFile,
			wantType: model.FilePayload{},
		},
		{
			name:      "invalid login password payload",
			payload:   []byte(`invalid json`),
			dataType:  model.DataTypeLoginPassword,
			expectErr: true,
		},
		{
			name:      "invalid bank card payload",
			payload:   []byte(`invalid json`),
			dataType:  model.DataTypeBankCard,
			expectErr: true,
		},
		{
			name:      "invalid text payload",
			payload:   []byte(`invalid json`),
			dataType:  model.DataTypeText,
			expectErr: true,
		},
		{
			name:      "invalid file payload",
			payload:   []byte(`invalid json`),
			dataType:  model.DataTypeFile,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := unmarshalPayload(tc.payload, tc.dataType)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.IsType(t, tc.wantType, result)
		})
	}
}
