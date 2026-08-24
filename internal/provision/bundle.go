// Package provision implements first-run enrollment for SSSonector peers:
// encrypted .ssp bundles, pairing codes, network redemption, CSR mode, and
// certificate verification. Spec: docs/provisioning_design.md (ACCEPTED).
package provision

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	// Magic marks an .ssp envelope; it is not a trust indicator.
	Magic = "SSP1"

	// FormatVersion is the only envelope version this build reads/writes.
	FormatVersion = 1

	// KDFIDArgon2id selects Argon2id with the v1 parameter set below.
	KDFIDArgon2id = 1

	// SaltLen / NonceLen are fixed at v1.
	SaltLen  = 16
	NonceLen = chacha20poly1305.NonceSizeX // 24 bytes

	// Argon2id v1 parameters: 64 MiB memory, 3 passes, 4 lanes.
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // KiB
	argon2Threads = 4
	keyLen        = 32
)

var (
	ErrBadMagic     = errors.New("provision: not an SSP1 bundle")
	ErrBadVersion   = errors.New("provision: unsupported bundle version")
	ErrBadKDF       = errors.New("provision: unknown KDF params id")
	ErrTruncated    = errors.New("provision: truncated bundle")
	ErrAuthFailed   = errors.New("provision: decryption failed (wrong pairing code or tampered bundle)")
	ErrEmptyPairing = errors.New("provision: pairing code must not be empty")
	ErrEmptySecret  = errors.New("provision: facade_token_secret must not be empty")
	ErrBadRole      = errors.New("provision: role must be 'client' or 'server'")
)

// PairingPayload is the JSON-encrypted body of an .ssp bundle.
type PairingPayload struct {
	Role              string `json:"role"`
	ServerAddr        string `json:"server_addr"`
	ServerPort        int    `json:"server_port"`
	FacadeTokenSecret string `json:"facade_token_secret"`
	CACertPEM         string `json:"ca_cert_pem"`
	ClientCertPEM     string `json:"client_cert_pem,omitempty"`
	ClientKeyPEM      string `json:"client_key_pem,omitempty"`
	CreatedAtRFC3339  string `json:"created_at"`
	Name              string `json:"name,omitempty"`
	FingerprintOfCA   string `json:"fingerprint_of_ca"`
}

// Validate enforces fail-closed payload rules.
func (p *PairingPayload) Validate() error {
	if p.Role != "client" && p.Role != "server" {
		return ErrBadRole
	}
	if p.FacadeTokenSecret == "" {
		return ErrEmptySecret
	}
	if p.CACertPEM == "" || p.FingerprintOfCA == "" {
		return errors.New("provision: CA material and fingerprint are required")
	}
	if p.ServerAddr == "" || p.ServerPort <= 0 {
		return errors.New("provision: server address/port are required")
	}
	return nil
}

// deriveKey runs the v1 KDF over the normalized pairing code.
func deriveKey(code string, salt []byte) []byte {
	return argon2.IDKey([]byte(code), salt, argon2Time, argon2Memory, argon2Threads, keyLen)
}

// Seal encodes and encrypts the payload under the pairing code.
// Output layout: Magic | ver(1) | kdfID(2,BE) | salt | nonce | AEAD ct.
func Seal(payload *PairingPayload, code string) ([]byte, error) {
	code, err := NormalizePairingCode(code)
	if err != nil {
		return nil, err
	}
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("provision: marshal payload: %w", err)
	}

	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("provision: salt: %w", err)
	}
	nonce := make([]byte, NonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("provision: nonce: %w", err)
	}

	aead, err := chacha20poly1305.NewX(deriveKey(code, salt))
	if err != nil {
		return nil, fmt.Errorf("provision: aead: %w", err)
	}

	out := make([]byte, 0, len(Magic)+1+2+SaltLen+NonceLen+len(plaintext)+aead.Overhead())
	out = append(out, Magic...)
	out = append(out, byte(FormatVersion))
	var kdf [2]byte
	binary.BigEndian.PutUint16(kdf[:], KDFIDArgon2id)
	out = append(out, kdf[:]...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = aead.Seal(out, nonce, plaintext, nil)
	return out, nil
}

// Open authenticates and decrypts an envelope produced by Seal.
// Wrong code and tampered ciphertext are indistinguishable (both yield
// ErrAuthFailed) by design.
func Open(envelope []byte, code string) (*PairingPayload, error) {
	// Canonicalize first: display grouping ("XXXX-XXXX") and operator-pasted
	// variants must derive the identical AEAD key.
	norm, nerr := NormalizePairingCode(code)
	if nerr != nil {
		return nil, nerr
	}
	code = norm
	fixed := len(Magic) + 1 + 2 + SaltLen + NonceLen
	if len(envelope) < fixed+aeadOverhead() {
		return nil, ErrTruncated
	}
	if string(envelope[:len(Magic)]) != Magic {
		return nil, ErrBadMagic
	}
	pos := len(Magic)
	ver := int(envelope[pos])
	pos++
	if ver != FormatVersion {
		return nil, fmt.Errorf("%w: %d", ErrBadVersion, ver)
	}
	kdf := binary.BigEndian.Uint16(envelope[pos : pos+2])
	pos += 2
	if kdf != KDFIDArgon2id {
		return nil, fmt.Errorf("%w: %d", ErrBadKDF, kdf)
	}
	salt := envelope[pos : pos+SaltLen]
	pos += SaltLen
	nonce := envelope[pos : pos+NonceLen]
	pos += NonceLen
	ct := envelope[pos:]

	aead, err := chacha20poly1305.NewX(deriveKey(code, salt))
	if err != nil {
		return nil, fmt.Errorf("provision: aead: %w", err)
	}
	plaintext, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrAuthFailed
	}
	var payload PairingPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("provision: unmarshal payload: %w", err)
	}
	if err := (&payload).Validate(); err != nil {
		return nil, err
	}
	return &payload, nil
}

func aeadOverhead() int { return chacha20poly1305.Overhead }

// Fingerprint returns the SHA-256 hex fingerprint of PEM certificate bytes,
// the value embedded in bundles and pinned by apply before tunnel trust.
func Fingerprint(certPEM []byte) string {
	sum := sha256.Sum256(certPEM)
	return fmt.Sprintf("%x", sum)
}
