package app

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	apiClient "github.com/liebeSonne/gophkeeper/internal/client/adapter/gophkeeper"
	clientconfig "github.com/liebeSonne/gophkeeper/internal/client/config"
	clientsync "github.com/liebeSonne/gophkeeper/internal/client/service"
	"github.com/liebeSonne/gophkeeper/internal/client/storage"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

var (
	instance *App
	mu       sync.Mutex
)

var ErrNotInitialized = errors.New("client not initialized: run 'gk init' first")

const defaultSyncInterval = 30 * time.Second

type App struct {
	Config  clientconfig.ClientConfig
	Logger  intlogger.Logger
	Storage *storage.Store
	Sync    *clientsync.Service
}

func EnsureInitialized(logLevelOverride string) (*App, error) {
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

	if logLevelOverride != "" && logLevelOverride != clientconfig.DefaultLogLevel {
		cfg.LogLevel = logLevelOverride
	}

	level, ok := logLevelMap[cfg.LogLevel]
	if !ok {
		return nil, fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}

	logger, err := intlogger.New(intlogger.Config{Level: level})
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	store, err := storage.NewStore(cfg.StoragePath, logger)
	if err != nil {
		logger.Warn("failed to open storage, continuing without it", "err", err)
		store = nil
	}

	api, err := apiClient.NewClient(cfg.ServerAddress)
	if err != nil {
		logger.Warn("failed to create API client, sync will be disabled", "err", err)
		api = nil
	}

	var syncService *clientsync.Service
	if store != nil && api != nil {
		syncService = clientsync.NewService(store, api, logger, defaultSyncInterval)
		syncService.Start()
	}

	instance = &App{
		Config:  cfg,
		Logger:  logger,
		Storage: store,
		Sync:    syncService,
	}

	return instance, nil
}

func Get() *App {
	mu.Lock()
	defer mu.Unlock()
	return instance
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

	if instance.Sync != nil {
		instance.Sync.Stop()
	}
	if instance.Storage != nil {
		_ = instance.Storage.Close()
	}
	if instance.Logger != nil {
		_ = instance.Logger.Sync()
	}
	instance = nil
	return nil
}

var logLevelMap = map[string]intlogger.LogLevel{
	clientconfig.LogLevelDebug: intlogger.DebugLevel,
	clientconfig.LogLevelInfo:  intlogger.InfoLevel,
	clientconfig.LogLevelWarn:  intlogger.WarnLevel,
	clientconfig.LogLevelError: intlogger.ErrorLevel,
	clientconfig.LogLevelFatal: intlogger.FatalLevel,
}
