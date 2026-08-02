package auth

import (
	"context"

	"github.com/google/uuid"
)

type userIDContextKey struct{}

var userIDKey = userIDContextKey{}

func CreateTokenContext(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if ok {
		return userID, true
	}
	return uuid.Nil, false
}
