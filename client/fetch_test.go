package client_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skit-debug/mycloudconfig-go/client"
	"github.com/skit-debug/mycloudconfig-go/client/mock"
	mcccrypto "github.com/skit-debug/mycloudconfig-go/mcc-crypto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	const (
		secret  = "secret"
		body    = "encrypted payload"
		plain   = "decrypted payload"
		version = 42
	)

	checksum := func(body []byte) string {
		sum := sha256.Sum256(body)
		return "sha256:" + hex.EncodeToString(sum[:])
	}

	tests := []struct {
		name       string
		statusCode int
		body       string
		headers    func(http.Header)
		decryptErr error

		wantBody    string
		wantVersion int
		wantErr     error
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v1")
				h.Set("Content-Type", "application/octet-stream")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfileV1))
				h.Set("MCC-Object-Checksum", checksum([]byte(body)))
				h.Set("MCC-Object-Version", "42")
			},
			wantBody:    plain,
			wantVersion: version,
		},
		{
			name:       "success without version",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v1")
				h.Set("Content-Type", "application/octet-stream")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfileV1))
				h.Set("MCC-Object-Checksum", checksum([]byte(body)))
			},
			wantBody:    plain,
			wantVersion: 0,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			wantErr:    client.ErrHTTPBadRequest,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    client.ErrHTTPNotFound,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    client.ErrHTTPServerError,
		},
		{
			name:       "unexpected status",
			statusCode: http.StatusCreated,
			wantErr:    client.ErrInvalidEnvelope,
		},
		{
			name:       "invalid protocol (protocol version)",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v42")
				h.Set("Content-Type", "application/octet-stream")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfileV1))
				h.Set("MCC-Object-Checksum", checksum([]byte(body)))
			},
			wantErr: client.ErrInvalidEnvelope,
		},
		{
			name:       "invalid protocol (content type)",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v1")
				h.Set("Content-Type", "application/json")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfileV1))
				h.Set("MCC-Object-Checksum", checksum([]byte(body)))
			},
			wantErr: client.ErrInvalidEnvelope,
		},
		{
			name:       "invalid protocol (crypto profile)",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v1")
				h.Set("Content-Type", "application/octet-stream")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfile("mcc-crypto-v42")))
				h.Set("MCC-Object-Checksum", checksum([]byte(body)))
			},
			wantErr: client.ErrInvalidEnvelope,
		},
		{
			name:       "missing checksum",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v1")
				h.Set("Content-Type", "application/octet-stream")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfileV1))
			},
			wantErr: client.ErrInvalidEnvelope,
		},
		{
			name:       "invalid checksum",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v1")
				h.Set("Content-Type", "application/octet-stream")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfileV1))
				h.Set("MCC-Object-Checksum", "sha256:invalid")
			},
			wantErr: client.ErrChecksumMismatch,
		},
		{
			name:       "decryption failed",
			statusCode: http.StatusOK,
			body:       body,
			headers: func(h http.Header) {
				h.Set("MCC-Protocol", "mcc-protocol-v1")
				h.Set("Content-Type", "application/octet-stream")
				h.Set("MCC-Crypto-Profile", string(mcccrypto.CryptoProfileV1))
				h.Set("MCC-Object-Checksum", checksum([]byte(body)))
			},
			decryptErr: errors.New("decrypt failed"),
			wantErr:    client.ErrDecryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			crypter := mock.NewMockCrypter(ctrl)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.headers != nil {
					tt.headers(w.Header())
				}

				w.WriteHeader(tt.statusCode)

				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			}))
			defer server.Close()

			if tt.statusCode == http.StatusOK &&
				tt.headers != nil &&
				!errors.Is(tt.wantErr, client.ErrInvalidEnvelope) &&
				!errors.Is(tt.wantErr, client.ErrChecksumMismatch) {
				crypter.
					EXPECT().
					Decrypt([]byte(tt.body), []byte(secret)).
					Return([]byte(tt.wantBody), tt.decryptErr)
			}

			got, gotVersion, err := client.Fetch(
				t.Context(),
				server.URL,
				[]byte(secret),
				&client.FetchOptions{
					HTTPClient: server.Client(),
					Crypter:    crypter,
				},
			)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantBody, string(got))
			assert.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}
