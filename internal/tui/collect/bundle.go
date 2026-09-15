package collect

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

// ClientBundleSkeleton is the pre-filled client config skeleton carried in
// the bundle (spec §3.3). It is a SKELETON, not a full config: the client
// wizard (WI 5.5) prefills from it and validates the result through the
// real loader before anything is written.
type ClientBundleSkeleton struct {
	// Instance is the server instance the bundle was generated from.
	Instance string
	// ServerHost is the host the client should dial: the server's
	// tunnel.listen_address when it is a specific (dialable) address,
	// otherwise this host's primary non-loopback IPv4 (0.0.0.0/::/empty
	// are never shipped as a host).
	ServerHost string
	// ServerPort is the server's tunnel.listen_port.
	ServerPort int
	// TunAddress is the suggested client TUN address: the server's TUN
	// subnet with the host address incremented by one (server .1 → client
	// .2). Same prefix length as the server.
	TunAddress string
}

// ClientBundle is the outcome of a [g] bundle generation.
type ClientBundle struct {
	// Path is the .tgz on disk.
	Path string
	// SHA256 is the hex-encoded digest of the tarball bytes.
	SHA256 string
	// Skeleton is the skeleton that was packed (exposed for tests/CLI).
	Skeleton ClientBundleSkeleton
}

// BundleGenFunc is the dashboard's injectable seam for the [g] action:
// generate a client bundle for the named instance.
type BundleGenFunc func(paths ConfigPaths, name string) (ClientBundle, error)

// DefaultServerHostFunc resolves this host's primary non-loopback IPv4
// address (used when the server listens on 0.0.0.0/::). Injectable for
// tests via generateClientBundleWith.
func DefaultServerHostFunc() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("resolve server host: %w", err)
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.IsLinkLocalUnicast() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			return ip4.String(), nil
		}
	}
	return "", fmt.Errorf("resolve server host: no non-loopback IPv4 address found")
}

// GenerateClientBundle implements the [g] action (docs/tui.md §3.3): locate
// the focused server instance's cert dir, issue the client leaf if absent
// (reused otherwise — reproducible content, never regenerated per press),
// build the client config skeleton, and pack ca.crt + client.crt +
// client.key + skeleton into a .tgz. The CA's key and the server's
// key NEVER travel in the bundle.
func GenerateClientBundle(paths ConfigPaths, name string) (ClientBundle, error) {
	return generateClientBundleWith(paths, name, DefaultServerHostFunc, os.TempDir)
}

func generateClientBundleWith(paths ConfigPaths, name string, hostFunc func() (string, error), tempDir func() string) (ClientBundle, error) {
	if name == "" {
		return ClientBundle{}, fmt.Errorf("bundle: no instance focused")
	}
	ic, err := ResolveInstanceConfig(paths, name)
	if err != nil {
		return ClientBundle{}, fmt.Errorf("bundle: %w", err)
	}
	if ic.Mode != "server" {
		return ClientBundle{}, fmt.Errorf("bundle: instance %q is mode %q — client bundles are generated for server instances", name, ic.Mode)
	}
	if ic.CertPaths.CAFile == "" {
		return ClientBundle{}, fmt.Errorf("bundle: instance %q has no auth.ca_file — cannot locate its cert dir", name)
	}
	// A ca_file whose file name was blanked (ca_file: "/x/certs/" — the
	// "ca.crt" suffix removed by an edit) is equally unusable: Dir()/Base()
	// would silently resolve to a directory and the bundle would ship the
	// wrong CA or fail obscurely later.
	base := filepath.Base(strings.TrimSpace(ic.CertPaths.CAFile))
	if base == "" || base == "." || base == string(filepath.Separator) || !strings.HasSuffix(base, ".crt") {
		return ClientBundle{}, fmt.Errorf("bundle: instance %q has an unusable auth.ca_file %q (no .crt file name)", name, ic.CertPaths.CAFile)
	}
	certDir := filepath.Dir(ic.CertPaths.CAFile)

	// Issue the client leaf only when absent (idempotent [g]: pressing
	// twice reuses the existing cert, so the bundle content is stable).
	clientCert := filepath.Join(certDir, "client.crt")
	clientKey := filepath.Join(certDir, "client.key")
	_, certErr := os.Stat(clientCert)
	_, keyErr := os.Stat(clientKey)
	if certErr != nil || keyErr != nil {
		if err := generator.IssueClientCert(certDir); err != nil {
			return ClientBundle{}, fmt.Errorf("bundle: issue client cert: %w", err)
		}
	}

	skel, err := buildSkeleton(ic, hostFunc)
	if err != nil {
		return ClientBundle{}, err
	}

	tgz := filepath.Join(tempDir(), fmt.Sprintf("sssonector-%s-client-bundle.tgz", name))
	sha, err := packBundle(tgz, certDir, skel)
	if err != nil {
		return ClientBundle{}, err
	}
	return ClientBundle{Path: tgz, SHA256: sha, Skeleton: skel}, nil
}

// buildSkeleton derives the skeleton from the server's effective config.
// Nothing is invented: every field traces to config values (or a resolved
// host address when the server listens wildcard).
func buildSkeleton(ic InstanceConfig, hostFunc func() (string, error)) (ClientBundleSkeleton, error) {
	host := strings.TrimSpace(ic.ListenAddress)
	switch host {
	case "", "0.0.0.0", "::":
		var err error
		if host, err = hostFunc(); err != nil {
			return ClientBundleSkeleton{}, fmt.Errorf("bundle: %w", err)
		}
	}
	if ic.ListenPort <= 0 {
		return ClientBundleSkeleton{}, fmt.Errorf("bundle: instance %q has no tunnel.listen_port", ic.Name)
	}
	if strings.TrimSpace(ic.TunAddr) == "" {
		return ClientBundleSkeleton{}, fmt.Errorf("bundle: instance %q has no network.address (TUN subnet) — cannot suggest a client address", ic.Name)
	}
	suggestion, err := suggestClientTunAddr(ic.TunAddr)
	if err != nil {
		return ClientBundleSkeleton{}, fmt.Errorf("bundle: %w", err)
	}
	return ClientBundleSkeleton{
		Instance:   ic.Name,
		ServerHost: host,
		ServerPort: ic.ListenPort,
		TunAddress: suggestion,
	}, nil
}

// suggestClientTunAddr increments the server's TUN host address within its
// own subnet (10.77.0.1/24 → 10.77.0.2/24). The server address itself is
// never suggested.
func suggestClientTunAddr(serverAddr string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(strings.TrimSpace(serverAddr))
	if err != nil {
		return "", fmt.Errorf("parse TUN address %q: %w", serverAddr, err)
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return "", fmt.Errorf("TUN address %q is not IPv4", serverAddr)
	}
	ones, _ := ipnet.Mask.Size()
	// COPY the 4 bytes: ip.To4() aliases ip's backing array, so incrementing
	// in place would mutate the server IP itself and make the
	// candidate.Equal(ip) guard below always true (every address rejected).
	next := append([]byte(nil), ip4...)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	candidate := net.IP(next)
	if !ipnet.Contains(candidate) || candidate.Equal(ip) || candidate.Equal(broadcast(ipnet)) {
		return "", fmt.Errorf("no free client address adjacent to %q in %s", serverAddr, ipnet.String())
	}
	return fmt.Sprintf("%s/%d", candidate.String(), ones), nil
}

func broadcast(ipnet *net.IPNet) net.IP {
	ip := make(net.IP, len(ipnet.IP.To4()))
	for i := range ip {
		ip[i] = ipnet.IP.To4()[i] | ^ipnet.Mask[i]
	}
	return ip
}

// bundleFiles are the exact members packed into the tarball. The CA's key
// (ca.key) and the server's key (server.key) are deliberately absent.
var bundleFiles = []string{"ca.crt", "client.crt", "client.key"}

// packBundle writes the .tgz (0600 — it carries the client's private key)
// and returns the hex SHA256 of the file bytes.
func packBundle(tgzPath, certDir string, skel ClientBundleSkeleton) (string, error) {
	f, err := os.OpenFile(tgzPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("bundle: create %s: %w", tgzPath, err)
	}
	defer f.Close()

	h := sha256.New()
	gz := gzip.NewWriter(io.MultiWriter(f, h))
	tw := tar.NewWriter(gz)

	if err := tarWriteFile(tw, certDir, "skeleton.yaml", []byte(renderSkeletonYAML(skel))); err != nil {
		return "", err
	}
	for _, name := range bundleFiles {
		data, err := os.ReadFile(filepath.Join(certDir, name))
		if err != nil {
			return "", fmt.Errorf("bundle: read %s: %w", name, err)
		}
		if err := tarWriteFile(tw, certDir, name, data); err != nil {
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", fmt.Errorf("bundle: tar close: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("bundle: gzip close: %w", err)
	}
	if err := f.Sync(); err != nil {
		return "", fmt.Errorf("bundle: sync %s: %w", tgzPath, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func tarWriteFile(tw *tar.Writer, dir, name string, data []byte) error {
	hdr := &tar.Header{
		Name: name,
		Mode: 0o600,
		Size: int64(len(data)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("bundle: tar header %s: %w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("bundle: tar write %s: %w", name, err)
	}
	return nil
}

// renderSkeletonYAML renders the skeleton as YAML (loaded by the WI 5.5
// client wizard for prefill).
func renderSkeletonYAML(s ClientBundleSkeleton) string {
	return fmt.Sprintf(`# SSSonector client bundle skeleton (prefill for 'tui --from-bundle')
instance: %s
server_host: %s
server_port: %d
tun_address: %s
`, s.Instance, s.ServerHost, s.ServerPort, s.TunAddress)
}
