package main

import (
	"fmt"

	internallogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/server/config"
)

var logLevelMap = map[string]internallogger.LogLevel{
	config.LogLevelDebug: internallogger.DebugLevel,
	config.LogLevelInfo:  internallogger.InfoLevel,
	config.LogLevelWarn:  internallogger.WarnLevel,
	config.LogLevelError: internallogger.ErrorLevel,
	config.LogLevelFatal: internallogger.FatalLevel,
}

func initLogger(cfg config.ServerConfig) (internallogger.Logger, error) {
	level, ok := logLevelMap[cfg.LogLevel]
	if !ok {
		return nil, fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}
	return internallogger.New(internallogger.Config{Level: level})
}
