package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

func TestNewVaultEncryptor(t *testing.T) {
	validKey := make([]byte, 32)
	validKey[0] = 1
	validKeyB64 := base64.StdEncoding.EncodeToString(validKey)

	testCases := []struct {
		name        string
		serverFn    func(t *testing.T) *httptest.Server
		address     string
		token       string
		keyPath     string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful key load",
			serverFn: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "dev-token", r.Header.Get("X-Vault-Token"))
					assert.Equal(t, "GET", r.Method)

					w.Header().Set("Content-Type", "application/json")
					resp := vaultResponse{
						Data: vaultKVData{
							Data: vaultSecretData{
								Key: validKeyB64,
							},
						},
					}
					err := json.NewEncoder(w).Encode(resp)
					assert.NoError(t, err)
				}))
			},
			address: "",
			token:   "dev-token",
			keyPath: "secret/data/gophkeeper/encryption",
			wantErr: false,
		},
		{
			name: "successful with custom key path",
			serverFn: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "custom/path/key")

					w.Header().Set("Content-Type", "application/json")
					resp := vaultResponse{
						Data: vaultKVData{
							Data: vaultSecretData{
								Key: validKeyB64,
							},
						},
					}
					err := json.NewEncoder(w).Encode(resp)
					assert.NoError(t, err)
				}))
			},
			address: "",
			token:   "dev-token",
			keyPath: "custom/path/key",
			wantErr: false,
		},
		{
			name: "default key path when empty",
			serverFn: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Contains(t, r.URL.Path, "secret/data/gophkeeper/encryption")

					w.Header().Set("Content-Type", "application/json")
					resp := vaultResponse{
						Data: vaultKVData{
							Data: vaultSecretData{
								Key: validKeyB64,
							},
						},
					}
					err := json.NewEncoder(w).Encode(resp)
					assert.NoError(t, err)
				}))
			},
			address: "",
			token:   "dev-token",
			keyPath: "",
			wantErr: false,
		},
		{
			name: "vault returns 403",
			serverFn: func(_ *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("forbidden"))
				}))
			},
			address:     "",
			token:       "wrong-token",
			keyPath:     "secret/data/key",
			wantErr:     true,
			errContains: "vault returned status 403",
		},
		{
			name: "vault returns 404",
			serverFn: func(_ *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte("not found"))
				}))
			},
			address:     "",
			token:       "dev-token",
			keyPath:     "secret/data/nonexistent",
			wantErr:     true,
			errContains: "vault returned status 404",
		},
		{
			name: "invalid JSON response",
			serverFn: func(_ *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte("not json"))
				}))
			},
			address:     "",
			token:       "dev-token",
			keyPath:     "secret/data/key",
			wantErr:     true,
			errContains: "decode response",
		},
		{
			name: "invalid base64 key",
			serverFn: func(_ *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					resp := vaultResponse{
						Data: vaultKVData{
							Data: vaultSecretData{
								Key: "!!!invalid-base64!!!",
							},
						},
					}
					err := json.NewEncoder(w).Encode(resp)
					assert.NoError(t, err)
				}))
			},
			address:     "",
			token:       "dev-token",
			keyPath:     "secret/data/key",
			wantErr:     true,
			errContains: "decode key",
		},
		{
			name: "invalid vault address",
			serverFn: func(_ *testing.T) *httptest.Server {
				return nil
			},
			address:     "::not-a-valid-url::",
			token:       "dev-token",
			keyPath:     "secret/data/key",
			wantErr:     true,
			errContains: "parse vault address",
		},
		{
			name: "unsupported scheme",
			serverFn: func(_ *testing.T) *httptest.Server {
				return nil
			},
			address:     "ftp://vault:8200",
			token:       "dev-token",
			keyPath:     "secret/data/key",
			wantErr:     true,
			errContains: "unsupported scheme: ftp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var address string
			if tc.serverFn != nil {
				srv := tc.serverFn(t)
				if srv != nil {
					defer srv.Close()
					address = srv.URL
				}
			}
			if address == "" {
				address = tc.address
			}

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Error(mock.Anything, mock.Anything).Return().Maybe()

			enc, err := NewVaultEncryptor(t.Context(), address, tc.token, tc.keyPath, l)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, enc)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, enc)
		})
	}
}

func TestVaultEncryptor_EncryptDecrypt(t *testing.T) {
	validKey := make([]byte, 32)
	validKey[0] = 1
	validKeyB64 := base64.StdEncoding.EncodeToString(validKey)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := vaultResponse{
			Data: vaultKVData{
				Data: vaultSecretData{
					Key: validKeyB64,
				},
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		assert.NoError(t, err)
	}))
	defer srv.Close()

	l := intlogger.NewMockLogger(t)
	l.EXPECT().Error(mock.Anything, mock.Anything).Return().Maybe()

	enc, err := NewVaultEncryptor(t.Context(), srv.URL, "dev-token", "secret/data/key", l)
	require.NoError(t, err)

	testCases := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: []byte{}},
		{name: "small", data: []byte("hello")},
		{name: "larger", data: make([]byte, 1024)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ciphertext, err := enc.Encrypt(tc.data)
			require.NoError(t, err)

			decrypted, err := enc.Decrypt(ciphertext)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(tc.data, decrypted))
		})
	}
}
