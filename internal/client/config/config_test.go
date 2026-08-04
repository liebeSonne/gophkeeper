package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()

	originalGetConfigDir := GetConfigDir
	originalGetConfigFilePath := GetConfigFilePath
	originalGetDefaultStoragePath := getDefaultStoragePath

	tmpConfigFile := filepath.Join(dir, "config.yml")
	tmpStorageFile := filepath.Join(dir, "gophkeeper.db")

	GetConfigDir = func() (string, error) {
		return dir, nil
	}
	GetConfigFilePath = func() (string, error) {
		return tmpConfigFile, nil
	}
	getDefaultStoragePath = func() string {
		return tmpStorageFile
	}

	defer func() {
		GetConfigDir = originalGetConfigDir
		GetConfigFilePath = originalGetConfigFilePath
		getDefaultStoragePath = originalGetDefaultStoragePath
	}()

	cfg := ClientConfig{
		ServerAddress: "http://test:9090",
		StoragePath:   tmpStorageFile,
		LogLevel:      "debug",
	}

	require.NoError(t, Save(cfg))

	_, err := os.Stat(tmpConfigFile)
	require.NoError(t, err, "config file was not created")

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, cfg.ServerAddress, loaded.ServerAddress)
	assert.Equal(t, cfg.StoragePath, loaded.StoragePath)
	assert.Equal(t, cfg.LogLevel, loaded.LogLevel)
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()

	originalGetConfigFilePath := GetConfigFilePath
	GetConfigFilePath = func() (string, error) {
		return filepath.Join(dir, "nonexistent.yml"), nil
	}
	defer func() {
		GetConfigFilePath = originalGetConfigFilePath
	}()

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, DefaultServerAddress, cfg.ServerAddress)
	assert.Equal(t, DefaultLogLevel, cfg.LogLevel)
}
