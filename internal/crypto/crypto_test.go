// nolint: revive
package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAESGCM(t *testing.T) {
	testCases := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			key:     makeKey(t, 32),
			wantErr: false,
		},
		{
			name:    "too short key",
			key:     makeKey(t, 16),
			wantErr: true,
		},
		{
			name:    "too long key",
			key:     makeKey(t, 64),
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     []byte{},
			wantErr: true,
		},
		{
			name:    "nil key",
			key:     nil,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := NewAESGCM(tc.key)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, enc)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, enc)
		})
	}
}

func TestAESGCM_EncryptDecrypt(t *testing.T) {
	key := makeKey(t, 32)
	enc, err := NewAESGCM(key)
	require.NoError(t, err)

	testCases := []struct {
		name string
		size int
	}{
		{name: "empty", size: 0},
		{name: "1 byte", size: 1},
		{name: "10 bytes", size: 10},
		{name: "100 bytes", size: 100},
		{name: "1 KB", size: 1024},
		{name: "10 KB", size: 10240},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plaintext := makeKey(t, tc.size)

			ciphertext, err := enc.Encrypt(plaintext)
			require.NoError(t, err)
			assert.NotEqual(t, plaintext, ciphertext)
			assert.GreaterOrEqual(t, len(ciphertext), len(plaintext)+nonceSize)

			decrypted, err := enc.Decrypt(ciphertext)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(plaintext, decrypted))
		})
	}
}

func TestAESGCM_DecryptCorrupted(t *testing.T) {
	key := makeKey(t, 32)
	enc, err := NewAESGCM(key)
	require.NoError(t, err)

	plaintext := []byte("test data")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	corrupted := make([]byte, len(ciphertext))
	copy(corrupted, ciphertext)
	corrupted[len(corrupted)-1] ^= 0xFF

	_, err = enc.Decrypt(corrupted)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestAESGCM_DecryptTooShort(t *testing.T) {
	key := makeKey(t, 32)
	enc, err := NewAESGCM(key)
	require.NoError(t, err)

	_, err = enc.Decrypt([]byte("short"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestAESGCM_DifferentKeys(t *testing.T) {
	key1 := makeKey(t, 32)
	key2 := makeKey(t, 32)

	enc1, err := NewAESGCM(key1)
	require.NoError(t, err)

	enc2, err := NewAESGCM(key2)
	require.NoError(t, err)

	plaintext := []byte("secret data")
	ciphertext, err := enc1.Encrypt(plaintext)
	require.NoError(t, err)

	_, err = enc2.Decrypt(ciphertext)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestAESGCM_RandomNonce(t *testing.T) {
	key := makeKey(t, 32)
	enc, err := NewAESGCM(key)
	require.NoError(t, err)

	plaintext := []byte("same plaintext")
	ciphertexts := make(map[string]bool)

	for i := 0; i < 10; i++ {
		ct, err := enc.Encrypt(plaintext)
		require.NoError(t, err)
		assert.False(t, ciphertexts[string(ct)], "ciphertext should be unique (different nonce)")
		ciphertexts[string(ct)] = true
	}
}

func TestAESGCM_DecryptEmptyPlaintext(t *testing.T) {
	key := makeKey(t, 32)
	enc, err := NewAESGCM(key)
	require.NoError(t, err)

	ciphertext, err := enc.Encrypt([]byte{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(ciphertext), nonceSize)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.True(t, bytes.Equal([]byte{}, decrypted))
}

func makeKey(t *testing.T, size int) []byte {
	t.Helper()
	key := make([]byte, size)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}
