package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESGCM_SameKeyDifferentNonce(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	encryptor, err := NewAESGCM(key)
	require.NoError(t, err)
	require.NotNil(t, encryptor)

	plaintext := []byte("the same plaintext")

	ciphertext1, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	ciphertext2, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, ciphertext1, ciphertext2,
		"same plaintext with same key should produce different ciphertexts due to random nonce")

	decrypted1, err := encryptor.Decrypt(ciphertext1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := encryptor.Decrypt(ciphertext2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}
