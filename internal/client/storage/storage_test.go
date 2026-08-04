package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/liebeSonne/gophkeeper/internal/client/model"
	"github.com/liebeSonne/gophkeeper/internal/logger"
)

func newTestStorage(t *testing.T) (ts *TokenStorage, cleanup func()) {
	t.Helper()

	dir := t.TempDir()
	storagePath := filepath.Join(dir, "test.db")
	l := logger.NewMockLogger(t)
	l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

	var err error
	ts, err = NewTokenStorage(storagePath, l)
	require.NoError(t, err)

	cleanup = func() {
		ts.Close()
		os.Remove(storagePath)
	}

	return ts, cleanup
}

func TestSaveAndGetTokens(t *testing.T) {
	ts, cleanup := newTestStorage(t)
	defer cleanup()

	tokens := model.Token{
		AccessToken:          "test-access-token",
		RefreshToken:         "test-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	require.NoError(t, ts.SaveTokens(tokens))

	got, err := ts.GetTokens()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tokens.AccessToken, got.AccessToken)
	assert.Equal(t, tokens.RefreshToken, got.RefreshToken)
}

func TestGetTokensEmpty(t *testing.T) {
	ts, cleanup := newTestStorage(t)
	defer cleanup()

	got, err := ts.GetTokens()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestHasValidAccessToken(t *testing.T) {
	ts, cleanup := newTestStorage(t)
	defer cleanup()

	assert.False(t, ts.HasValidAccessToken())

	tokens := model.Token{
		AccessToken:          "test-access-token",
		RefreshToken:         "test-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	require.NoError(t, ts.SaveTokens(tokens))
	assert.True(t, ts.HasValidAccessToken())
}

func TestHasExpiredAccessToken(t *testing.T) {
	ts, cleanup := newTestStorage(t)
	defer cleanup()

	tokens := model.Token{
		AccessToken:          "test-access-token",
		RefreshToken:         "test-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(-time.Hour),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	require.NoError(t, ts.SaveTokens(tokens))
	assert.False(t, ts.HasValidAccessToken())
}

func TestClearTokens(t *testing.T) {
	ts, cleanup := newTestStorage(t)
	defer cleanup()

	tokens := model.Token{
		AccessToken:          "test-access-token",
		RefreshToken:         "test-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	require.NoError(t, ts.SaveTokens(tokens))
	require.NoError(t, ts.ClearTokens())

	got, err := ts.GetTokens()
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestUpdateTokens(t *testing.T) {
	ts, cleanup := newTestStorage(t)
	defer cleanup()

	tokens := model.Token{
		AccessToken:          "test-access-token",
		RefreshToken:         "test-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	require.NoError(t, ts.SaveTokens(tokens))

	newTokens := model.Token{
		AccessToken:          "new-access-token",
		RefreshToken:         "new-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		RefreshExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	require.NoError(t, ts.SaveTokens(newTokens))

	got, err := ts.GetTokens()
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, newTokens.AccessToken, got.AccessToken)
}
