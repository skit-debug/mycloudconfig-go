package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"

	mcccrypto "github.com/skit-debug/mycloudconfig-go/mcc-crypto"
)

//go:generate mkdir -p mock
//go:generate ../tooling/bin/mockgen -source=fetch.go -destination=mock/fetch.go -package=mock

// Crypter encrypts and decrypts object payloads.
type Crypter interface {
	Decrypt(payload []byte, secret []byte) ([]byte, error)
	Encrypt(plaintext []byte, secret []byte) ([]byte, error)
}

type FetchOptions struct {
	HTTPClient *http.Client
	Crypter    Crypter
}

// Fetch downloads an object from the specified URL and decrypts its payload.
//
// The URL must contain the object ID and may optionally contain a version.
// secret is the secret used for decryption.
// opts specifies optional HTTP client and crypter configuration.
func Fetch(ctx context.Context, url string, secret []byte, opts *FetchOptions) ([]byte, int, error) {
	client := http.DefaultClient
	if opts != nil && opts.HTTPClient != nil {
		client = opts.HTTPClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, errors.Join(err, ErrDownloadFailed)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.Join(err, ErrDownloadFailed)
	}
	defer resp.Body.Close() //nolint:errcheck

	err = checkStatus(resp.StatusCode)
	if err != nil {
		return nil, 0, err
	}

	err = checkProtocol(resp)
	if err != nil {
		return nil, 0, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, ErrInvalidEnvelope
	}

	err = verifyChecksum(resp, body)
	if err != nil {
		return nil, 0, err
	}

	var crypter Crypter
	if opts != nil && opts.Crypter != nil {
		crypter = opts.Crypter
	} else {
		crypter, err = mcccrypto.NewCrypter(mcccrypto.CryptoProfileV1)
		if err != nil {
			return nil, 0, errors.Join(err, ErrInvalidEnvelope)
		}
	}

	plain, err := crypter.Decrypt(body, secret)
	if err != nil {
		return nil, 0, errors.Join(err, ErrDecryptionFailed)
	}

	version, err := getVersion(resp)
	if err != nil {
		return nil, 0, err
	}

	return plain, version, nil
}

func checkStatus(status int) error {
	switch status {
	case 200:
		return nil
	case 400:
		return ErrHTTPBadRequest
	case 404:
		return ErrHTTPNotFound
	case 500:
		return ErrHTTPServerError
	default:
		return ErrInvalidEnvelope
	}
}

func checkProtocol(resp *http.Response) error {
	if resp.Header.Get(headerProtocol) != "mcc-protocol-v1" ||
		resp.Header.Get(headerContentType) != "application/octet-stream" ||
		resp.Header.Get(headerCryptoProfile) != string(mcccrypto.CryptoProfileV1) {
		return ErrInvalidEnvelope
	}

	return nil
}

func verifyChecksum(resp *http.Response, body []byte) error {
	checksum := resp.Header.Get(headerObjectChecksum)
	if checksum == "" {
		return ErrInvalidEnvelope
	}

	sum := sha256.Sum256(body)
	calculated := "sha256:" + hex.EncodeToString(sum[:])

	if checksum != calculated {
		return ErrChecksumMismatch
	}

	return nil
}

func getVersion(resp *http.Response) (int, error) {
	versionStr := resp.Header.Get(headerObjectVersion)
	if versionStr == "" {
		return 0, nil
	}

	version, err := strconv.Atoi(versionStr)

	return version, err
}
