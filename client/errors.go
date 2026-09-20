package client

import "errors"

var (
	ErrInvalidEnvelope = errors.New("mcc: invalid envelope")

	ErrDownloadFailed   = errors.New("mcc: download failed")
	ErrChecksumMismatch = errors.New("mcc: checksum mismatch")
	ErrDecryptionFailed = errors.New("mcc: decryption failed")

	ErrHTTPBadRequest  = errors.New("mcc: invalid request (400)")
	ErrHTTPNotFound    = errors.New("mcc: object or version not found (404)")
	ErrHTTPServerError = errors.New("mcc: internal server error (500)")
)
