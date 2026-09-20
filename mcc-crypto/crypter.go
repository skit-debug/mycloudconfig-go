package mcccrypto

import (
	"errors"

	v1 "github.com/skit-debug/mycloudconfig-go/mcc-crypto/internal/v1"
)

var ErrWrongProfile = errors.New("crypto: wrong profile")

// Worker defines encryption and decryption operations used by a cryptographic profile.
type Worker interface {
	Decrypt(payload []byte, secret []byte) ([]byte, error)
	Encrypt(plaintext []byte, secret []byte) ([]byte, error)
}

// NewCrypter creates a crypter for the specified cryptographic profile.
func NewCrypter(profile CryptoProfile) (*crypter, error) {
	switch profile { //nolint:gocritic
	case CryptoProfileV1:
		return &crypter{v1.NewV1Crypter()}, nil
	}

	return nil, ErrWrongProfile
}

type crypter struct {
	worker Worker
}

func (c *crypter) Encrypt(plaintext []byte, secret []byte) ([]byte, error) {
	return c.worker.Encrypt(plaintext, secret)
}

func (c *crypter) Decrypt(payload []byte, secret []byte) ([]byte, error) {
	return c.worker.Decrypt(payload, secret)
}
