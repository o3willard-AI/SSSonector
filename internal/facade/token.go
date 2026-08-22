// Package facade provides HTTPS facade functionality for firewall traversal.
// It allows tunnel connections to operate over standard HTTPS (port 443) by
// disguising them as WebSocket upgrades, making traffic indistinguishable from
// normal web browsing to firewalls and deep packet inspection systems.
package facade

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	// tokenPortSize is the byte size of the port field in the token
	tokenPortSize = 2
	// tokenTimestampSize is the byte size of the timestamp field in the token
	tokenTimestampSize = 8
	// tokenHMACSize is the byte size of the HMAC-SHA256 signature
	tokenHMACSize = 32
	// tokenTotalSize is the total binary token size before base64 encoding
	tokenTotalSize = tokenPortSize + tokenTimestampSize + tokenHMACSize

	// DefaultTokenTTL is the default token validity duration
	DefaultTokenTTL = 30 * time.Second
)

// GenerateToken creates an HMAC-signed token encoding the target tunnel port.
// The token format is: base64(port[2] || timestamp[8] || hmac-sha256[32])
// where the HMAC is computed over port || timestamp using the shared secret.
func GenerateToken(port int, secret []byte) (string, error) {
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid port: %d", port)
	}
	if len(secret) == 0 {
		return "", fmt.Errorf("token secret cannot be empty")
	}

	// Build the payload: port (2 bytes, big-endian) + timestamp (8 bytes, big-endian)
	payload := make([]byte, tokenPortSize+tokenTimestampSize)
	binary.BigEndian.PutUint16(payload[0:tokenPortSize], uint16(port))
	binary.BigEndian.PutUint64(payload[tokenPortSize:tokenPortSize+tokenTimestampSize], uint64(time.Now().Unix()))

	// Compute HMAC-SHA256 over the payload
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	signature := mac.Sum(nil)

	// Concatenate payload + signature and base64-encode
	token := make([]byte, tokenTotalSize)
	copy(token[0:], payload)
	copy(token[tokenPortSize+tokenTimestampSize:], signature)

	return base64.StdEncoding.EncodeToString(token), nil
}

// ValidateToken verifies an HMAC-signed token and extracts the target port.
// Returns the port number if the token is valid and not expired, or an error.
func ValidateToken(tokenStr string, secret []byte, ttl time.Duration) (int, error) {
	if len(secret) == 0 {
		return 0, fmt.Errorf("token secret cannot be empty")
	}

	// Decode base64
	token, err := base64.StdEncoding.DecodeString(tokenStr)
	if err != nil {
		return 0, fmt.Errorf("invalid token encoding: %w", err)
	}

	if len(token) != tokenTotalSize {
		return 0, fmt.Errorf("invalid token size: expected %d, got %d", tokenTotalSize, len(token))
	}

	// Extract components
	payload := token[0 : tokenPortSize+tokenTimestampSize]
	receivedMAC := token[tokenPortSize+tokenTimestampSize:]

	// Verify HMAC
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)

	if !hmac.Equal(receivedMAC, expectedMAC) {
		return 0, fmt.Errorf("invalid token signature")
	}

	// Extract and validate timestamp
	timestamp := int64(binary.BigEndian.Uint64(payload[tokenPortSize : tokenPortSize+tokenTimestampSize]))
	tokenTime := time.Unix(timestamp, 0)
	elapsed := time.Since(tokenTime)

	if elapsed < 0 {
		// Token is from the future -- allow up to 5 seconds of clock skew
		if elapsed < -5*time.Second {
			return 0, fmt.Errorf("token timestamp is in the future")
		}
	} else if elapsed > ttl {
		return 0, fmt.Errorf("token expired: age %v exceeds TTL %v", elapsed, ttl)
	}

	// Extract port
	port := int(binary.BigEndian.Uint16(payload[0:tokenPortSize]))
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port in token: %d", port)
	}

	return port, nil
}

// ResolveSecret resolves the facade token secret from configuration.
// The secret MUST be configured explicitly on both server and client.
// Deriving it from public material such as the CA certificate is not
// permitted: the CA certificate is distributed to every client by design,
// so any derivation from it would allow anyone holding the CA file to
// forge valid tunnel tokens.
func ResolveSecret(tokenSecret string, _ string) ([]byte, error) {
	if tokenSecret == "" {
		return nil, fmt.Errorf("facade token_secret is required: configure a high-entropy shared secret explicitly on both server and client")
	}
	hash := sha256.Sum256([]byte(tokenSecret))
	return hash[:], nil
}
