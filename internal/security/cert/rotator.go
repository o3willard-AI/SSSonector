package cert

import (
	"context"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RotationMetrics tracks certificate rotation statistics
type RotationMetrics struct {
	LastRotation     time.Time
	NextRotation     time.Time
	RotationAttempts int64
	RotationErrors   int64
	ValidationErrors int64
}

// CertificateRotator manages automatic certificate rotation
type CertificateRotator struct {
	config  *RotationConfig
	logger  *zap.Logger
	manager CertificateManager

	mu            sync.RWMutex
	current       *Certificate
	previous      *Certificate
	rotationTimer *time.Timer
	stopCh        chan struct{}
	metrics       RotationMetrics
}

// NewCertificateRotator creates a new certificate rotator
func NewCertificateRotator(config *RotationConfig, manager CertificateManager, logger *zap.Logger) *CertificateRotator {
	if config == nil {
		config = DefaultRotationConfig()
	}

	return &CertificateRotator{
		config:  config,
		logger:  logger,
		manager: manager,
		stopCh:  make(chan struct{}),
	}
}

// Start begins the rotation monitoring process
func (r *CertificateRotator) Start(ctx context.Context) error {
	// Load initial certificate
	certPair, err := r.manager.GetCertificateStore().LoadCurrent()
	if err != nil {
		return fmt.Errorf("failed to load initial certificate: %v", err)
	}

	cert := certPair.ToCertificate()
	cert.Type = CertTypeServer // Default to server type, should be determined by usage
	cert.Status = CertStatusValid

	// Validate initial certificate
	if err := r.validateCertificate(cert); err != nil {
		return fmt.Errorf("initial certificate validation failed: %v", err)
	}

	r.mu.Lock()
	r.current = cert
	r.metrics.LastRotation = time.Now()
	r.metrics.NextRotation = cert.X509.NotAfter.Add(-r.config.RenewalWindow)
	r.mu.Unlock()

	// Start rotation monitoring
	go r.monitor(ctx)

	// Trigger initial rotation check
	if err := r.checkRotation(); err != nil {
		r.logger.Warn("Initial rotation check failed",
			zap.Error(err),
		)
	}

	return nil
}

// Stop stops the rotation monitoring process
func (r *CertificateRotator) Stop() {
	close(r.stopCh)
	if r.rotationTimer != nil {
		r.rotationTimer.Stop()
	}
}

// GetCurrent returns the current certificate
func (r *CertificateRotator) GetCurrent() *Certificate {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.current
}

// GetPrevious returns the previous certificate during grace period
func (r *CertificateRotator) GetPrevious() *Certificate {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.previous
}

// GetMetrics returns the current rotation metrics
func (r *CertificateRotator) GetMetrics() RotationMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metrics
}

// monitor periodically checks certificates for rotation
func (r *CertificateRotator) monitor(ctx context.Context) {
	ticker := time.NewTicker(r.config.RotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			if err := r.checkRotation(); err != nil {
				r.mu.Lock()
				r.metrics.RotationErrors++
				r.mu.Unlock()

				r.logger.Error("Certificate rotation check failed",
					zap.Error(err),
					zap.Int64("total_errors", r.metrics.RotationErrors),
				)
			}
		}
	}
}

// validateCertificate performs comprehensive certificate validation
func (r *CertificateRotator) validateCertificate(cert *Certificate) error {
	// Basic validation
	if err := r.manager.Validate(cert); err != nil {
		r.mu.Lock()
		r.metrics.ValidationErrors++
		r.mu.Unlock()
		return fmt.Errorf("certificate validation failed: %v", err)
	}

	store := r.manager.GetCertificateStore()

	// Check revocation status if CRL/OCSP endpoints are configured
	if len(cert.X509.CRLDistributionPoints) > 0 {
		for _, crlDP := range cert.X509.CRLDistributionPoints {
			if err := store.ValidateCRL(cert, &x509.RevocationList{
				ThisUpdate: time.Now(),
				NextUpdate: time.Now().Add(24 * time.Hour),
			}); err != nil {
				r.mu.Lock()
				r.metrics.ValidationErrors++
				r.mu.Unlock()
				return fmt.Errorf("CRL validation failed for %s: %v", crlDP, err)
			}
		}
	}

	if len(cert.X509.OCSPServer) > 0 {
		if err := store.ValidateOCSP(cert); err != nil {
			r.mu.Lock()
			r.metrics.ValidationErrors++
			r.mu.Unlock()
			return fmt.Errorf("OCSP validation failed: %v", err)
		}
	}

	return nil
}

// checkRotation checks if certificate rotation is needed
func (r *CertificateRotator) checkRotation() error {
	r.mu.RLock()
	cert := r.current
	r.mu.RUnlock()

	if cert == nil {
		return fmt.Errorf("no current certificate")
	}

	// Validate current certificate
	if err := r.validateCertificate(cert); err != nil {
		return fmt.Errorf("current certificate validation failed: %v", err)
	}

	// Check if we're within renewal window
	if time.Until(cert.X509.NotAfter) > r.config.RenewalWindow {
		return nil
	}

	r.mu.Lock()
	r.metrics.RotationAttempts++
	r.mu.Unlock()

	// Create certificate request for renewal
	req := &CertificateRequest{
		Type:        cert.Type,
		Subject:     cert.X509.Subject,
		CommonName:  cert.X509.Subject.CommonName,
		DNSNames:    cert.X509.DNSNames,
		IPAddresses: cert.X509.IPAddresses,
		KeyUsage:    cert.X509.KeyUsage,
		ExtKeyUsage: cert.X509.ExtKeyUsage,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(r.config.RenewalWindow * 2),
		KeySize:     r.config.KeySize,
		Metadata:    cert.Metadata,
	}

	// Create new certificate based on type
	var newCert *Certificate
	var err error

	switch cert.Type {
	case CertTypeCA:
		newCert, err = r.manager.CreateCA(req)
	case CertTypeIntermediate:
		newCert, err = r.manager.CreateIntermediate(req, cert)
	case CertTypeServer:
		newCert, err = r.manager.CreateServer(req, cert)
	case CertTypeClient:
		newCert, err = r.manager.CreateClient(req, cert)
	default:
		return fmt.Errorf("unknown certificate type: %v", cert.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create new certificate: %v", err)
	}

	// Validate new certificate before storing
	if err := r.validateCertificate(newCert); err != nil {
		return fmt.Errorf("new certificate validation failed: %v", err)
	}

	// Store new certificate
	if err := r.manager.GetCertificateStore().Store(newCert.ToCertPair()); err != nil {
		return fmt.Errorf("failed to store new certificate: %v", err)
	}

	// Update current and previous certificates
	r.mu.Lock()
	r.previous = r.current
	r.current = newCert
	r.metrics.LastRotation = time.Now()
	r.metrics.NextRotation = newCert.X509.NotAfter.Add(-r.config.RenewalWindow)
	r.mu.Unlock()

	// Notify rotation
	if r.config.OnRotation != nil {
		r.config.OnRotation(cert.X509, newCert.X509)
	}

	r.logger.Info("Certificate rotated successfully",
		zap.String("serial", newCert.SerialNumber),
		zap.Time("new_expiry", newCert.X509.NotAfter),
		zap.Time("next_rotation", r.metrics.NextRotation),
		zap.Int64("total_rotations", r.metrics.RotationAttempts),
	)

	// Schedule cleanup of previous certificate
	time.AfterFunc(r.config.GracePeriod, func() {
		r.mu.Lock()
		r.previous = nil
		r.mu.Unlock()

		// Revoke old certificate after grace period
		if err := r.manager.Revoke(cert.SerialNumber); err != nil {
			r.logger.Error("Failed to revoke old certificate",
				zap.String("serial", cert.SerialNumber),
				zap.Error(err),
			)
		}
	})

	return nil
}
