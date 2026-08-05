package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRefreshTokenIsActive(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		token    RefreshToken
		expected bool
	}{
		{
			name: "active token",
			token: RefreshToken{
				ExpiresAt: now.Add(24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "revoked token",
			token: RefreshToken{
				ExpiresAt: now.Add(24 * time.Hour),
				RevokedAt: &now,
			},
			expected: false,
		},
		{
			name: "expired token",
			token: RefreshToken{
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired and revoked token",
			token: RefreshToken{
				ExpiresAt: now.Add(-1 * time.Hour),
				RevokedAt: &now,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.token.IsActive())
		})
	}
}
