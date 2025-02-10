package cert

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CRLServer manages certificate revocation lists
type CRLServer struct {
	manager CertificateManager
	logger  *zap.Logger

	mu           sync.RWMutex
	revokedCerts map[string]time.Time
}

// NewCRLServer creates a new CRL server
func NewCRLServer(manager CertificateManager, logger *zap.Logger) *CRLServer {
	return &CRLServer{
		manager:      manager,
		logger:       logger,
		revokedCerts: make(map[string]time.Time),
	}
}

// ServeHTTP implements http.Handler
func (s *CRLServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	crl, err := s.generateCRL()
	if err != nil {
		s.logger.Error("Failed to generate CRL",
			zap.Error(err),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pkix-crl")
	w.Write(crl)
}

// generateCRL creates a new CRL
func (s *CRLServer) generateCRL() ([]byte, error) {
	s.mu.RLock()
	revokedCerts := make([]pkix.RevokedCertificate, 0, len(s.revokedCerts))
	for serialStr, revokedAt := range s.revokedCerts {
		serial := new(big.Int)
		if _, ok := serial.SetString(serialStr, 10); !ok {
			s.logger.Error("Invalid serial number in revoked certs map",
				zap.String("serial", serialStr),
			)
			continue
		}

		revokedCerts = append(revokedCerts, pkix.RevokedCertificate{
			SerialNumber:   serial,
			RevocationTime: revokedAt,
		})
	}
	s.mu.RUnlock()

	// Get the CA certificate
	caCerts, err := s.manager.GetCertificateStore().GetByType(CertTypeCA)
	if err != nil {
		return nil, fmt.Errorf("failed to get CA certificate: %v", err)
	}
	if len(caCerts) == 0 {
		return nil, fmt.Errorf("no CA certificate found")
	}
	ca := caCerts[0]

	// Create CRL template
	now := time.Now()
	template := &x509.RevocationList{
		SignatureAlgorithm:  x509.SHA256WithRSA,
		RevokedCertificates: revokedCerts,
		Number:              big.NewInt(time.Now().Unix()),
		ThisUpdate:          now,
		NextUpdate:          now.Add(24 * time.Hour),
	}

	// Sign CRL
	return x509.CreateRevocationList(
		rand.Reader,
		template,
		ca.X509,
		ca.PrivateKey,
	)
}

// RevokeHandler handles certificate revocation requests
func (s *CRLServer) RevokeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serialNumber := r.FormValue("serial")
	if serialNumber == "" {
		http.Error(w, "Missing serial number", http.StatusBadRequest)
		return
	}

	if err := s.manager.Revoke(serialNumber); err != nil {
		s.logger.Error("Failed to revoke certificate",
			zap.String("serial", serialNumber),
			zap.Error(err),
		)
		http.Error(w, "Failed to revoke certificate", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.revokedCerts[serialNumber] = time.Now()
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}
