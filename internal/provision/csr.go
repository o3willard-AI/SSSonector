package provision

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

// GenerateKeyAndCSR creates the client private key locally (never leaves the
// machine) plus a signing request carrying the subject name. The key is
// written to <dir>/client.key with restrictive permissions; the CSR is
// returned as PEM ready to submit for signing.
func GenerateKeyAndCSR(dir, commonName string) (csrPEM string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("provision: generate key: %w", err)
	}
	tpl := x509.CertificateRequest{
		Subject:            pkix.Name{CommonName: commonName},
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &tpl, key)
	if err != nil {
		return "", fmt.Errorf("provision: create CSR: %w", err)
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("provision: mkdir: %w", err)
	}
	keyPath := filepath.Join(dir, "client.key")
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return "", fmt.Errorf("provision: write key: %w", err)
	}
	if err := RestrictKeyFile(keyPath); err != nil {
		return "", fmt.Errorf("provision: harden key: %w", err)
	}

	csrOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
	return string(csrOut), nil
}

// SignCSR issues a client certificate from a PKCS#10 request using the CA
// material in caDir. The CSR controls Subject, public key and SANs; EKU is
// forced to ClientAuth and validity matches the standard enrollment window.
func SignCSR(csrPEM, caDir string, duration time.Duration) (leafPEM string, err error) {
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return "", errors.New("provision: body is not a CERTIFICATE REQUEST")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("provision: parse CSR: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return "", fmt.Errorf("provision: CSR signature invalid: %w", err)
	}

	caCert, caKey, err := generator.LoadCA(caDir)
	if err != nil {
		return "", err
	}
	if duration <= 0 {
		duration = 180 * 24 * time.Hour
	}
	serial, serr := randomSerial()
	if serr != nil {
		return "", serr
	}
	tpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               csr.Subject,
		PublicKey:             csr.PublicKey,
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(duration),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IPAddresses:           csr.IPAddresses,
		DNSNames:              csr.DNSNames,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, caCert, csr.PublicKey, caKey)
	if err != nil {
		return "", fmt.Errorf("provision: sign certificate: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	n, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("provision: serial: %w", err)
	}
	return n, nil
}

// SplitLeafAndCA separates a redemption response containing the freshly
// signed leaf certificate followed by the CA certificate into its parts.
func SplitLeafAndCA(body []byte) (leafPEM, caPEM string, err error) {
	s := string(body)
	const marker = "-----END CERTIFICATE-----"
	first := strings.Index(s, marker)
	if first < 0 {
		return "", "", errors.New("provision: no certificate in response")
	}
	first += len(marker)
	rest := strings.TrimSpace(s[first:])
	if rest == "" {
		return "", "", errors.New("provision: response missing CA certificate")
	}
	return s[:first] + "\n", rest + "\n", nil
}
