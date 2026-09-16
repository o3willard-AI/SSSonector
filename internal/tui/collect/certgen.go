package collect

import (
	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

// certGenCertificates is the production cert-generator seam — a fresh CA +
// server leaf (plus a client leaf used by the WI 5.4 bundle [g]) into the
// given cert dir, via the SAME path `provision create` uses. Isolated here
// so netcheck.go can stay free of the generator import (and tests fake it).
func certGenCertificates(certDir string, serverIPs ...string) error {
	return generator.GenerateCertificates(certDir, serverIPs...)
}