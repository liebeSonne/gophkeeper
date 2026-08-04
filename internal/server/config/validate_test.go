package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	jwtSecret1 := "testsecret"
	encryptionKey := "YWJjZGVmZzEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIz"

	testCases := []struct {
		name    string
		cfg     ServerConfig
		wantErr error
	}{
		{
			name:    "valid config",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1, EncryptionKey: encryptionKey},
			wantErr: nil,
		},
		{
			name:    "valid wildcard address",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: "0.0.0.0:8080", JWTSecret: jwtSecret1, EncryptionKey: encryptionKey},
			wantErr: nil,
		},
		{
			name:    "valid empty host",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: ":8080", JWTSecret: jwtSecret1, EncryptionKey: encryptionKey},
			wantErr: nil,
		},
		{
			name:    "valid https config",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: "0.0.0.0:443", EnableHTTPS: true, TLSCert: "/cert.pem", TLSKey: "/key.pem", JWTSecret: jwtSecret1, EncryptionKey: encryptionKey},
			wantErr: nil,
		},
		{
			name:    "all log levels valid",
			cfg:     ServerConfig{LogLevel: LogLevelDebug, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1, EncryptionKey: encryptionKey},
			wantErr: nil,
		},
		{
			name:    "invalid log level",
			cfg:     ServerConfig{LogLevel: "invalid", ServerAddress: DefaultServerAddress},
			wantErr: errInvalidLogLevel,
		},
		{
			name:    "invalid address no port",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: "not-an-address"},
			wantErr: errInvalidServerAddress,
		},
		{
			name:    "invalid host",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: "invalid-host:8080"},
			wantErr: errInvalidServerAddressHost,
		},
		{
			name:    "invalid port zero",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: "0.0.0.0:0"},
			wantErr: errInvalidServerAddressPort,
		},
		{
			name:    "invalid port too high",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: "0.0.0.0:70000"},
			wantErr: errInvalidServerAddressPort,
		},
		{
			name:    "https without cert",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, EnableHTTPS: true, TLSKey: "/key.pem"},
			wantErr: errEmptyTLSCert,
		},
		{
			name:    "https without key",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, EnableHTTPS: true, TLSCert: "/cert.pem"},
			wantErr: errEmptyTLSKey,
		},
		{
			name:    "valid jwt secret",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1, EncryptionKey: encryptionKey},
			wantErr: nil,
		},
		{
			name:    "empty jwt secret",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: ""},
			wantErr: errEmptyJWTSecret,
		},
		{
			name:    "valid vault config",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1, EnableVault: true, VaultAddress: "http://127.0.0.1:8200", VaultToken: "dev-token"},
			wantErr: nil,
		},
		{
			name:    "vault without address",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1, EnableVault: true, VaultToken: "dev-token"},
			wantErr: errEmptyVaultAddress,
		},
		{
			name:    "vault without token",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1, EnableVault: true, VaultAddress: "http://127.0.0.1:8200"},
			wantErr: errEmptyVaultToken,
		},
		{
			name:    "no encryption key set",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1},
			wantErr: errEncryptionKeyNotSet,
		},
		{
			name:    "both vault and encryption key",
			cfg:     ServerConfig{LogLevel: LogLevelInfo, ServerAddress: DefaultServerAddress, JWTSecret: jwtSecret1, EnableVault: true, VaultAddress: "http://127.0.0.1:8200", VaultToken: "dev-token", EncryptionKey: encryptionKey},
			wantErr: errBothVaultAndEncryptionKey,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.cfg)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
