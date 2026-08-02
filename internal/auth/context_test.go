package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateAndGetUserIDFromContext(t *testing.T) {
	userID1 := uuid.New()

	testCases := []struct {
		name     string
		userID   uuid.UUID
		expected uuid.UUID
		hasValue bool
	}{
		{
			name:     "valid user ID",
			userID:   userID1,
			expected: userID1,
			hasValue: true,
		},
		{
			name:     "nil user ID",
			userID:   uuid.Nil,
			expected: uuid.Nil,
			hasValue: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			ctx = CreateTokenContext(ctx, tc.userID)

			gotUserID, ok := GetUserIDFromContext(ctx)
			assert.Equal(t, tc.hasValue, ok)
			assert.Equal(t, tc.expected, gotUserID)
		})
	}
}

func TestGetUserIDFromContext_NotFound(t *testing.T) {
	ctx := t.Context()

	gotUserID, ok := GetUserIDFromContext(ctx)
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, gotUserID)
}

func TestGetUserIDFromContext_WrongType(t *testing.T) {
	ctx := t.Context()
	ctx = context.WithValue(ctx, userIDKey, "not a uuid")

	gotUserID, ok := GetUserIDFromContext(ctx)
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, gotUserID)
}
