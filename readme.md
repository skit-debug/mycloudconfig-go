# My Cloud Config Go SDK

Go SDK for working with encrypted objects stored in a **My Cloud Config (MCC)** service.

The SDK implements the client side of:

* **MCC Object Fetch Protocol v1** — retrieving objects over HTTP
* **MCC Crypto Profile v1** — decrypting and encrypting object payloads

The server stores encrypted payloads as opaque binary data and does not need access to the encryption secret.

## Features

* Fetch an object by URL
* Fetch a specific object version
* Verify the encrypted payload checksum before decryption
* Decrypt payloads using `mcc-crypto-v1`
* AES-256-GCM authenticated encryption
* PBKDF2-HMAC-SHA256 key derivation
* Custom `http.Client` support
* Custom cryptographic implementation support
* No JSON or additional envelope format

## Installation

```bash
go get github.com/skit-debug/mycloudconfig-go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	mccclient "github.com/skit-debug/mycloudconfig-go/client"
)

func main() {
	ctx := context.Background()

	url := "https://example.com/mcc/v1/object/550e8400-e29b-41d4-a716-446655440000"
	secret := []byte("my-secret")

	data, version, err := mccclient.Fetch(ctx, url, secret, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("version: %d\n", version)
	fmt.Printf("data: %s\n", data)
}
```

The SDK:

1. sends an HTTP `GET` request;
2. validates the HTTP status;
3. validates the MCC protocol headers;
4. reads the encrypted payload;
5. verifies its SHA-256 checksum;
6. decrypts the payload using the configured crypto profile;
7. returns the plaintext and object version.

## Fetching a Specific Version

The version is part of the request URL:

```go
url := "https://example.com/mcc/v1/object/550e8400-e29b-41d4-a716-446655440000?version=3"

data, version, err := mccclient.Fetch(
	ctx,
	url,
	secret,
	nil,
)
```

If the version is omitted, the server returns the latest version.

## Custom HTTP Client

A custom `http.Client` can be provided through `FetchOptions`:

```go
client := &http.Client{
	Timeout: 10 * time.Second,
}

data, version, err := mccclient.Fetch(
	ctx,
	url,
	secret,
	&mccclient.FetchOptions{
		HTTPClient: client,
	},
)
```

This can be used to configure timeouts, transports, proxies, TLS settings, or other standard `net/http` behavior.

## Cryptography

The default implementation uses **MCC Crypto Profile v1**.

The profile uses:

* **PBKDF2-HMAC-SHA256** for key derivation
* **100,000 PBKDF2 iterations**
* **32-byte derived key**
* **AES-256-GCM** for authenticated encryption
* **16-byte random salt**
* **12-byte random nonce**

The encrypted payload has the following structure:

```text
| salt (16) | nonce (12) | ciphertext | auth_tag (16) |
```

The secret is an arbitrary byte sequence supplied by the caller.

The salt and nonce are generated randomly and are included in the encrypted payload. They are not secret.

The authentication tag is provided by AES-GCM and is verified during decryption.

## Payload Integrity

The MCC Fetch Protocol provides an additional SHA-256 checksum in the response:

```text
MCC-Object-Checksum: sha256:<64 hex characters>
```

The checksum is calculated over the **complete encrypted payload**.

The SDK verifies the checksum **before attempting decryption**.

The checksum is intended to detect accidental payload corruption. It is not a replacement for the authentication provided by AES-GCM.

## Custom Crypter

The SDK allows a custom implementation of the `Crypter` interface:

```go
type Crypter interface {
	Decrypt(payload []byte, secret []byte) ([]byte, error)
	Encrypt(plaintext []byte, secret []byte) ([]byte, error)
}
```

For example:

```go
crypter := myCrypter{}

data, version, err := mccclient.Fetch(
	ctx,
	url,
	secret,
	&mccclient.FetchOptions{
		Crypter: crypter,
	},
)
```

This makes the transport layer independent from a particular cryptographic implementation and allows additional crypto profiles to be introduced without changing the fetch logic.

## Error Handling

The SDK exposes errors for the main failure categories:

```go
mccclient.ErrDownloadFailed
mccclient.ErrInvalidEnvelope
mccclient.ErrChecksumMismatch
mccclient.ErrDecryptionFailed

mccclient.ErrHTTPBadRequest
mccclient.ErrHTTPNotFound
mccclient.ErrHTTPServerError
```

Errors from the underlying HTTP client and cryptographic implementation may be wrapped with these higher-level errors.

Use `errors.Is` when checking errors:

```go
if errors.Is(err, mccclient.ErrChecksumMismatch) {
	// The encrypted payload was corrupted or does not match the checksum.
}
```

## Protocol Validation

The client validates the following response headers:

```text
MCC-Protocol: mcc-protocol-v1
MCC-Crypto-Profile: mcc-crypto-v1
Content-Type: application/octet-stream
```

It also validates the checksum before decryption.

The object version is returned separately:

```go
data, version, err := mccclient.Fetch(...)
```

If the `MCC-Object-Version` header is absent, the returned version is `0`.

## Architecture

The SDK is intentionally split into two independent layers:

```text
                    My Cloud Config Server
                             │
                             │ HTTP GET
                             ▼
                       ┌───────────┐
                       │  client   │
                       │  package  │
                       └─────┬─────┘
                             │
                   verify protocol
                   verify checksum
                             │
                             ▼
                       ┌───────────┐
                       │  Crypter  │
                       └─────┬─────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ mcc-crypto-v1   │
                    │                 │
                    │ PBKDF2-SHA256   │
                    │ AES-256-GCM     │
                    └─────────────────┘
```

The transport layer does not depend on the internal details of the cryptographic profile.

The server only stores and returns the encrypted payload. It does not need to know the secret or plaintext.

## Packages

### `client`

HTTP client implementing the MCC Object Fetch Protocol.

Main entry point:

```go
client.Fetch(...)
```

### `mcccrypto`

Cryptographic profile abstraction and profile selection.

```go
mcccrypto.NewCrypter(mcccrypto.CryptoProfileV1)
```

### `mcc-crypto/internal/v1`

Implementation of MCC Crypto Profile v1.

This package is internal and is not intended to be used directly.

## Specifications

The SDK is based on two protocol specifications:

* **MCC Object Fetch Protocol v1**
* **MCC Crypto Profile v1**

The specifications are maintained separately from the SDK and define the wire format and interoperability requirements.

## Design Principles

The SDK follows a few simple principles:

* **Opaque server storage** — the server does not interpret encrypted object contents.
* **Client-side cryptography** — encryption and decryption happen on the client.
* **Explicit protocols** — transport and cryptography are defined independently.
* **Minimal wire format** — no JSON envelope or unnecessary metadata in the payload.
* **Profile-based cryptography** — new cryptographic schemes are introduced as new profiles.
* **Standard Go interfaces** — HTTP and cryptographic implementations can be replaced where necessary.

## Security Notes

The SDK does not provide key management.

Callers are responsible for protecting secrets used for encryption and decryption.

The same secret should not be exposed to the MCC server.

AES-GCM requires nonce uniqueness for a given encryption key. The MCC Crypto Profile generates a new random nonce for every encrypted payload.

The SHA-256 transport checksum provides corruption detection but must not be treated as an authentication mechanism.

## Status

This SDK currently implements:

* MCC Object Fetch Protocol v1
* MCC Crypto Profile v1

The project is intended as an MVP and may introduce additional protocol and cryptographic profiles in future versions.
