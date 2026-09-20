package v1_test

import (
	"testing"

	v1 "github.com/skit-debug/mycloudconfig-go/mcc-crypto/internal/v1"

	"github.com/stretchr/testify/require"
)

func Test_Crypter(t *testing.T) {
	crypt := v1.NewV1Crypter()

	text := []byte("this is a text")
	password := []byte("somePassword")

	plaintext, err := crypt.Encrypt(text, password)
	require.NoError(t, err)

	decryptedText, err := crypt.Decrypt(plaintext, password)
	require.NoError(t, err)

	require.Equal(t, text, decryptedText)
}
