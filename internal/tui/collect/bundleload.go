package collect

import (
	"archive/tar"
	"compress/gzip"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadedBundle is the result of opening a client bundle (WI 5.5): the
// unpacked cert material in a staging dir plus the prefill from the
// skeleton. Nothing is written to the host config by loading — the wizard
// validates first (the write goes through the WI 5.3 pipeline).
type LoadedBundle struct {
	// StagingDir holds ca.crt, client.crt, client.key unpacked from the
	// bundle (os.MkdirTemp; the wizard's write path relocates them).
	StagingDir string
	// Prefill carries the skeleton values for the wizard fields.
	Prefill ClientBundleSkeleton
	// SkeletonRaw is the skeleton.yaml text as shipped (for display).
	SkeletonRaw string
	// CAPath / ClientCertPath / ClientKeyPath are the unpacked files.
	CAPath         string
	ClientCertPath string
	ClientKeyPath  string
}

// LoadBundle opens a client bundle .tgz produced by [g] (WI 5.4): unpacks
// the cert material into a staging dir and parses skeleton.yaml for the
// wizard prefill. Fail-closed: a bundle missing any required member, or
// with an unparseable skeleton, errors and stages nothing.
func LoadBundle(tgzPath string, tempDir func() string) (LoadedBundle, error) {
	lb := LoadedBundle{}
	if strings.TrimSpace(tgzPath) == "" {
		return lb, fmt.Errorf("bundle load: empty bundle path")
	}
	f, err := os.Open(tgzPath)
	if err != nil {
		return lb, fmt.Errorf("bundle load: open %s: %w", tgzPath, err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return lb, fmt.Errorf("bundle load: %s is not a gzip tarball: %w", tgzPath, err)
	}
	tr := tar.NewReader(gz)

	members := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return lb, fmt.Errorf("bundle load: read %s: %w", tgzPath, err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return lb, fmt.Errorf("bundle load: read member %s: %w", hdr.Name, err)
		}
		members[hdr.Name] = data
	}

	// Fail-closed: all four required members must be present.
	for _, name := range []string{"ca.crt", "client.crt", "client.key", "skeleton.yaml"} {
		if _, ok := members[name]; !ok {
			return lb, fmt.Errorf("bundle load: %s is missing member %q (required: ca.crt, client.crt, client.key, skeleton.yaml)", tgzPath, name)
		}
	}

	skel, err := parseSkeletonYAML(string(members["skeleton.yaml"]))
	if err != nil {
		return lb, fmt.Errorf("bundle load: %w", err)
	}
	// Fail-closed preflight: a skeleton missing any prefill field cannot
	// drive the wizard (the fields would be silently blank).
	if strings.TrimSpace(skel.Instance) == "" || strings.TrimSpace(skel.ServerHost) == "" ||
		skel.ServerPort <= 0 || strings.TrimSpace(skel.TunAddress) == "" {
		return lb, fmt.Errorf("bundle load: skeleton.yaml is incomplete (instance=%q server_host=%q server_port=%d tun_address=%q)",
			skel.Instance, skel.ServerHost, skel.ServerPort, skel.TunAddress)
	}

	staging, err := os.MkdirTemp(tempDir(), "sssonector-bundle-")
	if err != nil {
		return lb, fmt.Errorf("bundle load: staging dir: %w", err)
	}
	for _, name := range []string{"ca.crt", "client.crt", "client.key"} {
		p := filepath.Join(staging, name)
		// 0600 for the key; 0644 is fine for the certs but uniform 0600 is
		// simpler and safe.
		if err := os.WriteFile(p, members[name], 0o600); err != nil {
			return lb, fmt.Errorf("bundle load: stage %s: %w", name, err)
		}
	}
	return LoadedBundle{
		StagingDir:     staging,
		Prefill:        skel,
		SkeletonRaw:    string(members["skeleton.yaml"]),
		CAPath:         filepath.Join(staging, "ca.crt"),
		ClientCertPath: filepath.Join(staging, "client.crt"),
		ClientKeyPath:  filepath.Join(staging, "client.key"),
	}, nil
}

// parseSkeletonYAML parses the skeleton's flat key: value format (see
// renderSkeletonYAML). Not general YAML — the bundle format is fixed and
// self-produced; parse errors are fail-closed.
func parseSkeletonYAML(text string) (ClientBundleSkeleton, error) {
	skel := ClientBundleSkeleton{}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			return skel, fmt.Errorf("skeleton.yaml: malformed line %q", raw)
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "instance":
			skel.Instance = v
		case "server_host":
			skel.ServerHost = v
		case "server_port":
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
				return skel, fmt.Errorf("skeleton.yaml: server_port %q is not a number", v)
			}
			skel.ServerPort = n
		case "tun_address":
			skel.TunAddress = v
		default:
			return skel, fmt.Errorf("skeleton.yaml: unknown key %q", strings.TrimSpace(k))
		}
	}
	return skel, nil
}

// VerifyClientChainAgainstCA is the WI 5.5 pre-flight: the loaded CA must
// verify the loaded client cert chain (real x509). A mismatch BLOCKS the
// wizard rather than failing later in the daemon log.
func VerifyClientChainAgainstCA(caPath, clientCertPath string) error {
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		return fmt.Errorf("chain pre-flight: read CA: %w", err)
	}
	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil {
		return fmt.Errorf("chain pre-flight: no PEM block in CA %s", caPath)
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return fmt.Errorf("chain pre-flight: parse CA: %w", err)
	}
	cliPEM, err := os.ReadFile(clientCertPath)
	if err != nil {
		return fmt.Errorf("chain pre-flight: read client cert: %w", err)
	}
	cliBlock, _ := pem.Decode(cliPEM)
	if cliBlock == nil {
		return fmt.Errorf("chain pre-flight: no PEM block in client cert %s", clientCertPath)
	}
	cliCert, err := x509.ParseCertificate(cliBlock.Bytes)
	if err != nil {
		return fmt.Errorf("chain pre-flight: parse client cert: %w", err)
	}
	if err := cliCert.CheckSignatureFrom(caCert); err != nil {
		return fmt.Errorf("chain pre-flight: CA %s does NOT verify client cert %s: %w", caPath, clientCertPath, err)
	}
	return nil
}
