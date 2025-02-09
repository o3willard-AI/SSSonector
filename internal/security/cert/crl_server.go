package cert

import (
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CRLServer provides certificate revocation list services
type CRLServer struct {
	manager CertificateManager
	logger  *zap.Logger
	mu      sync.RWMutex

	// CRL update settings
	updateInterval time.Duration
	nextUpdate     time.Time

	// Current CRL
	crl       *x509.RevocationList
	crlPEM    []byte
	revokedBy map[string]string // serial number -> revoked by

	// HTTP server
	server *http.Server
}

// NewCRLServer creates a new CRL server
func NewCRLServer(manager CertificateManager, logger *zap.Logger) *CRLServer {
	return &CRLServer{
		manager:        manager,
		logger:         logger,
		updateInterval: 24 * time.Hour,
		revokedBy:      make(map[string]string),
	}
}

// Start starts the CRL server
func (s *CRLServer) Start(addr string, signer crypto.Signer) error {
	// Generate initial CRL
	if err := s.generateCRL(signer); err != nil {
		return fmt.Errorf("failed to generate initial CRL: %w", err)
	}

	// Set up HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/crl", s.handleCRL)
	mux.HandleFunc("/revoke", s.handleRevoke)
	mux.HandleFunc("/status", s.handleStatus)

	// Create HTTP server
	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start CRL update goroutine
	go s.updateLoop(signer)

	// Start HTTP server
	return s.server.ListenAndServe()
}

// Stop stops the CRL server
func (s *CRLServer) Stop() error {
	return s.server.Close()
}

// generateCRL generates a new CRL
func (s *CRLServer) generateCRL(signer crypto.Signer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get CA certificate
	caCerts, err := s.manager.GetCertificateStore().ListByType(CertTypeCA)
	if err != nil {
		return fmt.Errorf("failed to get CA certificate: %w", err)
	}
	if len(caCerts) == 0 {
		return fmt.Errorf("no CA certificate found")
	}
	ca := caCerts[0]

	// Get all revoked certificates
	revokedCerts, err := s.manager.GetCertificateStore().ListByStatus(CertStatusRevoked)
	if err != nil {
		return fmt.Errorf("failed to list revoked certificates: %w", err)
	}

	// Create revoked certificate list
	revokedList := make([]pkix.RevokedCertificate, 0, len(revokedCerts))
	for _, cert := range revokedCerts {
		serial := new(big.Int)
		serial.SetString(cert.SerialNumber, 16)
		revokedList = append(revokedList, pkix.RevokedCertificate{
			SerialNumber:   serial,
			RevocationTime: *cert.RevokedAt,
		})
	}

	// Create CRL template
	now := time.Now()
	template := &x509.RevocationList{
		SignatureAlgorithm:  x509.ECDSAWithSHA384,
		RevokedCertificates: revokedList,
		Number:              big.NewInt(time.Now().Unix()),
		ThisUpdate:          now,
		NextUpdate:          now.Add(s.updateInterval),
	}

	// Sign CRL
	crlBytes, err := x509.CreateRevocationList(nil, template, ca.X509, signer)
	if err != nil {
		return fmt.Errorf("failed to create CRL: %w", err)
	}

	// Encode CRL in PEM format
	pemBlock := &pem.Block{
		Type:  "X509 CRL",
		Bytes: crlBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)

	// Update server state
	s.crl = template
	s.crlPEM = pemBytes
	s.nextUpdate = template.NextUpdate

	s.logger.Info("Generated new CRL",
		zap.Int("revoked_count", len(revokedList)),
		zap.Time("next_update", template.NextUpdate),
	)

	return nil
}

// updateLoop periodically updates the CRL
func (s *CRLServer) updateLoop(signer crypto.Signer) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if time.Now().After(s.nextUpdate) {
			if err := s.generateCRL(signer); err != nil {
				s.logger.Error("Failed to update CRL", zap.Error(err))
			}
		}
	}
}

// handleCRL serves the current CRL
func (s *CRLServer) handleCRL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	crlPEM := s.crlPEM
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Write(crlPEM)
}

// RevokeRequest represents a certificate revocation request
type RevokeRequest struct {
	SerialNumber string `json:"serial_number"`
	Reason       string `json:"reason"`
	RevokedBy    string `json:"revoked_by"`
}

// handleRevoke handles certificate revocation requests
func (s *CRLServer) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req RevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Revoke certificate
	if err := s.manager.Revoke(req.SerialNumber, req.Reason); err != nil {
		s.logger.Error("Failed to revoke certificate",
			zap.String("serial_number", req.SerialNumber),
			zap.Error(err),
		)
		http.Error(w, "Failed to revoke certificate", http.StatusInternalServerError)
		return
	}

	// Record who revoked the certificate
	s.mu.Lock()
	s.revokedBy[req.SerialNumber] = req.RevokedBy
	s.mu.Unlock()

	s.logger.Info("Certificate revoked",
		zap.String("serial_number", req.SerialNumber),
		zap.String("reason", req.Reason),
		zap.String("revoked_by", req.RevokedBy),
	)

	w.WriteHeader(http.StatusOK)
}

// StatusResponse represents the CRL server status
type StatusResponse struct {
	RevokedCount int       `json:"revoked_count"`
	LastUpdate   time.Time `json:"last_update"`
	NextUpdate   time.Time `json:"next_update"`
}

// handleStatus serves the CRL server status
func (s *CRLServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	status := StatusResponse{
		RevokedCount: len(s.crl.RevokedCertificates),
		LastUpdate:   s.crl.ThisUpdate,
		NextUpdate:   s.crl.NextUpdate,
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
