package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultConfigName = "config.yml"
	defaultConfigDir  = "gophkeeper"
)

const (
	FieldServerAddress = "server_address"
	FieldStoragePath   = "storage_path"
	FieldLogLevel      = "log_level"
)

const (
	FlagServerAddress = "server-address"
	FlagStoragePath   = "storage-path"
	FlagLogLevel      = "log-level"
)

const (
	DefaultServerAddress = "http://localhost:8080"
	DefaultLogLevel      = LogLevelInfo
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)

type ClientConfig struct {
	ServerAddress string
	StoragePath   string
	LogLevel      string
}

var (
	getDefaultStoragePath = func() string {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", "gophkeeper.db")
		}
		return GetDefaultStoragePath(homeDir)
	}

	GetConfigFilePath = func() (string, error) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir := filepath.Join(homeDir, ".config", defaultConfigDir)
		return filepath.Join(configDir, defaultConfigName), nil
	}

	GetConfigDir = func() (string, error) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".config", defaultConfigDir), nil
	}
)

func GetDefaultStoragePath(homeDir string) string {
	configDir := filepath.Join(homeDir, ".config", defaultConfigDir)
	return filepath.Join(configDir, "gophkeeper.db")
}

func Load() (ClientConfig, error) {
	v := viper.New()

	v.SetDefault(FieldServerAddress, DefaultServerAddress)
	v.SetDefault(FieldStoragePath, getDefaultStoragePath())
	v.SetDefault(FieldLogLevel, DefaultLogLevel)

	configFile, err := GetConfigFilePath()
	if err != nil {
		return ClientConfig{}, err
	}

	_, err = os.Stat(configFile)
	if err == nil {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return ClientConfig{}, err
		}
	}

	p := pflag.NewFlagSet("client", pflag.ContinueOnError)
	flagServerAddr := p.String(FlagServerAddress, DefaultServerAddress, "server address (e.g. http://localhost:8080)")
	flagStoragePath := p.String(FlagStoragePath, getDefaultStoragePath(), "path to SQLite storage file")
	flagLogLevel := p.String(FlagLogLevel, DefaultLogLevel, "log level (debug, info, warn, error)")

	if err := p.Parse(os.Args[1:]); err != nil {
		return ClientConfig{}, err
	}

	if p.Changed(FlagServerAddress) {
		v.Set(FieldServerAddress, *flagServerAddr)
	}
	if p.Changed(FlagStoragePath) {
		v.Set(FieldStoragePath, *flagStoragePath)
	}
	if p.Changed(FlagLogLevel) {
		v.Set(FieldLogLevel, *flagLogLevel)
	}

	return ClientConfig{
		ServerAddress: v.GetString(FieldServerAddress),
		StoragePath:   v.GetString(FieldStoragePath),
		LogLevel:      v.GetString(FieldLogLevel),
	}, nil
}

func Save(cfg ClientConfig) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	configFile, err := GetConfigFilePath()
	if err != nil {
		return err
	}

	v := viper.New()
	v.Set(FieldServerAddress, cfg.ServerAddress)
	v.Set(FieldStoragePath, cfg.StoragePath)
	v.Set(FieldLogLevel, cfg.LogLevel)

	v.SetConfigType("yaml")

	return v.WriteConfigAs(configFile)
}
