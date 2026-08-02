package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidTokenType = errors.New("invalid token type")

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWT(secret string, accessTTL, refreshTTL time.Duration) *JWT {
	return &JWT{
		secret:     secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (j *JWT) AccessTTL() time.Duration {
	return j.accessTTL
}

func (j *JWT) RefreshTTL() time.Duration {
	return j.refreshTTL
}

func (j *JWT) GenerateAccess(userID uuid.UUID) (string, error) {
	return j.generate(userID, TokenTypeAccess)
}

func (j *JWT) GenerateRefresh(userID uuid.UUID) (string, error) {
	return j.generate(userID, TokenTypeRefresh)
}

func (j *JWT) ParseAccess(tokenString string) (*Claims, error) {
	return j.parse(tokenString, TokenTypeAccess)
}

func (j *JWT) ParseRefresh(tokenString string) (*Claims, error) {
	return j.parse(tokenString, TokenTypeRefresh)
}

func (j *JWT) generate(userID uuid.UUID, tokenType TokenType) (string, error) {
	var ttl time.Duration

	switch tokenType {
	case TokenTypeAccess:
		ttl = j.accessTTL
	case TokenTypeRefresh:
		ttl = j.refreshTTL
	default:
		return "", ErrInvalidTokenType
	}

	claims := &Claims{
		UserID: userID,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secret))
}

func (j *JWT) parse(tokenString string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	if claims.Type != expectedType {
		return nil, fmt.Errorf("unexpected token type: %s", claims.Type)
	}

	return claims, nil
}
