package collect

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// defaultRotationThreshold matches internal/cert's defaultRotationThreshold
// (manager.go): 30 days. Used when config.auth.cert_rotation.interval is 0.
const defaultRotationThreshold = 30 * 24 * time.Hour

// CertInfo is the typed certificate status for the dashboard's shared
// CERTIFICATE panel. Parsed with stdlib crypto/x509 from the instance's
// config-referenced cert files — no daemon endpoint, no internal/monitor.
type CertInfo struct {
	// Issuer is cert.Issuer.String() of the leaf certificate.
	Issuer string
	// NotAfter is the leaf certificate's expiry.
	NotAfter time.Time
	// DaysRemaining is whole days until NotAfter; negative when expired.
	DaysRemaining int
	// NeedsRotation is true when the certificate expires within the
	// rotation threshold (config.auth.cert_rotation.interval, default
	// 30 days) or is already expired.
	NeedsRotation bool
}

// ReadCertInfo loads and parses the leaf certificate from CertPaths.CertFile
// and reports issuer/expiry/rotation status.
//
// Missing, unreadable, or unparseable files return an explicit error —
// never a panic, never a fabricated issuer (fail-closed; AGENTS.md).
func ReadCertInfo(paths CertPaths, rotationInterval time.Duration, now time.Time) (CertInfo, error) {
	var info CertInfo

	if paths.CertFile == "" {
		return info, fmt.Errorf("cert: no cert_file configured for this instance")
	}
	pemBytes, err := os.ReadFile(paths.CertFile)
	if err != nil {
		return info, fmt.Errorf("cert: read %s: %w", paths.CertFile, err)
	}

	cert, err := parseLeafCert(pemBytes)
	if err != nil {
		return info, fmt.Errorf("cert: %s: %w", paths.CertFile, err)
	}

	if rotationInterval <= 0 {
		rotationInterval = defaultRotationThreshold
	}

	// Effective current time: real clock when now is zero (production),
	// the injected fixed time in tests.
	effectiveNow := now
	if effectiveNow.IsZero() {
		effectiveNow = time.Now()
	}
	remaining := cert.NotAfter.Sub(effectiveNow)
	days := int(remaining / (24 * time.Hour))
	info = CertInfo{
		Issuer:        cert.Issuer.String(),
		NotAfter:      cert.NotAfter,
		DaysRemaining: days,
		NeedsRotation: remaining <= rotationInterval,
	}
	return info, nil
}

// parseLeafCert decodes the first CERTIFICATE block from PEM data using
// stdlib crypto/x509 (the same path internal/cert uses internally; it does
// not export a standalone loader, so x509 is used directly here).
func parseLeafCert(pemBytes []byte) (*x509.Certificate, error) {
	rest := pemBytes
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			return nil, fmt.Errorf("no CERTIFICATE PEM block found")
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("x509 parse: %w", err)
			}
			return cert, nil
		}
		rest = remaining
	}
}
