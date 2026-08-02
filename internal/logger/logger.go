package logger

import "io"

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	Sync() error
	Print(v ...interface{})
}

type Config struct {
	Level  LogLevel
	Writer io.Writer
}
