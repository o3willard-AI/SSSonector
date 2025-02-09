package cert

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// RotationPolicy defines when certificates should be rotated
type RotationPolicy struct {
	// MinimumValidityDuration is the minimum remaining validity period
	// before a certificate should be rotated
	MinimumValidityDuration time.Duration

	// MaximumValidityDuration is the maximum validity period for new certificates
	MaximumValidityDuration time.Duration

	// RenewBeforeDuration is how long before expiration to start renewal
	RenewBeforeDuration time.Duration

	// RetryInterval is how often to retry failed rotations
	RetryInterval time.Duration

	// Types specifies which certificate types to rotate
	Types []CertificateType
}

// DefaultRotationPolicy returns the default rotation policy
func DefaultRotationPolicy() *RotationPolicy {
	return &RotationPolicy{
		MinimumValidityDuration: 30 * 24 * time.Hour,  // 30 days
		MaximumValidityDuration: 365 * 24 * time.Hour, // 1 year
		RenewBeforeDuration:     14 * 24 * time.Hour,  // 14 days
		RetryInterval:           1 * time.Hour,
		Types: []CertificateType{
			CertTypeServer,
			CertTypeClient,
			CertTypeIntermediate,
		},
	}
}

// Rotator implements CertificateRotator interface
type Rotator struct {
	manager CertificateManager
	policy  *RotationPolicy
	logger  *zap.Logger
}

// NewRotator creates a new certificate rotator
func NewRotator(manager CertificateManager, policy *RotationPolicy, logger *zap.Logger) *Rotator {
	if policy == nil {
		policy = DefaultRotationPolicy()
	}

	return &Rotator{
		manager: manager,
		policy:  policy,
		logger:  logger,
	}
}

// ShouldRotate determines if a certificate should be rotated
func (r *Rotator) ShouldRotate(cert *Certificate) bool {
	// Check if certificate type is included in policy
	included := false
	for _, t := range r.policy.Types {
		if cert.Type == t {
			included = true
			break
		}
	}
	if !included {
		return false
	}

	// Check if certificate is valid
	if cert.Status != CertStatusValid {
		return true
	}

	// Check remaining validity period
	now := time.Now()
	remaining := cert.X509.NotAfter.Sub(now)

	// Check if certificate is expired or close to expiring
	if remaining <= 0 {
		return true
	}
	if remaining < r.policy.MinimumValidityDuration {
		return true
	}

	// Check if certificate exceeds maximum validity period
	validity := cert.X509.NotAfter.Sub(cert.X509.NotBefore)
	if validity > r.policy.MaximumValidityDuration {
		return true
	}

	return false
}

// Rotate rotates a certificate
func (r *Rotator) Rotate(cert *Certificate) (*Certificate, error) {
	// Get issuer certificate
	issuer, err := r.manager.GetCertificateStore().Load(cert.IssuerSerial)
	if err != nil {
		return nil, fmt.Errorf("failed to load issuer certificate: %w", err)
	}

	// Create certificate request
	req := &CertificateRequest{
		Type: cert.Type,
		Subject: struct {
			CommonName         string
			Organization       []string
			OrganizationalUnit []string
			Country            []string
			Province           []string
			Locality           []string
		}{
			CommonName:         cert.X509.Subject.CommonName,
			Organization:       cert.X509.Subject.Organization,
			OrganizationalUnit: cert.X509.Subject.OrganizationalUnit,
			Country:            cert.X509.Subject.Country,
			Province:           cert.X509.Subject.Province,
			Locality:           cert.X509.Subject.Locality,
		},
		SANs:        cert.SANs,
		KeyUsage:    cert.KeyUsage,
		ExtKeyUsage: cert.ExtKeyUsage,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(r.policy.MaximumValidityDuration),
		KeySize:     384, // Use P-384 curve
		Metadata:    cert.Metadata,
	}

	// Create new certificate
	var newCert *Certificate
	var createErr error

	switch cert.Type {
	case CertTypeServer:
		newCert, createErr = r.manager.CreateServer(req, issuer)
	case CertTypeClient:
		newCert, createErr = r.manager.CreateClient(req, issuer)
	case CertTypeIntermediate:
		newCert, createErr = r.manager.CreateIntermediate(req, issuer)
	default:
		return nil, fmt.Errorf("unsupported certificate type: %v", cert.Type)
	}

	if createErr != nil {
		return nil, fmt.Errorf("failed to create new certificate: %w", createErr)
	}

	// Revoke old certificate
	if err := r.manager.Revoke(cert.SerialNumber, "superseded by new certificate"); err != nil {
		r.logger.Error("Failed to revoke old certificate",
			zap.String("serial_number", cert.SerialNumber),
			zap.Error(err),
		)
	}

	return newCert, nil
}

// RotateAll rotates all certificates of a given type
func (r *Rotator) RotateAll(certType CertificateType) error {
	// Get all certificates of the specified type
	certs, err := r.manager.GetCertificateStore().ListByType(certType)
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	// Check each certificate
	for _, cert := range certs {
		if r.ShouldRotate(cert) {
			r.logger.Info("Rotating certificate",
				zap.String("serial_number", cert.SerialNumber),
				zap.String("subject", cert.X509.Subject.CommonName),
			)

			if _, err := r.Rotate(cert); err != nil {
				r.logger.Error("Failed to rotate certificate",
					zap.String("serial_number", cert.SerialNumber),
					zap.Error(err),
				)
				continue
			}

			r.logger.Info("Certificate rotated successfully",
				zap.String("serial_number", cert.SerialNumber),
			)
		}
	}

	return nil
}
