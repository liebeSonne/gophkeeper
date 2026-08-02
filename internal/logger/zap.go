package logger

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	levelKey   = "level"
	messageKey = "message"
	timeKey    = "time"
)

var zapLogLevelMap = map[LogLevel]zapcore.Level{
	DebugLevel: zapcore.DebugLevel,
	InfoLevel:  zapcore.InfoLevel,
	WarnLevel:  zapcore.WarnLevel,
	ErrorLevel: zapcore.ErrorLevel,
	FatalLevel: zapcore.FatalLevel,
}

func New(cfg Config) (Logger, error) {
	if cfg.Writer == nil {
		cfg.Writer = os.Stderr
	}

	level, ok := zapLogLevelMap[cfg.Level]
	if !ok {
		return nil, fmt.Errorf("invalid log level: %v", cfg.Level)
	}

	zapLogger, err := initZapLogger(level, cfg.Writer)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	sugarLogger := zapLogger.Sugar()

	return &zapLoggerImpl{
		logger: sugarLogger,
	}, nil
}

func initZapLogger(level zapcore.Level, w io.Writer) (*zap.Logger, error) {
	writeSyncer := zapcore.AddSync(w)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = levelKey
	encoderConfig.MessageKey = messageKey
	encoderConfig.TimeKey = timeKey
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(encoder, writeSyncer, level)

	logger := zap.New(core)

	return logger, nil
}

type zapLoggerImpl struct {
	logger *zap.SugaredLogger
}

func (l *zapLoggerImpl) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debugw(msg, keysAndValues...)
}
func (l *zapLoggerImpl) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Infow(msg, keysAndValues...)
}
func (l *zapLoggerImpl) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warnw(msg, keysAndValues...)
}
func (l *zapLoggerImpl) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Errorw(msg, keysAndValues...)
}
func (l *zapLoggerImpl) Fatal(msg string, keysAndValues ...interface{}) {
	l.logger.Fatalw(msg, keysAndValues...)
}
func (l *zapLoggerImpl) Sync() error {
	return l.logger.Sync()
}
func (l *zapLoggerImpl) Print(v ...interface{}) {
	l.logger.Infow(fmt.Sprint(v...))
}
