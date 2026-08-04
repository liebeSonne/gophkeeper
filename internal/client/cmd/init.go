package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/liebeSonne/gophkeeper/internal/client/app"
	"github.com/liebeSonne/gophkeeper/internal/client/config"
	"github.com/liebeSonne/gophkeeper/internal/client/storage"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

var (
	initServerAddress string
	initStoragePath   string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize client configuration and storage",
	Long: `Initialize the GophKeeper client by creating a configuration file
and SQLite storage database. This command creates the config directory
at ~/.config/gophkeeper/ if it doesn't exist.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		serverAddr := initServerAddress
		if serverAddr == "" {
			serverAddr = config.DefaultServerAddress
		}

		storagePath := initStoragePath
		if storagePath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("get home dir: %w", err)
			}
			storagePath = config.GetDefaultStoragePath(homeDir)
		}

		cfg := config.ClientConfig{
			ServerAddress: serverAddr,
			StoragePath:   storagePath,
			LogLevel:      "info",
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		logger, err := initLogger(cfg)
		if err != nil {
			return fmt.Errorf("initialize logger: %w", err)
		}

		_, err = storage.NewTokenStorage(storagePath, logger)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}

		app.Reset()

		fmt.Printf("Configuration created successfully.\n")
		fmt.Printf("  Config: %s\n", func() string {
			path, _ := config.GetConfigFilePath()
			return path
		}())
		fmt.Printf("  Storage: %s\n", storagePath)
		fmt.Printf("  Server: %s\n", serverAddr)

		return nil
	},
}

var logLevelMap = map[string]intlogger.LogLevel{
	config.LogLevelDebug: intlogger.DebugLevel,
	config.LogLevelInfo:  intlogger.InfoLevel,
	config.LogLevelWarn:  intlogger.WarnLevel,
	config.LogLevelError: intlogger.ErrorLevel,
	config.LogLevelFatal: intlogger.FatalLevel,
}

func initLogger(cfg config.ClientConfig) (intlogger.Logger, error) {
	level, ok := logLevelMap[cfg.LogLevel]
	if !ok {
		return nil, fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}
	return intlogger.New(intlogger.Config{Level: level})
}
