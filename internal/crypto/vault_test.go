// nolint:revive // package name matches existing codebase conventions
package crypto

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

var testVaultClient = &http.Client{
	Timeout: 10 * time.Second,
}

func TestNewVaultEncryptor(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	keyB64 := base64.StdEncoding.EncodeToString(key)

	testCases := []struct {
		name              string
		keyPath           string
		handler           func(w http.ResponseWriter, r *http.Request)
		expectError       bool
		expectErrContains string
		checkFunc         func(t *testing.T, encryptor *VaultEncryptor, calledPath string, receivedToken string)
	}{
		{
			name:    "success with encryption",
			keyPath: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"data":{"data":{"key":"` + keyB64 + `"}}}`))
				assert.NoError(t, err)
			},
			checkFunc: func(t *testing.T, encryptor *VaultEncryptor, _ string, receivedToken string) {
				assert.Equal(t, "test-token", receivedToken)

				plaintext := []byte("test data")
				ciphertext, err := encryptor.Encrypt(plaintext)
				require.NoError(t, err)

				decrypted, err := encryptor.Decrypt(ciphertext)
				require.NoError(t, err)
				assert.Equal(t, plaintext, decrypted)
			},
		},
		{
			name:    "default key path",
			keyPath: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"data":{"data":{"key":"` + keyB64 + `"}}}`))
				assert.NoError(t, err)
			},
			checkFunc: func(t *testing.T, _ *VaultEncryptor, calledPath string, _ string) {
				assert.Contains(t, calledPath, "/v1/secret/data/gophkeeper/encryption")
			},
		},
		{
			name:    "custom key path",
			keyPath: "custom/path",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"data":{"data":{"key":"` + keyB64 + `"}}}`))
				assert.NoError(t, err)
			},
			checkFunc: func(t *testing.T, _ *VaultEncryptor, calledPath string, _ string) {
				assert.Contains(t, calledPath, "/v1/custom/path")
			},
		},
		{
			name:              "vault error",
			keyPath:           "",
			expectError:       true,
			expectErrContains: "vault returned status 404",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"errors":["key not found"]}`))
				assert.NoError(t, err)
			},
		},
		{
			name:              "invalid key encoding",
			keyPath:           "",
			expectError:       true,
			expectErrContains: "decode key",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"data":{"data":{"key":"invalid-base64!!"}}}`))
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var calledPath string
			var receivedToken string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calledPath = r.URL.Path
				receivedToken = r.Header.Get("X-Vault-Token")
				tc.handler(w, r)
			}))
			defer server.Close()

			l := intlogger.NewMockLogger(t)
			encryptor, err := NewVaultEncryptor(context.Background(), server.URL, "test-token", tc.keyPath, testVaultClient, l)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, encryptor)

			if tc.checkFunc != nil {
				tc.checkFunc(t, encryptor, calledPath, receivedToken)
			}
		})
	}
}

func TestLoadKeyFromVault(t *testing.T) {
	testCases := []struct {
		name              string
		address           string
		expectError       bool
		expectErrContains string
	}{
		{
			name:              "invalid address format",
			address:           "://invalid",
			expectError:       true,
			expectErrContains: "parse vault address",
		},
		{
			name:              "unsupported scheme",
			address:           "ftp://vault:8200",
			expectError:       true,
			expectErrContains: "unsupported scheme",
		},
		{
			name:              "http connection error",
			address:           "http://nonexistent:99999",
			expectError:       true,
			expectErrContains: "http request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := intlogger.NewMockLogger(t)
			_, err := loadKeyFromVault(context.Background(), tc.address, "token", "path", testVaultClient, l)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectErrContains)
		})
	}
}
