package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_GenerateAndParse(t *testing.T) {
	testCases := []struct {
		name         string
		generate     func(jwt *JWT, userID uuid.UUID) (string, error)
		parse        func(jwt *JWT, tokenString string) (*Claims, error)
		expectedType TokenType
	}{
		{
			name:         "generate and parse access token",
			generate:     func(j *JWT, userID uuid.UUID) (string, error) { return j.GenerateAccess(userID) },
			parse:        func(j *JWT, tokenString string) (*Claims, error) { return j.ParseAccess(tokenString) },
			expectedType: TokenTypeAccess,
		},
		{
			name:         "generate and parse refresh token",
			generate:     func(j *JWT, userID uuid.UUID) (string, error) { return j.GenerateRefresh(userID) },
			parse:        func(j *JWT, tokenString string) (*Claims, error) { return j.ParseRefresh(tokenString) },
			expectedType: TokenTypeRefresh,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userID := uuid.New()
			jwt := NewJWT("testsecret", 15*time.Minute, 24*time.Hour)

			tokenString, err := tc.generate(jwt, userID)
			require.NoError(t, err)
			assert.NotEmpty(t, tokenString)

			claims, err := tc.parse(jwt, tokenString)
			require.NoError(t, err)
			assert.Equal(t, userID, claims.UserID)
			assert.Equal(t, tc.expectedType, claims.Type)
		})
	}
}

func TestJWT_ParseInvalidToken(t *testing.T) {
	testCases := []struct {
		name    string
		parse   func(jwt *JWT, tokenString string) (*Claims, error)
		token   string
		wantErr bool
	}{
		{
			name:    "invalid access token",
			parse:   func(j *JWT, tokenString string) (*Claims, error) { return j.ParseAccess(tokenString) },
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "invalid refresh token",
			parse:   func(j *JWT, tokenString string) (*Claims, error) { return j.ParseRefresh(tokenString) },
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "empty token",
			parse:   func(j *JWT, tokenString string) (*Claims, error) { return j.ParseAccess(tokenString) },
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jwt := NewJWT("testsecret", 15*time.Minute, 24*time.Hour)

			_, err := tc.parse(jwt, tc.token)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJWT_WrongTokenType(t *testing.T) {
	userID := uuid.New()
	jwt := NewJWT("testsecret", 15*time.Minute, 24*time.Hour)

	// Generate access token but try to parse as refresh
	accessToken, err := jwt.GenerateAccess(userID)
	require.NoError(t, err)

	_, err = jwt.ParseRefresh(accessToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected token type")

	// Generate refresh token but try to parse as access
	refreshToken, err := jwt.GenerateRefresh(userID)
	require.NoError(t, err)

	_, err = jwt.ParseAccess(refreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected token type")
}

func TestJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	jwt1 := NewJWT("secret1", 15*time.Minute, 24*time.Hour)
	jwt2 := NewJWT("secret2", 15*time.Minute, 24*time.Hour)

	token, err := jwt1.GenerateAccess(userID)
	require.NoError(t, err)

	_, err = jwt2.ParseAccess(token)
	assert.Error(t, err)
}

func TestJWT_TTLAccessors(t *testing.T) {
	accessTTL := 30 * time.Minute
	refreshTTL := 48 * time.Hour
	jwt := NewJWT("testsecret", accessTTL, refreshTTL)

	assert.Equal(t, accessTTL, jwt.AccessTTL())
	assert.Equal(t, refreshTTL, jwt.RefreshTTL())
}
