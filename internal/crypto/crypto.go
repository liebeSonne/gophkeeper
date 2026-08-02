// nolint: revive
package crypto

import "errors"

var ErrInvalidKey = errors.New("invalid encryption key")
var ErrDecryptionFailed = errors.New("decryption failed")

type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}
