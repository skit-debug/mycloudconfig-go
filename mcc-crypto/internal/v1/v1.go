package v1

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// saltSize is the size of the PBKDF2 salt in bytes.
	saltSize = 16

	// keySize is the AES-256 key size in bytes.
	keySize = 32

	// iterations is the PBKDF2 iteration count.
	iterations = 100_000
)

var (
	ErrInvalidPayload       = errors.New("crypto: invalid payload")
	ErrAuthenticationFailed = errors.New("crypto: authentication failed")
)

func NewV1Crypter() *crypter {
	return &crypter{}
}

type crypter struct{}

func (c *crypter) Encrypt(plaintext, secret []byte) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key(
		secret,
		salt,
		iterations,
		keySize,
		sha256.New,
	)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// The authentication tag is appended to the ciphertext by GCM.
	ciphertext := gcm.Seal(
		nil,
		nonce,
		plaintext,
		nil,
	)

	payload := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	payload = append(payload, salt...)
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	return payload, nil
}

func (c *crypter) Decrypt(payload, secret []byte) ([]byte, error) {
	if len(payload) < saltSize {
		return nil, ErrInvalidPayload
	}

	salt := payload[:saltSize]

	key := pbkdf2.Key(
		secret,
		salt,
		iterations,
		keySize,
		sha256.New,
	)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()

	if len(payload) < saltSize+nonceSize+gcm.Overhead() {
		return nil, ErrInvalidPayload
	}

	nonceStart := saltSize
	nonceEnd := nonceStart + nonceSize
	nonce := payload[nonceStart:nonceEnd]

	// The remaining bytes contain the ciphertext and authentication tag.
	ciphertext := payload[nonceEnd:]

	plaintext, err := gcm.Open(
		nil,
		nonce,
		ciphertext,
		nil,
	)
	if err != nil {
		return nil, ErrAuthenticationFailed
	}

	return plaintext, nil
}
