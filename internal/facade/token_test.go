package facade

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := []byte("test-secret-key-for-hmac-signing")

	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 8443", 8443, false},
		{"valid port 443", 443, false},
		{"valid port 1", 1, false},
		{"valid port 65535", 65535, false},
		{"invalid port 0", 0, true},
		{"invalid port negative", -1, true},
		{"invalid port too high", 70000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.port, secret)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate the token
			port, err := ValidateToken(token, secret, DefaultTokenTTL)
			require.NoError(t, err)
			assert.Equal(t, tt.port, port)
		})
	}
}

func TestGenerateTokenEmptySecret(t *testing.T) {
	_, err := GenerateToken(8443, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret cannot be empty")

	_, err = GenerateToken(8443, []byte{})
	assert.Error(t, err)
}

func TestValidateTokenEmptySecret(t *testing.T) {
	_, err := ValidateToken("sometoken", nil, DefaultTokenTTL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret cannot be empty")
}

func TestValidateTokenInvalidBase64(t *testing.T) {
	secret := []byte("test-secret")
	_, err := ValidateToken("not-valid-base64!!!", secret, DefaultTokenTTL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token encoding")
}

func TestValidateTokenWrongSize(t *testing.T) {
	secret := []byte("test-secret")
	// Too short
	shortToken := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	_, err := ValidateToken(shortToken, secret, DefaultTokenTTL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token size")
}

func TestValidateTokenWrongSecret(t *testing.T) {
	secret1 := []byte("secret-one")
	secret2 := []byte("secret-two")

	token, err := GenerateToken(8443, secret1)
	require.NoError(t, err)

	_, err = ValidateToken(token, secret2, DefaultTokenTTL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token signature")
}

func TestValidateTokenExpired(t *testing.T) {
	secret := []byte("test-secret")

	// Create a token with a timestamp in the past
	payload := make([]byte, tokenPortSize+tokenTimestampSize)
	binary.BigEndian.PutUint16(payload[0:tokenPortSize], uint16(8443))
	// Set timestamp to 60 seconds ago
	binary.BigEndian.PutUint64(payload[tokenPortSize:], uint64(time.Now().Add(-60*time.Second).Unix()))

	// Sign it
	mac := hmacSHA256(payload, secret)
	token := make([]byte, tokenTotalSize)
	copy(token[0:], payload)
	copy(token[tokenPortSize+tokenTimestampSize:], mac)

	tokenStr := base64.StdEncoding.EncodeToString(token)

	// Validate with a 30-second TTL -- should fail
	_, err := ValidateToken(tokenStr, secret, 30*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
}

func TestValidateTokenTamperedPort(t *testing.T) {
	secret := []byte("test-secret")

	token, err := GenerateToken(8443, secret)
	require.NoError(t, err)

	// Decode, tamper with port, re-encode
	raw, err := base64.StdEncoding.DecodeString(token)
	require.NoError(t, err)

	// Change port from 8443 to 9999
	binary.BigEndian.PutUint16(raw[0:tokenPortSize], 9999)

	tampered := base64.StdEncoding.EncodeToString(raw)
	_, err = ValidateToken(tampered, secret, DefaultTokenTTL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token signature")
}

func TestValidateTokenFutureTimestampBeyondSkew(t *testing.T) {
	secret := []byte("test-secret")

	payload := make([]byte, tokenPortSize+tokenTimestampSize)
	binary.BigEndian.PutUint16(payload[0:tokenPortSize], uint16(8443))
	// Timestamp one hour in the future -- beyond the allowed clock skew
	binary.BigEndian.PutUint64(payload[tokenPortSize:], uint64(time.Now().Add(time.Hour).Unix()))

	mac := hmacSHA256(payload, secret)
	token := make([]byte, tokenTotalSize)
	copy(token[0:], payload)
	copy(token[tokenPortSize+tokenTimestampSize:], mac)

	tokenStr := base64.StdEncoding.EncodeToString(token)

	_, err := ValidateToken(tokenStr, secret, DefaultTokenTTL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "future")
}

func TestValidateTokenFutureTimestampWithinSkew(t *testing.T) {
	secret := []byte("test-secret")

	payload := make([]byte, tokenPortSize+tokenTimestampSize)
	binary.BigEndian.PutUint16(payload[0:tokenPortSize], uint16(8443))
	// Timestamp 2 seconds in the future -- inside the 5-second skew window
	binary.BigEndian.PutUint64(payload[tokenPortSize:], uint64(time.Now().Add(2*time.Second).Unix()))

	mac := hmacSHA256(payload, secret)
	token := make([]byte, tokenTotalSize)
	copy(token[0:], payload)
	copy(token[tokenPortSize+tokenTimestampSize:], mac)

	tokenStr := base64.StdEncoding.EncodeToString(token)

	port, err := ValidateToken(tokenStr, secret, DefaultTokenTTL)
	assert.NoError(t, err)
	assert.Equal(t, 8443, port)
}


func TestValidateTokenTamperedTimestamp(t *testing.T) {
	secret := []byte("test-secret")

	token, err := GenerateToken(8443, secret)
	require.NoError(t, err)

	// Decode, tamper with timestamp, re-encode
	raw, err := base64.StdEncoding.DecodeString(token)
	require.NoError(t, err)

	// Change timestamp
	binary.BigEndian.PutUint64(raw[tokenPortSize:tokenPortSize+tokenTimestampSize], uint64(time.Now().Add(time.Hour).Unix()))

	tampered := base64.StdEncoding.EncodeToString(raw)
	_, err = ValidateToken(tampered, secret, DefaultTokenTTL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token signature")
}

func TestResolveSecret(t *testing.T) {
	// With explicit secret
	secret, err := ResolveSecret("my-explicit-secret", "")
	require.NoError(t, err)
	assert.Len(t, secret, 32)
	expectedHash := sha256.Sum256([]byte("my-explicit-secret"))
	assert.Equal(t, expectedHash[:], secret)

	// Deterministic for the same input
	secret2, err := ResolveSecret("my-explicit-secret", "")
	require.NoError(t, err)
	assert.Equal(t, secret, secret2)

	// Without an explicit secret -- must fail closed even when a CA file is
	// provided. Secrets derived from public material (the CA certificate is
	// distributed to every client) would allow token forgery.
	_, err = ResolveSecret("", "/some/ca.crt")
	assert.Error(t, err)

	_, err = ResolveSecret("", "")
	assert.Error(t, err)
}

// hmacSHA256 is a test helper that computes HMAC-SHA256
func hmacSHA256(data, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
