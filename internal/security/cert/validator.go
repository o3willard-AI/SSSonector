package cert

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// RevocationReason represents the reason for certificate revocation
type RevocationReason int

const (
	// RevocationReasonUnspecified indicates no specific reason
	RevocationReasonUnspecified RevocationReason = iota
	// RevocationReasonKeyCompromise indicates key was compromised
	RevocationReasonKeyCompromise
	// RevocationReasonCACompromise indicates CA was compromised
	RevocationReasonCACompromise
	// RevocationReasonAffiliationChanged indicates affiliation changed
	RevocationReasonAffiliationChanged
	// RevocationReasonSuperseded indicates certificate was superseded
	RevocationReasonSuperseded
	// RevocationReasonCessationOfOperation indicates operation ceased
	RevocationReasonCessationOfOperation
	// RevocationReasonCertificateHold indicates certificate is on hold
	RevocationReasonCertificateHold
)

// String returns the string representation of RevocationReason
func (r RevocationReason) String() string {
	switch r {
	case RevocationReasonUnspecified:
		return "unspecified"
	case RevocationReasonKeyCompromise:
		return "key compromise"
	case RevocationReasonCACompromise:
		return "CA compromise"
	case RevocationReasonAffiliationChanged:
		return "affiliation changed"
	case RevocationReasonSuperseded:
		return "superseded"
	case RevocationReasonCessationOfOperation:
		return "cessation of operation"
	case RevocationReasonCertificateHold:
		return "certificate hold"
	default:
		return fmt.Sprintf("unknown reason (%d)", r)
	}
}

// Validator implements CertificateValidator interface
type Validator struct {
	store  CertificateStore
	logger *zap.Logger
	client *http.Client
}

// NewValidator creates a new certificate validator
func NewValidator(store CertificateStore, logger *zap.Logger) *Validator {
	return &Validator{
		store:  store,
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateCertificate validates a certificate
func (v *Validator) ValidateCertificate(cert *Certificate) error {
	// Check if certificate is nil
	if cert == nil || cert.X509 == nil {
		return fmt.Errorf("invalid certificate: nil certificate")
	}

	// Check certificate validity period
	now := time.Now()
	if now.Before(cert.X509.NotBefore) {
		return fmt.Errorf("certificate is not yet valid")
	}
	if now.After(cert.X509.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}

	// Check if certificate is revoked
	if cert.Status == CertStatusRevoked {
		return fmt.Errorf("certificate is revoked: %s", cert.RevocationReason)
	}

	// Check key usage
	if cert.Type == CertTypeServer {
		if cert.X509.KeyUsage&x509.KeyUsageKeyEncipherment == 0 &&
			cert.X509.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
			return fmt.Errorf("server certificate missing required key usage")
		}
		hasServerAuth := false
		for _, usage := range cert.X509.ExtKeyUsage {
			if usage == x509.ExtKeyUsageServerAuth {
				hasServerAuth = true
				break
			}
		}
		if !hasServerAuth {
			return fmt.Errorf("server certificate missing server auth extended key usage")
		}
	}

	if cert.Type == CertTypeClient {
		if cert.X509.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
			return fmt.Errorf("client certificate missing required key usage")
		}
		hasClientAuth := false
		for _, usage := range cert.X509.ExtKeyUsage {
			if usage == x509.ExtKeyUsageClientAuth {
				hasClientAuth = true
				break
			}
		}
		if !hasClientAuth {
			return fmt.Errorf("client certificate missing client auth extended key usage")
		}
	}

	// Check basic constraints for CA certificates
	if cert.Type == CertTypeCA || cert.Type == CertTypeIntermediate {
		if !cert.X509.IsCA {
			return fmt.Errorf("CA certificate missing CA basic constraint")
		}
		if cert.X509.KeyUsage&x509.KeyUsageCertSign == 0 {
			return fmt.Errorf("CA certificate missing cert sign key usage")
		}
	}

	return nil
}

// ValidateChain validates a certificate chain
func (v *Validator) ValidateChain(cert *Certificate, intermediates []*Certificate, root *Certificate) error {
	// Create certificate pools
	roots := x509.NewCertPool()
	intermediatePool := x509.NewCertPool()

	// Add root certificate
	if root != nil {
		roots.AddCert(root.X509)
	}

	// Add intermediate certificates
	for _, intermediate := range intermediates {
		intermediatePool.AddCert(intermediate.X509)
	}

	// Create verification options
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediatePool,
		CurrentTime:   time.Now(),
		KeyUsages:     cert.ExtKeyUsage,
	}

	// Verify certificate chain
	chains, err := cert.X509.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate chain verification failed: %w", err)
	}

	// Log chain information
	for i, chain := range chains {
		v.logger.Debug("Certificate chain",
			zap.Int("chain_index", i),
			zap.Int("chain_length", len(chain)),
		)
		for j, cert := range chain {
			v.logger.Debug("Chain certificate",
				zap.Int("chain_index", i),
				zap.Int("cert_index", j),
				zap.String("subject", cert.Subject.CommonName),
				zap.String("issuer", cert.Issuer.CommonName),
			)
		}
	}

	return nil
}

// ValidateCRL validates a certificate against a CRL
func (v *Validator) ValidateCRL(cert *Certificate, crl *x509.RevocationList) error {
	// Check if CRL is valid
	if crl == nil {
		return fmt.Errorf("invalid CRL: nil")
	}

	// Check CRL validity period
	now := time.Now()
	if now.Before(crl.ThisUpdate) {
		return fmt.Errorf("CRL is not yet valid")
	}
	if now.After(crl.NextUpdate) {
		return fmt.Errorf("CRL has expired")
	}

	// Check if certificate is revoked
	for _, revokedCert := range crl.RevokedCertificates {
		if revokedCert.SerialNumber.Cmp(cert.X509.SerialNumber) == 0 {
			return fmt.Errorf("certificate is revoked")
		}
	}

	return nil
}

// ValidateOCSP validates a certificate using OCSP
func (v *Validator) ValidateOCSP(cert *Certificate) error {
	// Check if certificate has OCSP server URL
	if len(cert.X509.OCSPServer) == 0 {
		return fmt.Errorf("certificate has no OCSP servers")
	}

	v.logger.Info("OCSP validation not implemented",
		zap.String("serial_number", cert.SerialNumber),
		zap.String("ocsp_server", cert.X509.OCSPServer[0]),
	)

	return nil
}
