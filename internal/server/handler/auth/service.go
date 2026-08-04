package auth

import "github.com/google/uuid"

type Service interface {
	GetUserIDFromToken(tokenString string) (uuid.UUID, error)
}
