package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/liebeSonne/gophkeeper/internal/errors"
	"github.com/liebeSonne/gophkeeper/internal/jwt"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

type AuthService struct {
	userRepo   UserRepository
	tokenRepo  TokenRepository
	jwtService *jwt.JWT
}

func NewAuthService(userRepo UserRepository, tokenRepo TokenRepository, jwtService *jwt.JWT) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtService: jwtService,
	}
}

func (s *AuthService) Register(ctx context.Context, login, password string) (model.Token, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.Token{}, fmt.Errorf("hash password: %w", err)
	}

	user := model.User{
		ID:        s.userRepo.NextID(ctx),
		Login:     login,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	err = s.userRepo.Store(ctx, user)
	if err != nil {
		return model.Token{}, err
	}

	return s.generateTokenResponse(ctx, user.ID)
}

func (s *AuthService) Login(ctx context.Context, login, password string) (model.Token, error) {
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Token{}, apperrors.ErrInvalidCredentials
		}
		return model.Token{}, fmt.Errorf("get user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return model.Token{}, apperrors.ErrInvalidCredentials
	}

	return s.generateTokenResponse(ctx, user.ID)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (model.Token, error) {
	tokenHash := s.hashToken(refreshToken)
	storedToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Token{}, apperrors.ErrInvalidCredentials
		}
		return model.Token{}, fmt.Errorf("get token: %w", err)
	}

	if !storedToken.IsActive() {
		return model.Token{}, apperrors.ErrInvalidCredentials
	}

	err = s.tokenRepo.Revoke(ctx, storedToken.ID)
	if err != nil {
		return model.Token{}, fmt.Errorf("revoke token: %w", err)
	}

	return s.generateTokenResponse(ctx, storedToken.UserID)
}

func (s *AuthService) RevokeToken(ctx context.Context, refreshToken string) error {
	tokenHash := s.hashToken(refreshToken)
	storedToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get token: %w", err)
	}

	err = s.tokenRepo.Revoke(ctx, storedToken.ID)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}

	return nil
}

func (s *AuthService) GetUserIDFromToken(tokenString string) (uuid.UUID, error) {
	claims, err := s.jwtService.ParseAccess(tokenString)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) generateTokenResponse(ctx context.Context, userID uuid.UUID) (model.Token, error) {
	refreshToken, err := s.jwtService.GenerateRefresh(userID)
	if err != nil {
		return model.Token{}, fmt.Errorf("generate refresh token: %w", err)
	}

	accessToken, err := s.jwtService.GenerateAccess(userID)
	if err != nil {
		return model.Token{}, fmt.Errorf("generate access token: %w", err)
	}

	tokenHash := s.hashToken(refreshToken)
	token := model.RefreshToken{
		ID:        s.tokenRepo.NextID(ctx),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.jwtService.RefreshTTL()),
		CreatedAt: time.Now(),
	}

	err = s.tokenRepo.Store(ctx, token)
	if err != nil {
		return model.Token{}, fmt.Errorf("store token: %w", err)
	}

	return model.Token{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		ExpiresIn:             int(s.jwtService.AccessTTL().Seconds()),
		RefreshTokenExpiresIn: int(s.jwtService.RefreshTTL().Seconds()),
	}, nil
}
