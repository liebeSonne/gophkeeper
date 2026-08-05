// nolint:goconst
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	apperrors "github.com/liebeSonne/gophkeeper/internal/errors"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

func TestHealthCheck(t *testing.T) {
	testCases := []struct {
		name           string
		expectedStatus int
		expectedBody   server.HealthResponse
	}{
		{
			name:           "returns ok status",
			expectedStatus: http.StatusOK,
			expectedBody: server.HealthResponse{
				Status: "ok",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockAuthService(t)

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Error(mock.Anything, mock.Anything).Return().Maybe()

			mockDataService := NewMockDataService(t)

			mockFileService := NewMockFileService(t)
			h := NewServerHandler(mockService, mockDataService, mockFileService, l)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/health", http.NoBody)
			w := httptest.NewRecorder()

			h.HealthCheck(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var resp server.HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBody.Status, resp.Status)
		})
	}
}

// nolint: dupl
func TestRegisterUser(t *testing.T) {
	conflictErr := repository.NewErrConflictUserLogin("existing", errors.New("unique violation"))
	someErr := errors.New("internal error")

	testCases := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "successful registration",
			body: server.RegisterRequest{Login: "newuser", Password: "password"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().Register(mock.Anything, "newuser", "password").Return(makeToken(), nil)
			},
			expectedStatus: http.StatusCreated,
			expectToken:    true,
		},
		{
			name:           "invalid JSON body",
			body:           testNotJSON,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty login",
			body:           server.RegisterRequest{Login: "", Password: "password"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty password",
			body:           server.RegisterRequest{Login: "user", Password: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "conflict error",
			body: server.RegisterRequest{Login: "existing", Password: "password"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().Register(mock.Anything, "existing", "password").Return(model.Token{}, conflictErr)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "internal error",
			body: server.RegisterRequest{Login: "user", Password: "password"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().Register(mock.Anything, "user", "password").Return(model.Token{}, someErr)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockService)
			}
			l := intlogger.NewMockLogger(t)
			l.EXPECT().Error(mock.Anything, mock.Anything).Return().Maybe()

			mockFileService := NewMockFileService(t)
			h := NewServerHandler(mockService, mockDataService, mockFileService, l)

			var bodyBytes []byte
			var err error
			if tc.body == nil {
				bodyBytes = []byte("{}")
			} else {
				bodyBytes, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.RegisterUser(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectToken {
				var resp server.TokenResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				apiResp := makeTokenResponse()
				assert.Equal(t, apiResp.AccessToken, resp.AccessToken)
				assert.Equal(t, apiResp.RefreshToken, resp.RefreshToken)
			}
		})
	}
}

// nolint: dupl
func TestLoginUser(t *testing.T) {
	invalidCredErr := apperrors.ErrInvalidCredentials
	someErr := errors.New("internal error")

	testCases := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "successful login",
			body: server.LoginRequest{Login: "user", Password: "password"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().Login(mock.Anything, "user", "password").Return(makeToken(), nil)
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name:           "invalid JSON body",
			body:           testNotJSON,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty login",
			body:           server.LoginRequest{Login: "", Password: "password"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty password",
			body:           server.LoginRequest{Login: "user", Password: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid credentials",
			body: server.LoginRequest{Login: "user", Password: "wrong"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().Login(mock.Anything, "user", "wrong").Return(model.Token{}, invalidCredErr)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "internal error",
			body: server.LoginRequest{Login: "user", Password: "password"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().Login(mock.Anything, "user", "password").Return(model.Token{}, someErr)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockService)
			}
			l := intlogger.NewMockLogger(t)
			l.EXPECT().Error(mock.Anything, mock.Anything).Return().Maybe()

			mockFileService := NewMockFileService(t)
			h := NewServerHandler(mockService, mockDataService, mockFileService, l)

			var bodyBytes []byte
			var err error
			if tc.body == nil {
				bodyBytes = []byte("{}")
			} else {
				bodyBytes, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.LoginUser(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectToken {
				var resp server.TokenResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				apiResp := makeTokenResponse()
				assert.Equal(t, apiResp.AccessToken, resp.AccessToken)
				assert.Equal(t, apiResp.RefreshToken, resp.RefreshToken)
			}
		})
	}
}

// nolint: dupl
func TestRefreshToken(t *testing.T) {
	invalidCredErr := apperrors.ErrInvalidCredentials
	someErr := errors.New("internal error")

	testCases := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockAuthService)
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "successful refresh",
			body: server.RefreshRequest{RefreshToken: "valid-refresh-token"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().RefreshToken(mock.Anything, "valid-refresh-token").Return(makeToken(), nil)
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name:           "invalid JSON body",
			body:           testNotJSON,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty refresh token",
			body:           server.RefreshRequest{RefreshToken: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid refresh token",
			body: server.RefreshRequest{RefreshToken: "invalid-token"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().RefreshToken(mock.Anything, "invalid-token").Return(model.Token{}, invalidCredErr)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "internal error",
			body: server.RefreshRequest{RefreshToken: "token"},
			setupMock: func(m *MockAuthService) {
				m.EXPECT().RefreshToken(mock.Anything, "token").Return(model.Token{}, someErr)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockService)
			}
			l := intlogger.NewMockLogger(t)
			l.EXPECT().Error(mock.Anything, mock.Anything).Return().Maybe()

			mockFileService := NewMockFileService(t)
			h := NewServerHandler(mockService, mockDataService, mockFileService, l)

			var bodyBytes []byte
			var err error
			if tc.body == nil {
				bodyBytes = []byte("{}")
			} else {
				bodyBytes, err = json.Marshal(tc.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.RefreshToken(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectToken {
				var resp server.TokenResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				apiResp := makeTokenResponse()
				assert.Equal(t, apiResp.AccessToken, resp.AccessToken)
				assert.Equal(t, apiResp.RefreshToken, resp.RefreshToken)
			}
		})
	}
}

func makeToken() model.Token {
	return model.Token{
		AccessToken:           "access-token",
		ExpiresIn:             900,
		RefreshToken:          "refresh-token",
		RefreshTokenExpiresIn: 86400,
	}
}

func makeTokenResponse() server.TokenResponse {
	return server.TokenResponse{
		AccessToken:           "access-token",
		ExpiresIn:             900,
		TokenType:             "Bearer",
		RefreshToken:          "refresh-token",
		RefreshTokenExpiresIn: 86400,
	}
}
