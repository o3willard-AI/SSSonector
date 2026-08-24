package provision

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
)

// codeAlphabet is the unambiguous pairing-code set: digits + uppercase minus
// I, L, O, U (visually confusable with 1, 1, 0, V respectively). 32 symbols
// => 5 bits of entropy per character.
const codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const (
	codeGroupLen   = 4
	codeGroups     = 2
	codeChars      = codeGroupLen * codeGroups // 8 chars
	codeEntropy    = codeChars * 5             // 40 bits
	groupSeparator = '-'
)

var errCodeChar = errors.New("provision: pairing code contains an invalid character")

// GeneratePairingCode returns a fresh display-grouped code ("XXXX-XXXX").
//
// Entropy note (ADR amendment, committed alongside Phase 1): 40 bits is below
// the design's "~48" aspiration. This is deliberate: online guessing is the
// only viable attack and it is bounded by redemption rate limiting plus a
// 15-minute default TTL (~hundreds of attempts vs ~10^12 space). Offline
// brute force requires possession of the AEAD bundle itself. Documented in
// docs/provisioning_design.md amendments section.
func GeneratePairingCode() (string, error) {
	raw := make([]byte, codeChars)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("provision: code randomness: %w", err)
	}
	var b strings.Builder
	for i, r := range raw {
		if i > 0 && i%codeGroupLen == 0 {
			b.WriteByte(groupSeparator)
		}
		b.WriteByte(codeAlphabet[int(r)%len(codeAlphabet)])
	}
	return b.String(), nil
}

// NormalizePairingCode uppercases, strips separators/whitespace, maps common
// look-alike characters onto the canonical alphabet (O->0, I->1, L->1), and
// validates every remaining character.
func NormalizePairingCode(input string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(input)) {
		switch r {
		case groupSeparator, ' ', '.':
			continue
		case 'O':
			b.WriteByte('0')
		case 'I', 'L':
			b.WriteByte('1')
		default:
			if !strings.ContainsRune(codeAlphabet, r) {
				return "", fmt.Errorf("%w: %q", errCodeChar, r)
			}
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "", ErrEmptyPairing
	}
	return out, nil
}

// CodesMatch compares two normalized codes in constant time.
func CodesMatch(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// DisplayGroup re-groups a normalized code into XXXX-XXXX for output.
func DisplayGroup(normalized string) (string, error) {
	if len(normalized) != codeChars {
		return "", fmt.Errorf("provision: expected %d characters, got %d", codeChars, len(normalized))
	}
	for _, r := range normalized {
		if !strings.ContainsRune(codeAlphabet, r) {
			return "", errCodeChar
		}
	}
	return normalized[:codeGroupLen] + string(groupSeparator) + normalized[codeGroupLen:], nil
}
