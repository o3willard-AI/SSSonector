package provision

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func mustSeal(t *testing.T, payload *PairingPayload, code string) []byte {
	t.Helper()
	b, err := Seal(payload, code)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	return b
}

func validPayload() *PairingPayload {
	return &PairingPayload{
		Role:              "client",
		ServerAddr:        "192.0.2.10",
		ServerPort:        18443,
		FacadeTokenSecret: "e2e-test-secret-do-not-log",
		CACertPEM:         "-----BEGIN CERTIFICATE-----\nCA\n-----END CERTIFICATE-----\n",
		ClientCertPEM:     "-----BEGIN CERTIFICATE-----\nLEAF\n-----END CERTIFICATE-----\n",
		ClientKeyPEM:      "-----BEGIN PRIVATE KEY-----\nKEY\n-----END PRIVATE KEY-----\n",
		CreatedAtRFC3339:  "2026-08-23T00:00:00Z",
		Name:              "enroll-a",
		FingerprintOfCA:   "aa55",
	}
}

// TestRoundTrip is the canonical create->apply equivalence proof.
func TestRoundTrip(t *testing.T) {
	code, err := GeneratePairingCode()
	if err != nil {
		t.Fatalf("GeneratePairingCode: %v", err)
	}
	env := mustSeal(t, validPayload(), code)

	got, err := Open(env, code)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	want := validPayload()
	if got.Role != want.Role || got.ServerAddr != want.ServerAddr ||
		got.FacadeTokenSecret != want.FacadeTokenSecret ||
		got.ClientKeyPEM != want.ClientKeyPEM ||
		got.FingerprintOfCA != want.FingerprintOfCA {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}

	norm, err := NormalizePairingCode(code)
	if err != nil {
		t.Fatalf("Normalize(code): %v", err)
	}
	if _, err := Open(env, norm); err != nil {
		t.Errorf("normalized code should open the same bundle: %v", err)
	}
}

func TestWrongCodeRejected(t *testing.T) {
	env := mustSeal(t, validPayload(), "AAAA-BBBB")

	if _, err := Open(env, "CCCC-DDDD"); !errors.Is(err, ErrAuthFailed) {
		t.Errorf("wrong code: got %v, want ErrAuthFailed", err)
	}
	// Near-miss (single character different) must also fail — this is the
	// mutation-sensitive case for the KDF/AEAD path.
	if _, err := Open(env, "AAAA-BBBC"); !errors.Is(err, ErrAuthFailed) {
		t.Errorf("near-miss code: got %v, want ErrAuthFailed", err)
	}
}

func TestTamperedCiphertextRejected(t *testing.T) {
	env := mustSeal(t, validPayload(), "AAAA-BBBB")
	mutations := map[string]func([]byte) []byte{
		"flip-last-byte":    func(b []byte) []byte { c := bytes.Clone(b); c[len(c)-1] ^= 0x01; return c },
		"flip-ct-middle":    func(b []byte) []byte { c := bytes.Clone(b); c[len(b)-40] ^= 0x80; return c },
		"flip-nonce":        func(b []byte) []byte { c := bytes.Clone(b); i := len(b) - 60; c[i] ^= 0xFF; return c },
		"truncate":          func(b []byte) []byte { return b[:len(b)-1] },
		"bad-magic":         func(b []byte) []byte { c := bytes.Clone(b); copy(c, "XSP1"); return c },
		"unknown-version":   func(b []byte) []byte { c := bytes.Clone(b); c[4] = 9; return c },
		"unknown-kdf-id":    func(b []byte) []byte { c := bytes.Clone(b); c[5] = 0; c[6] = 99; return c },
		"empty-payload-nil": func(b []byte) []byte { return nil },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			_, err := Open(mutate(bytes.Clone(env)), "AAAA-BBBB")
			if err == nil {
				t.Fatal("mutated envelope accepted")
			}
			if errors.Is(err, ErrBadMagic) || errors.Is(err, ErrBadVersion) ||
				errors.Is(err, ErrBadKDF) || errors.Is(err, ErrTruncated) {
				return // precise structural rejection
			}
			if !errors.Is(err, ErrAuthFailed) && !errors.Is(err, ErrTruncated) &&
				err.Error() != "provision: unmarshal payload: unexpected end of JSON input" {
				t.Errorf("unexpected error class: %v", err)
			}
		})
	}
}

func TestEmptyCodeFailClosed(t *testing.T) {
	if _, err := Seal(validPayload(), ""); !errors.Is(err, ErrEmptyPairing) {
		t.Errorf("Seal empty code: %v", err)
	}
	if _, err := Open(mustSeal(t, validPayload(), "AAAA-BBBB"), ""); !errors.Is(err, ErrEmptyPairing) {
		t.Errorf("Open empty code: %v", err)
	}
}

func TestPayloadValidationFailClosed(t *testing.T) {
	cases := map[string]*PairingPayload{
		"bad-role":     {Role: "peer", ServerAddr: "a", ServerPort: 1, FacadeTokenSecret: "s", CACertPEM: "c", FingerprintOfCA: "f"},
		"empty-secret": {Role: "client", ServerAddr: "a", ServerPort: 1, CACertPEM: "c", FingerprintOfCA: "f"},
		"missing-ca":   {Role: "client", ServerAddr: "a", ServerPort: 1, FacadeTokenSecret: "s", FingerprintOfCA: "f"},
		"missing-fp":   {Role: "client", ServerAddr: "a", ServerPort: 1, FacadeTokenSecret: "s", CACertPEM: "c"},
		"no-addr":      {Role: "client", ServerPort: 1, FacadeTokenSecret: "s", CACertPEM: "c", FingerprintOfCA: "f"},
	}
	for name, p := range cases {
		if _, err := Seal(p, "AAAA-BBBB"); err == nil {
			t.Errorf("%s: Seal accepted invalid payload", name)
		}
	}
}

func TestCodesMatchConstantTimeContract(t *testing.T) {
	a, _ := NormalizePairingCode("7F3K-9QPD")
	b, _ := NormalizePairingCode("7f3k9qpd")
	if !CodesMatch(a, b) {
		t.Error("normalize+match should be case/separator insensitive")
	}
	if CodesMatch(a, "7F3K-9QPE") {
		t.Error("different codes matched")
	}
}

func TestNormalizeRejectsGarbage(t *testing.T) {
	if _, err := NormalizePairingCode("ABCD-EFGH"); err != nil {
		t.Errorf("ABCD-EFGH uses only alphabet chars, should normalize: %v", err)
	}
	if _, err := NormalizePairingCode("7F3K-9QPU"); err == nil {
		t.Error("U is excluded from the alphabet and should not silently map")
	}
	for _, bad := range []string{"7F3K-9QP@", "7F3K 9QPD!"} {
		if bad == "" {
			continue
		}
		if _, err := NormalizePairingCode(bad); err == nil {
			t.Errorf("expected rejection for %q", bad)
		}
	}
	if _, err := NormalizePairingCode("7F3K-OQPD"); err != nil { // O maps to 0
		t.Errorf("confusable O should normalize to 0: %v", err)
	}
}

func TestGeneratedCodesAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code, err := GeneratePairingCode()
		if err != nil {
			t.Fatalf("GeneratePairingCode: %v", err)
		}
		if len(code) != 9 || code[4] != '-' {
			t.Fatalf("code %q malformed", code)
		}
		norm, err := NormalizePairingCode(code)
		if err != nil || norm != strings.ReplaceAll(code, "-", "") {
			t.Fatalf("generated code fails its own normalizer: %q (%v)", code, err)
		}
		if seen[code] {
			t.Fatalf("duplicate code generated: %s", code)
		}
		seen[code] = true
	}
}
