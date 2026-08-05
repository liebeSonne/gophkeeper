package app

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clientconfig "github.com/liebeSonne/gophkeeper/internal/client/config"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

var testMu sync.Mutex

func setupTestEnv(t *testing.T) (configFile, configDir, storagePath string) {
	t.Helper()

	dir := t.TempDir()
	configDir = filepath.Join(dir, ".config", "gophkeeper")
	storagePath = filepath.Join(configDir, "gophkeeper.db")

	require.NoError(t, os.MkdirAll(configDir, 0o700))

	configFile = filepath.Join(configDir, "config.yml")

	origFilePath := clientconfig.GetConfigFilePath
	origDirPath := clientconfig.GetConfigDir

	clientconfig.GetConfigFilePath = func() (string, error) {
		return configFile, nil
	}
	clientconfig.GetConfigDir = func() (string, error) {
		return configDir, nil
	}

	cfg := clientconfig.ClientConfig{
		ServerAddress: "http://localhost:8080",
		StoragePath:   storagePath,
		LogLevel:      "info",
	}
	require.NoError(t, clientconfig.Save(cfg))

	t.Cleanup(func() {
		clientconfig.GetConfigFilePath = origFilePath
		clientconfig.GetConfigDir = origDirPath
	})

	return configFile, configDir, storagePath
}

func overrideConfigPaths(configFile, configDir string) (restore func()) {
	origFilePath := clientconfig.GetConfigFilePath
	origDirPath := clientconfig.GetConfigDir

	clientconfig.GetConfigFilePath = func() (string, error) {
		return configFile, nil
	}
	clientconfig.GetConfigDir = func() (string, error) {
		return configDir, nil
	}

	restore = func() {
		clientconfig.GetConfigFilePath = origFilePath
		clientconfig.GetConfigDir = origDirPath
	}

	return restore
}

func TestApp(t *testing.T) {
	t.Run("NotInitialized", func(t *testing.T) {
		testMu.Lock()
		defer testMu.Unlock()
		Reset()

		dir := t.TempDir()
		fakeConfig := filepath.Join(dir, "nonexistent", "config.yml")
		fakeDir := filepath.Dir(fakeConfig)

		restore := overrideConfigPaths(fakeConfig, fakeDir)
		defer restore()

		_, err := EnsureInitialized("")
		assert.ErrorIs(t, err, ErrNotInitialized)
	})

	t.Run("SingleInstance", func(t *testing.T) {
		testMu.Lock()
		defer testMu.Unlock()
		Reset()

		configFile, _, _ := setupTestEnv(t)
		restore := overrideConfigPaths(configFile, filepath.Dir(configFile))
		defer restore()

		_, err := EnsureInitialized("")
		require.NoError(t, err)

		app1, err := EnsureInitialized("")
		require.NoError(t, err)
		assert.Same(t, app1, instance)
	})

	t.Run("Reset", func(t *testing.T) {
		testMu.Lock()
		defer testMu.Unlock()
		Reset()

		configFile, _, _ := setupTestEnv(t)
		restore := overrideConfigPaths(configFile, filepath.Dir(configFile))
		defer restore()

		_, err := EnsureInitialized("")
		require.NoError(t, err)
		require.NotNil(t, instance)

		Reset()
		assert.Nil(t, instance)
	})

	t.Run("Close_NotInitialized", func(t *testing.T) {
		testMu.Lock()
		defer testMu.Unlock()
		Reset()

		assert.NoError(t, Close())
	})

	t.Run("Close_AfterInit", func(t *testing.T) {
		testMu.Lock()
		defer testMu.Unlock()
		Reset()

		configFile, _, _ := setupTestEnv(t)
		restore := overrideConfigPaths(configFile, filepath.Dir(configFile))
		defer restore()

		_, err := EnsureInitialized("")
		require.NoError(t, err)

		assert.NoError(t, Close())
		assert.Nil(t, instance)
	})

	t.Run("LogLevels", func(t *testing.T) {
		expected := map[string]intlogger.LogLevel{
			"debug": intlogger.DebugLevel,
			"info":  intlogger.InfoLevel,
			"warn":  intlogger.WarnLevel,
			"error": intlogger.ErrorLevel,
			"fatal": intlogger.FatalLevel,
		}

		for level, expectedValue := range expected {
			got, ok := logLevelMap[level]
			assert.True(t, ok, "log level %s should be mapped", level)
			assert.Equal(t, expectedValue, got, "log level %s should map to correct value", level)
		}
	})
}
