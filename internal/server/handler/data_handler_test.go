package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	authctx "github.com/liebeSonne/gophkeeper/internal/auth"
	"github.com/liebeSonne/gophkeeper/internal/errors"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/model"
)

func makeTestDataModel(userID, id uuid.UUID) *model.Data {
	return &model.Data{
		ID:       id,
		UserID:   userID,
		Type:     model.DataTypeLoginPassword,
		Payload:  mustJSON(model.LoginPasswordPayload{Login: "testuser", Password: "testpass"}),
		Metadata: "test metadata",
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func withUserID(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := authctx.CreateTokenContext(r.Context(), userID)
	return r.WithContext(ctx)
}

func TestCreateData(t *testing.T) {
	userID := uuid.New()
	dataID := uuid.New()

	testCases := []struct {
		name           string
		body           interface{}
		withUser       bool
		setupMock      func(*MockDataService)
		expectedStatus int
	}{
		{
			name:     "successful create",
			withUser: true,
			body: server.CreateDataRequest{
				Data: func() *server.Data {
					d := &server.Data{}
					_ = d.FromLoginPasswordData(server.LoginPasswordData{
						Type:     server.LoginPasswordDataTypeLOGINPASSWORD,
						Login:    "testuser",
						Password: "testpass",
					})
					return d
				}(),
			},
			setupMock: func(m *MockDataService) {
				m.On("CreateData", mock.Anything, userID, mock.Anything, model.DataTypeLoginPassword, "").
					Return(*makeTestDataModel(userID, dataID), nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "unauthorized",
			withUser:       false,
			body:           server.CreateDataRequest{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid JSON",
			withUser:       true,
			body:           "not json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "nil data",
			withUser: true,
			body: server.CreateDataRequest{
				Data: nil,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockDataService)
			}
			l := intlogger.NewMockLogger(t)
			l.On("Error", mock.Anything, mock.Anything).Return().Maybe()

			h := NewServerHandler(mockAuthService, mockDataService, l)

			var bodyBytes []byte
			var err error
			if tc.body == nil {
				bodyBytes = []byte("{}")
			} else {
				bodyBytes, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/data", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tc.withUser {
				req = withUserID(req, userID)
			}
			w := httptest.NewRecorder()

			h.CreateData(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestGetData(t *testing.T) {
	userID := uuid.New()
	testDataID := uuid.New()
	dataID := testDataID

	testCases := []struct {
		name           string
		withUser       bool
		setupMock      func(*MockDataService)
		expectedStatus int
	}{
		{
			name:     "successful get",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("GetData", mock.Anything, testDataID, userID).
					Return(*makeTestDataModel(userID, testDataID), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized",
			withUser:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "not found",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("GetData", mock.Anything, testDataID, userID).
					Return(model.Data{}, errors.ErrDataNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "access denied",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("GetData", mock.Anything, testDataID, userID).
					Return(model.Data{}, errors.ErrDataAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockDataService)
			}
			l := intlogger.NewMockLogger(t)
			l.On("Error", mock.Anything, mock.Anything).Return().Maybe()

			h := NewServerHandler(mockAuthService, mockDataService, l)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/data/"+testDataID.String(), http.NoBody)
			if tc.withUser {
				req = withUserID(req, userID)
			}
			w := httptest.NewRecorder()

			h.GetData(w, req, dataID)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestListData(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		withUser       bool
		setupMock      func(*MockDataService)
		expectedStatus int
	}{
		{
			name:     "successful list",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("ListData", mock.Anything, userID, 1, 20, []model.DataType(nil), (*string)(nil)).
					Return([]model.Data{*makeTestDataModel(userID, uuid.New())}, 1, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized",
			withUser:       false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockDataService)
			}
			l := intlogger.NewMockLogger(t)
			l.On("Error", mock.Anything, mock.Anything).Return().Maybe()

			h := NewServerHandler(mockAuthService, mockDataService, l)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/data/list", http.NoBody)
			if tc.withUser {
				req = withUserID(req, userID)
			}
			w := httptest.NewRecorder()

			h.ListData(w, req, server.ListDataParams{})

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestUpdateData(t *testing.T) {
	userID := uuid.New()
	testDataID := uuid.New()
	dataID := testDataID

	testCases := []struct {
		name           string
		withUser       bool
		setupMock      func(*MockDataService)
		expectedStatus int
	}{
		{
			name:     "successful update",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("UpdateData", mock.Anything, testDataID, userID, mock.Anything, model.DataTypeLoginPassword, "").
					Return(*makeTestDataModel(userID, testDataID), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized",
			withUser:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "not found",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("UpdateData", mock.Anything, testDataID, userID, mock.Anything, model.DataTypeLoginPassword, "").
					Return(model.Data{}, errors.ErrDataNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockDataService)
			}
			l := intlogger.NewMockLogger(t)
			l.On("Error", mock.Anything, mock.Anything).Return().Maybe()

			h := NewServerHandler(mockAuthService, mockDataService, l)

			body := server.UpdateDataRequest{
				Data: func() *server.Data {
					d := &server.Data{}
					_ = d.FromLoginPasswordData(server.LoginPasswordData{
						Type:     server.LoginPasswordDataTypeLOGINPASSWORD,
						Login:    "updated",
						Password: "updated",
					})
					return d
				}(),
			}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/data/"+testDataID.String(), bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tc.withUser {
				req = withUserID(req, userID)
			}
			w := httptest.NewRecorder()

			h.UpdateData(w, req, dataID)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestDeleteData(t *testing.T) {
	userID := uuid.New()
	testDataID := uuid.New()
	dataID := testDataID

	testCases := []struct {
		name           string
		withUser       bool
		setupMock      func(*MockDataService)
		expectedStatus int
	}{
		{
			name:     "successful delete",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("DeleteData", mock.Anything, testDataID, userID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "unauthorized",
			withUser:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "not found",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("DeleteData", mock.Anything, testDataID, userID).Return(errors.ErrDataNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "access denied",
			withUser: true,
			setupMock: func(m *MockDataService) {
				m.On("DeleteData", mock.Anything, testDataID, userID).Return(errors.ErrDataAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockDataService)
			}
			l := intlogger.NewMockLogger(t)
			l.On("Error", mock.Anything, mock.Anything).Return().Maybe()

			h := NewServerHandler(mockAuthService, mockDataService, l)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/data/"+testDataID.String(), http.NoBody)
			if tc.withUser {
				req = withUserID(req, userID)
			}
			w := httptest.NewRecorder()

			h.DeleteData(w, req, dataID)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
