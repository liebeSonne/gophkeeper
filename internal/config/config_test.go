package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEnvKey(envPrefix, name string) string {
	if envPrefix == "" {
		return name
	}
	return envPrefix + "_" + name
}

// nolint: gosec
func TestLoad(t *testing.T) {
	testCases := []struct {
		name      string
		args      []string
		setEnv    map[string]string
		want      ServerConfig
		isWantErr bool
	}{
		{
			name: "defaults",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvJWTSecret: "testsecret",
			},
			want: ServerConfig{
				LogLevel:      DefaultLogLevel,
				ServerAddress: DefaultServerAddress,
				EnableHTTPS:   DefaultEnableHTTPS,
				DatabaseURI:   DefaultDatabaseURI,
				JWTSecret:     "testsecret",
				JWTAccessTTL:  DefaultJWTAccessTTL,
				JWTRefreshTTL: DefaultJWTRefreshTTL,
			},
		},
		{
			name: "env override",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvLogLevel:      LogLevelDebug,
				EnvServerAddress: "127.0.0.1:9090",
				EnvJWTSecret:     "testsecret",
			},
			want: ServerConfig{
				LogLevel:      LogLevelDebug,
				ServerAddress: "127.0.0.1:9090",
				DatabaseURI:   DefaultDatabaseURI,
				JWTSecret:     "testsecret",
				JWTAccessTTL:  DefaultJWTAccessTTL,
				JWTRefreshTTL: DefaultJWTRefreshTTL,
			},
		},
		{
			name: "flags override env",
			args: []string{
				"gophkeeper-server",
				fmt.Sprintf("--%s", FlagLogLevel), LogLevelError,
				fmt.Sprintf("--%s", FlagServerAddress), "0.0.0.0:3000",
			},
			setEnv: map[string]string{
				EnvLogLevel:      LogLevelDebug,
				EnvServerAddress: "127.0.0.1:9090",
				EnvJWTSecret:     "testsecret",
			},
			want: ServerConfig{
				LogLevel:      LogLevelError,
				ServerAddress: "0.0.0.0:3000",
				DatabaseURI:   DefaultDatabaseURI,
				JWTSecret:     "testsecret",
				JWTAccessTTL:  DefaultJWTAccessTTL,
				JWTRefreshTTL: DefaultJWTRefreshTTL,
			},
		},
		{
			name: "env enable https",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvEnableHTTPS: "true",
				EnvTLSCert:     "/path/cert.pem",
				EnvTLSKey:      "/path/key.pem",
				EnvJWTSecret:   "testsecret",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				EnableHTTPS:   true,
				TLSCert:       "/path/cert.pem",
				TLSKey:        "/path/key.pem",
				DatabaseURI:   DefaultDatabaseURI,
				JWTSecret:     "testsecret",
				JWTAccessTTL:  DefaultJWTAccessTTL,
				JWTRefreshTTL: DefaultJWTRefreshTTL,
			},
		},
		{
			name: "env database uri",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvDatabaseURI: "postgres://myuser:mypass@db.example.com:5432/mydb?sslmode=require",
				EnvJWTSecret:   "testsecret",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				DatabaseURI:   "postgres://myuser:mypass@db.example.com:5432/mydb?sslmode=require",
				JWTSecret:     "testsecret",
				JWTAccessTTL:  DefaultJWTAccessTTL,
				JWTRefreshTTL: DefaultJWTRefreshTTL,
			},
		},
		{
			name: "flag database uri",
			args: []string{
				"gophkeeper-server",
				fmt.Sprintf("--%s", FlagDatabaseURI), "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
			},
			setEnv: map[string]string{
				EnvDatabaseURI: "postgres://envuser:envpass@envhost:5432/envdb?sslmode=disable",
				EnvJWTSecret:   "testsecret",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				DatabaseURI:   "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
				JWTSecret:     "testsecret",
				JWTAccessTTL:  DefaultJWTAccessTTL,
				JWTRefreshTTL: DefaultJWTRefreshTTL,
			},
		},
		{
			name: "defaults with jwt",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvJWTSecret: "mysecret",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				DatabaseURI:   DefaultDatabaseURI,
				JWTSecret:     "mysecret",
				JWTAccessTTL:  DefaultJWTAccessTTL,
				JWTRefreshTTL: DefaultJWTRefreshTTL,
			},
		},
		{
			name: "env jwt params",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvJWTSecret:     "envsecret",
				EnvJWTAccessTTL:  "30m",
				EnvJWTRefreshTTL: "48h",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				DatabaseURI:   DefaultDatabaseURI,
				JWTSecret:     "envsecret",
				JWTAccessTTL:  30 * time.Minute,
				JWTRefreshTTL: 48 * time.Hour,
			},
		},
		{
			name: "flag jwt override env",
			args: []string{
				"gophkeeper-server",
				fmt.Sprintf("--%s", FlagJWTSecret), "flagsecret",
				fmt.Sprintf("--%s", FlagJWTAccessTTL), "10m",
				fmt.Sprintf("--%s", FlagJWTRefreshTTL), "12h",
			},
			setEnv: map[string]string{
				EnvJWTSecret:     "envsecret",
				EnvJWTAccessTTL:  "30m",
				EnvJWTRefreshTTL: "48h",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				DatabaseURI:   DefaultDatabaseURI,
				JWTSecret:     "flagsecret",
				JWTAccessTTL:  10 * time.Minute,
				JWTRefreshTTL: 12 * time.Hour,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := os.Args
			os.Args = tc.args
			defer func() { os.Args = args }()

			for key, value := range tc.setEnv {
				t.Setenv(makeEnvKey(DefaultEnvPrefix, key), value)
			}

			cfg, err := Load(DefaultEnvPrefix)

			if tc.isWantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want.LogLevel, cfg.LogLevel)
			assert.Equal(t, tc.want.ServerAddress, cfg.ServerAddress)
			assert.Equal(t, tc.want.EnableHTTPS, cfg.EnableHTTPS)
			assert.Equal(t, tc.want.TLSCert, cfg.TLSCert)
			assert.Equal(t, tc.want.TLSKey, cfg.TLSKey)
			assert.Equal(t, tc.want.DatabaseURI, cfg.DatabaseURI)
			assert.Equal(t, tc.want.JWTSecret, cfg.JWTSecret)
			assert.Equal(t, tc.want.JWTAccessTTL, cfg.JWTAccessTTL)
			assert.Equal(t, tc.want.JWTRefreshTTL, cfg.JWTRefreshTTL)
		})
	}
}
