package app

import (
	"errors"
	"fmt"
	"os"
	"sync"

	clientconfig "github.com/liebeSonne/gophkeeper/internal/client/config"
	"github.com/liebeSonne/gophkeeper/internal/client/storage"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

var (
	instance *App
	mu       sync.Mutex
)

var ErrNotInitialized = errors.New("client not initialized: run 'gk init' first")

type App struct {
	Config  clientconfig.ClientConfig
	Logger  intlogger.Logger
	Storage *storage.TokenStorage
}

func EnsureInitialized() (*App, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	configFile, err := clientconfig.GetConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("get config path: %w", err)
	}

	if _, statErr := os.Stat(configFile); statErr != nil {
		return nil, ErrNotInitialized
	}

	cfg, err := clientconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	level, ok := logLevelMap[cfg.LogLevel]
	if !ok {
		return nil, fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}

	logger, err := intlogger.New(intlogger.Config{Level: level})
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	store, err := storage.NewTokenStorage(cfg.StoragePath, logger)
	if err != nil {
		logger.Warn("failed to open storage, continuing without it", "err", err)
		store = nil
	}

	instance = &App{
		Config:  cfg,
		Logger:  logger,
		Storage: store,
	}

	return instance, nil
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		if instance.Storage != nil {
			_ = instance.Storage.Close()
		}
		instance = nil
	}
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		return nil
	}

	var closeErr error
	if instance.Storage != nil {
		closeErr = instance.Storage.Close()
	}
	if instance.Logger != nil {
		_ = instance.Logger.Sync()
	}
	instance = nil
	return closeErr
}

var logLevelMap = map[string]intlogger.LogLevel{
	clientconfig.LogLevelDebug: intlogger.DebugLevel,
	clientconfig.LogLevelInfo:  intlogger.InfoLevel,
	clientconfig.LogLevelWarn:  intlogger.WarnLevel,
	clientconfig.LogLevelError: intlogger.ErrorLevel,
	clientconfig.LogLevelFatal: intlogger.FatalLevel,
}
