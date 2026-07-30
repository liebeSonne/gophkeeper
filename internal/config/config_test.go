package config

import (
	"fmt"
	"os"
	"testing"

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
			name:   "defaults",
			args:   []string{"gophkeeper-server"},
			setEnv: nil,
			want: ServerConfig{
				LogLevel:      DefaultLogLevel,
				ServerAddress: DefaultServerAddress,
				EnableHTTPS:   DefaultEnableHTTPS,
				DatabaseURI:   DefaultDatabaseURI,
			},
		},
		{
			name: "env override",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvLogLevel:      LogLevelDebug,
				EnvServerAddress: "127.0.0.1:9090",
			},
			want: ServerConfig{
				LogLevel:      LogLevelDebug,
				ServerAddress: "127.0.0.1:9090",
				DatabaseURI:   DefaultDatabaseURI,
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
			},
			want: ServerConfig{
				LogLevel:      LogLevelError,
				ServerAddress: "0.0.0.0:3000",
				DatabaseURI:   DefaultDatabaseURI,
			},
		},
		{
			name: "env enable https",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvEnableHTTPS: "true",
				EnvTLSCert:     "/path/cert.pem",
				EnvTLSKey:      "/path/key.pem",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				EnableHTTPS:   true,
				TLSCert:       "/path/cert.pem",
				TLSKey:        "/path/key.pem",
				DatabaseURI:   DefaultDatabaseURI,
			},
		},
		{
			name: "env database uri",
			args: []string{"gophkeeper-server"},
			setEnv: map[string]string{
				EnvDatabaseURI: "postgres://myuser:mypass@db.example.com:5432/mydb?sslmode=require",
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				DatabaseURI:   "postgres://myuser:mypass@db.example.com:5432/mydb?sslmode=require",
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
			},
			want: ServerConfig{
				LogLevel:      LogLevelInfo,
				ServerAddress: DefaultServerAddress,
				DatabaseURI:   "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
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
		})
	}
}
