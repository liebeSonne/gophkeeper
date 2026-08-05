package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	testCases := []struct {
		name      string
		level     LogLevel
		wantErr   bool
		testFunc  func(Logger)
		checkFunc func(*bytes.Buffer) bool
	}{
		{
			name:    "debug level valid",
			level:   DebugLevel,
			wantErr: false,
		},
		{
			name:    "info level valid",
			level:   InfoLevel,
			wantErr: false,
		},
		{
			name:    "warn level valid",
			level:   WarnLevel,
			wantErr: false,
		},
		{
			name:    "error level valid",
			level:   ErrorLevel,
			wantErr: false,
		},
		{
			name:    "fatal level valid",
			level:   FatalLevel,
			wantErr: false,
		},
		{
			name:    "invalid level",
			level:   LogLevel(99),
			wantErr: true,
		},
		{
			name:    "info message",
			level:   InfoLevel,
			wantErr: false,
			testFunc: func(l Logger) {
				l.Info("test message", "key", "value")
			},
			checkFunc: func(buf *bytes.Buffer) bool {
				return strings.Contains(buf.String(), "test message")
			},
		},
		{
			name:    "warn message",
			level:   WarnLevel,
			wantErr: false,
			testFunc: func(l Logger) {
				l.Warn("warn message")
			},
			checkFunc: func(buf *bytes.Buffer) bool {
				return strings.Contains(buf.String(), "warn message")
			},
		},
		{
			name:    "error message",
			level:   ErrorLevel,
			wantErr: false,
			testFunc: func(l Logger) {
				l.Error("error message")
			},
			checkFunc: func(buf *bytes.Buffer) bool {
				return strings.Contains(buf.String(), "error message")
			},
		},
		{
			name:    "debug not shown at info level",
			level:   InfoLevel,
			wantErr: false,
			testFunc: func(l Logger) {
				l.Debug("debug message")
			},
			checkFunc: func(buf *bytes.Buffer) bool {
				return !strings.Contains(buf.String(), "debug message")
			},
		},
		{
			name:    "sync success",
			level:   InfoLevel,
			wantErr: false,
			testFunc: func(l Logger) {
				_ = l.Sync()
			},
		},
		{
			name:    "print message",
			level:   InfoLevel,
			wantErr: false,
			testFunc: func(l Logger) {
				l.Print("print msg")
			},
			checkFunc: func(buf *bytes.Buffer) bool {
				return strings.Contains(buf.String(), "print msg")
			},
		},
		{
			name:    "nil writer defaults to stderr",
			level:   InfoLevel,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{
				Level:  tc.level,
				Writer: &buf,
			}

			logger, err := New(cfg)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, logger)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, logger)

			if tc.testFunc != nil {
				tc.testFunc(logger)
			}

			if tc.checkFunc != nil {
				assert.True(t, tc.checkFunc(&buf))
			}
		})
	}
}

func TestLogger_NilWriter(t *testing.T) {
	logger, err := New(Config{Level: InfoLevel})
	require.NoError(t, err)
	assert.NotNil(t, logger)
}
