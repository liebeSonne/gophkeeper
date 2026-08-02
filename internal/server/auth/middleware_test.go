package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_Handle(t *testing.T) {
	testUserID := uuid.New()

	testCases := []struct {
		name           string
		tokenUserID    uuid.UUID
		tokenUserIDErr error
		header         string
		expectedStatus int
		expectUserID   bool
	}{
		{
			name:           "valid bearer token",
			tokenUserID:    testUserID,
			tokenUserIDErr: nil,
			header:         "Bearer validtoken",
			expectedStatus: http.StatusOK,
			expectUserID:   true,
		},
		{
			name:           "missing token",
			tokenUserID:    uuid.Nil,
			tokenUserIDErr: nil,
			header:         "",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "invalid token format",
			tokenUserID:    uuid.Nil,
			tokenUserIDErr: nil,
			header:         "InvalidFormat token",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
		{
			name:           "invalid token from service",
			tokenUserID:    uuid.Nil,
			tokenUserIDErr: assert.AnError,
			header:         "Bearer invalidtoken",
			expectedStatus: http.StatusUnauthorized,
			expectUserID:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := NewMockService(t)
			if tc.header != "" && tc.header[:7] == "Bearer " {
				service.EXPECT().GetUserIDFromToken(mock.Anything).Return(tc.tokenUserID, tc.tokenUserIDErr)
			}

			middleware := NewAuthMiddleware(service)

			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()

			middleware.Handle(next).ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestExtractToken(t *testing.T) {
	testCases := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "valid bearer token",
			header:   "Bearer mytoken123",
			expected: "mytoken123",
		},
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "invalid prefix",
			header:   "Basic mytoken123",
			expected: "",
		},
		{
			name:     "bearer without space",
			header:   "Bearermytoken123",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			got := extractToken(req)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestMiddleware_ToMiddlewareFunc(t *testing.T) {
	testUserID := uuid.New()

	service := NewMockService(t)
	service.EXPECT().GetUserIDFromToken("validtoken").Return(testUserID, nil)

	middleware := NewAuthMiddleware(service)
	middlewareFunc := middleware.ToMiddlewareFunc()

	require.NotNil(t, middlewareFunc)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middlewareFunc(next)
	require.NotNil(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer validtoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
