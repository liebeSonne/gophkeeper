// nolint: revive
package crypto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
)

const defaultVaultKeyPath = "secret/data/gophkeeper/encryption"

type VaultEncryptor struct {
	aes *AESGCM
}

func NewVaultEncryptor(
	ctx context.Context,
	address, token, keyPath string,
	client *http.Client,
	logger intlogger.Logger,
) (*VaultEncryptor, error) {
	if keyPath == "" {
		keyPath = defaultVaultKeyPath
	}

	key, err := loadKeyFromVault(ctx, address, token, keyPath, client, logger)
	if err != nil {
		return nil, fmt.Errorf("load key from vault: %w", err)
	}

	aes, err := NewAESGCM(key)
	if err != nil {
		return nil, fmt.Errorf("create aes gcm: %w", err)
	}

	return &VaultEncryptor{aes: aes}, nil
}

func (v *VaultEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	return v.aes.Encrypt(plaintext)
}

func (v *VaultEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	return v.aes.Decrypt(ciphertext)
}

func loadKeyFromVault(
	ctx context.Context,
	address, token, keyPath string,
	client *http.Client,
	logger intlogger.Logger,
) ([]byte, error) {
	base, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parse vault address: %w", err)
	}

	if base.Scheme != "https" && base.Scheme != "http" {
		return nil, fmt.Errorf("unsupported scheme: %s", base.Scheme)
	}

	relPath := fmt.Sprintf("v1/%s", strings.TrimPrefix(keyPath, "/"))
	rel, err := url.Parse(relPath)
	if err != nil {
		return nil, fmt.Errorf("invalid key path: %w", err)
	}

	fullURL := base.ResolveReference(rel).String()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("Content-Type", "application/json")

	// nolint: gosec
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.Error("failed to close response body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("vault returned status %d: %s", resp.StatusCode, string(body))
	}

	var result vaultResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	key, err := base64.StdEncoding.DecodeString(result.Data.Data.Key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}

	return key, nil
}

type vaultResponse struct {
	Data vaultKVData `json:"data"`
}

type vaultKVData struct {
	Data vaultSecretData `json:"data"`
}

type vaultSecretData struct {
	Key string `json:"key"`
}
