package service

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/liebeSonne/gophkeeper/internal/jwt"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

const (
	testLogin     = "testuser"
	testPassword  = "password123"
	testTokenHash = "hashed"
)

func hashPassword(password string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed)
}

func TestAuthService_Register(t *testing.T) {
	testCases := []struct {
		name       string
		login      string
		password   string
		setupMocks func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository)
		wantErr    bool
		isConflict bool
	}{
		{
			name:     "successful registration",
			login:    testLogin,
			password: testPassword,
			setupMocks: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userID := uuid.New()
				userRepo.EXPECT().NextID(mock.Anything).Return(userID)
				userRepo.EXPECT().Store(mock.Anything, mock.MatchedBy(func(u model.User) bool {
					return u.Login == testLogin
				})).Return(nil)
				tokenID := uuid.New()
				tokenRepo.EXPECT().NextID(mock.Anything).Return(tokenID)
				tokenRepo.EXPECT().Store(mock.Anything, mock.MatchedBy(func(tok model.RefreshToken) bool {
					return tok.UserID == userID
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "conflict error",
			login:    "existinguser",
			password: testPassword,
			setupMocks: func(userRepo *MockUserRepository, _ *MockTokenRepository) {
				userID := uuid.New()
				userRepo.EXPECT().NextID(mock.Anything).Return(userID)
				userRepo.EXPECT().Store(mock.Anything, mock.MatchedBy(func(u model.User) bool {
					return u.Login == "existinguser"
				})).Return(repository.NewErrConflictUserLogin("existinguser", errors.New("unique violation")))
			},
			wantErr:    true,
			isConflict: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := NewMockUserRepository(t)
			tokenRepo := NewMockTokenRepository(t)
			j := jwt.NewJWT("testsecret", 15*time.Minute, 24*time.Hour)
			svc := NewAuthService(userRepo, tokenRepo, j)

			tc.setupMocks(userRepo, tokenRepo)

			ctx := t.Context()
			resp, err := svc.Register(ctx, tc.login, tc.password)

			if tc.wantErr {
				require.Error(t, err)
				if tc.isConflict {
					assert.True(t, repository.IsConflict(err))
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, resp.AccessToken)
			assert.NotEmpty(t, resp.RefreshToken)
			assert.Equal(t, 900, resp.ExpiresIn)
			assert.Equal(t, 86400, resp.RefreshTokenExpiresIn)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	testCases := []struct {
		name       string
		login      string
		password   string
		setupMocks func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository)
		wantErr    bool
		errType    error
	}{
		{
			name:     "successful login",
			login:    testLogin,
			password: testPassword,
			setupMocks: func(userRepo *MockUserRepository, tokenRepo *MockTokenRepository) {
				userID := uuid.New()
				userRepo.EXPECT().GetByLogin(mock.Anything, testLogin).Return(model.User{
					ID:       userID,
					Login:    testLogin,
					Password: hashPassword(testPassword),
				}, nil)
				tokenID := uuid.New()
				tokenRepo.EXPECT().NextID(mock.Anything).Return(tokenID)
				tokenRepo.EXPECT().Store(mock.Anything, mock.MatchedBy(func(tok model.RefreshToken) bool {
					return tok.UserID == userID
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "user not found",
			login:    "unknownuser",
			password: testPassword,
			setupMocks: func(userRepo *MockUserRepository, _ *MockTokenRepository) {
				userRepo.EXPECT().GetByLogin(mock.Anything, "unknownuser").Return(model.User{}, repository.ErrNotFound)
			},
			wantErr: true,
			errType: ErrInvalidCredentials,
		},
		{
			name:     "invalid password",
			login:    testLogin,
			password: "wrongpassword",
			setupMocks: func(userRepo *MockUserRepository, _ *MockTokenRepository) {
				userRepo.EXPECT().GetByLogin(mock.Anything, testLogin).Return(model.User{
					ID:       uuid.New(),
					Login:    testLogin,
					Password: hashPassword(testPassword),
				}, nil)
			},
			wantErr: true,
			errType: ErrInvalidCredentials,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := NewMockUserRepository(t)
			tokenRepo := NewMockTokenRepository(t)
			j := jwt.NewJWT("testsecret", 15*time.Minute, 24*time.Hour)
			svc := NewAuthService(userRepo, tokenRepo, j)

			tc.setupMocks(userRepo, tokenRepo)

			ctx := t.Context()
			resp, err := svc.Login(ctx, tc.login, tc.password)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, resp.AccessToken)
			assert.NotEmpty(t, resp.RefreshToken)
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	testCases := []struct {
		name           string
		refreshToken   string
		setupUserRepo  func(userRepo *MockUserRepository)
		setupTokenRepo func(tokenRepo *MockTokenRepository)
		wantErr        bool
		errType        error
	}{
		{
			name:         "successful refresh",
			refreshToken: "validrefresh",
			setupUserRepo: func(_ *MockUserRepository) {
			},
			setupTokenRepo: func(tokenRepo *MockTokenRepository) {
				oldTokenID := uuid.New()
				userID := uuid.New()
				tokenRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(model.RefreshToken{
					ID:        oldTokenID,
					UserID:    userID,
					TokenHash: testTokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}, nil)
				tokenRepo.EXPECT().Revoke(mock.Anything, oldTokenID).Return(nil)
				newTokenID := uuid.New()
				tokenRepo.EXPECT().NextID(mock.Anything).Return(newTokenID)
				tokenRepo.EXPECT().Store(mock.Anything, mock.MatchedBy(func(tok model.RefreshToken) bool {
					return tok.UserID == userID
				})).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "token not found",
			refreshToken: "invalidrefresh",
			setupUserRepo: func(_ *MockUserRepository) {
			},
			setupTokenRepo: func(tokenRepo *MockTokenRepository) {
				tokenRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(model.RefreshToken{}, repository.ErrNotFound)
			},
			wantErr: true,
			errType: ErrInvalidCredentials,
		},
		{
			name:         "revoked token",
			refreshToken: "revokedrefresh",
			setupUserRepo: func(_ *MockUserRepository) {
			},
			setupTokenRepo: func(tokenRepo *MockTokenRepository) {
				revokedAt := time.Now()
				tokenRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(model.RefreshToken{
					ID:        uuid.New(),
					UserID:    uuid.New(),
					TokenHash: testTokenHash,
					ExpiresAt: time.Now().Add(24 * time.Hour),
					RevokedAt: &revokedAt,
				}, nil)
			},
			wantErr: true,
			errType: ErrInvalidCredentials,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := NewMockUserRepository(t)
			tokenRepo := NewMockTokenRepository(t)
			j := jwt.NewJWT("testsecret", 15*time.Minute, 24*time.Hour)
			svc := NewAuthService(userRepo, tokenRepo, j)

			tc.setupUserRepo(userRepo)
			tc.setupTokenRepo(tokenRepo)

			ctx := t.Context()
			resp, err := svc.RefreshToken(ctx, tc.refreshToken)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, resp.AccessToken)
			assert.NotEmpty(t, resp.RefreshToken)
		})
	}
}

func TestAuthService_RevokeToken(t *testing.T) {
	testCases := []struct {
		name           string
		refreshToken   string
		setupTokenRepo func(tokenRepo *MockTokenRepository)
		wantErr        bool
	}{
		{
			name:         "successful revoke",
			refreshToken: "validrefresh",
			setupTokenRepo: func(tokenRepo *MockTokenRepository) {
				tokenID := uuid.New()
				tokenRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(model.RefreshToken{
					ID:        tokenID,
					TokenHash: testTokenHash,
				}, nil)
				tokenRepo.EXPECT().Revoke(mock.Anything, tokenID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "token not found (idempotent)",
			refreshToken: "invalidrefresh",
			setupTokenRepo: func(tokenRepo *MockTokenRepository) {
				tokenRepo.EXPECT().GetByTokenHash(mock.Anything, mock.Anything).Return(model.RefreshToken{}, repository.ErrNotFound)
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := NewMockUserRepository(t)
			tokenRepo := NewMockTokenRepository(t)
			j := jwt.NewJWT("testsecret", 15*time.Minute, 24*time.Hour)
			svc := NewAuthService(userRepo, tokenRepo, j)

			tc.setupTokenRepo(tokenRepo)

			ctx := t.Context()
			err := svc.RevokeToken(ctx, tc.refreshToken)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_GetUserIDFromToken(t *testing.T) {
	testCases := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "",
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := NewMockUserRepository(t)
			tokenRepo := NewMockTokenRepository(t)
			j := jwt.NewJWT("testsecret", 15*time.Minute, 24*time.Hour)
			svc := NewAuthService(userRepo, tokenRepo, j)

			if tc.name == "valid token" {
				userID := uuid.New()
				var err error
				tc.token, err = j.GenerateAccess(userID)
				require.NoError(t, err)
			}

			userID, err := svc.GetUserIDFromToken(tc.token)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Equal(t, uuid.Nil, userID)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, userID)
			}
		})
	}
}
