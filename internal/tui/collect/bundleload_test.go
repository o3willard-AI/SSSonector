package collect

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

// WI 5.5: bundle-load + chain pre-flight tests. Bundles are built by
//GenerateClientBundle (the real generator) or malformed by hand; no real
// network or systemctl.

// TestLoadBundle_PrefillAndStaging: a real generated bundle loads; the
// prefill traces to the skeleton and all three cert files are staged.
func TestLoadBundle_PrefillAndStaging(t *testing.T) {
	root, certDir := bundleFixture(t, "192.168.101.7")
	genCerts(t, certDir)
	b, err := GenerateClientBundle(ConfigPaths{ConfigRoot: root}, "srv-a")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	lb, err := LoadBundle(b.Path, func() string { return t.TempDir() })
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	defer os.RemoveAll(lb.StagingDir)
	if lb.Prefill.Instance != "srv-a" || lb.Prefill.ServerHost != "192.168.101.7" ||
		lb.Prefill.ServerPort != 9443 || lb.Prefill.TunAddress != "10.77.0.2/24" {
		t.Errorf("prefill must trace to the skeleton: %+v", lb.Prefill)
	}
	for _, p := range []string{lb.CAPath, lb.ClientCertPath, lb.ClientKeyPath} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("staged file missing: %s: %v", p, err)
		}
	}
}

// TestLoadBundle_MissingMember_FailsClosed: a bundle missing a required
// member (e.g. client.key removed) must error and stage nothing.
func TestLoadBundle_MissingMember_FailsClosed(t *testing.T) {
	root, certDir := bundleFixture(t, "192.168.101.7")
	genCerts(t, certDir)
	b, err := GenerateClientBundle(ConfigPaths{ConfigRoot: root}, "srv-a")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Rewrite the tgz without client.key.
	src := readTarMembers(t, b.Path)
	delete(src, "client.key")
	rewritten := filepath.Join(t.TempDir(), "crippled.tgz")
	writeTarMembers(t, rewritten, src)

	lb, err := LoadBundle(rewritten, func() string { return t.TempDir() })
	if err == nil {
		t.Fatal("missing client.key must fail closed")
	}
	if !strings.Contains(err.Error(), "client.key") {
		t.Errorf("error must name the missing member: %v", err)
	}
	if lb.StagingDir != "" {
		t.Errorf("failed load must stage nothing, got %q", lb.StagingDir)
	}
}

// TestLoadBundle_MissingSkeletonFile_Fails: not a bundle at all.
func TestLoadBundle_NotAGzip_Fails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "garbage.tgz")
	if err := os.WriteFile(p, []byte("definitely not gzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBundle(p, func() string { return t.TempDir() }); err == nil {
		t.Fatal("non-gzip input must error")
	}
}

// TestLoadBundle_MissingFile_Fails: nonexistent path errors explicitly.
func TestLoadBundle_MissingFile_Fails(t *testing.T) {
	_, err := LoadBundle(filepath.Join(t.TempDir(), "nope.tgz"), func() string { return t.TempDir() })
	if err == nil || !strings.Contains(err.Error(), "nope.tgz") {
		t.Errorf("missing bundle must error naming the path: %v", err)
	}
}

// TestLoadBundle_IncompleteSkeleton_Fails: a skeleton missing fields
// cannot silently drive the wizard with blank prefill.
func TestLoadBundle_IncompleteSkeleton_Fails(t *testing.T) {
	root, certDir := bundleFixture(t, "192.168.101.7")
	genCerts(t, certDir)
	b, err := GenerateClientBundle(ConfigPaths{ConfigRoot: root}, "srv-a")
	if err != nil {
		t.Fatal(err)
	}
	members := readTarMembers(t, b.Path)
	// Drop the tun_address line from the skeleton.
	skel := strings.Replace(string(members["skeleton.yaml"]), "tun_address: 10.77.0.2/24\n", "", 1)
	members["skeleton.yaml"] = []byte(skel)
	rewritten := filepath.Join(t.TempDir(), "incomplete.tgz")
	writeTarMembers(t, rewritten, members)

	if _, err := LoadBundle(rewritten, func() string { return t.TempDir() }); err == nil {
		t.Fatal("incomplete skeleton must fail closed")
	}
}

// TestVerifyClientChainAgainstCA_ChainOK: the bundle's own CA verifies
// its client cert (the happy pre-flight).
func TestVerifyClientChainAgainstCA_ChainOK(t *testing.T) {
	root, certDir := bundleFixture(t, "192.168.101.7")
	genCerts(t, certDir)
	b, err := GenerateClientBundle(ConfigPaths{ConfigRoot: root}, "srv-a")
	if err != nil {
		t.Fatal(err)
	}
	lb, err := LoadBundle(b.Path, func() string { return t.TempDir() })
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(lb.StagingDir)
	if err := VerifyClientChainAgainstCA(lb.CAPath, lb.ClientCertPath); err != nil {
		t.Errorf("bundle's own chain must verify: %v", err)
	}
}

// TestVerifyClientChainAgainstCA_MismatchBlocks: a client cert signed by
// a DIFFERENT CA must be blocked by the pre-flight (the §3.3 blocking
// behavior).
func TestVerifyClientChainAgainstCA_MismatchBlocks(t *testing.T) {
	root, certDir := bundleFixture(t, "192.168.101.7")
	genCerts(t, certDir)
	b, err := GenerateClientBundle(ConfigPaths{ConfigRoot: root}, "srv-a")
	if err != nil {
		t.Fatal(err)
	}
	lb, err := LoadBundle(b.Path, func() string { return t.TempDir() })
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(lb.StagingDir)

	// Mint an unrelated CA + client pair.
	otherRoot, otherCertDir := bundleFixture(t, "10.9.9.9")
	genCerts(t, otherCertDir)
	_ = otherRoot
	// genCerts removes the auto-issued client pair for the bundle tests;
	// for the mismatch test we need the other CA's client leaf, so mint it.
	if err := generator.IssueClientCert(otherCertDir); err != nil {
		t.Fatalf("mint other client cert: %v", err)
	}

	// The pre-flight against the OTHER CA (cert replaced, CA original)
	// must fail: swap in the other client.crt while keeping the original
	// CA — a signature mismatch.
	otherClient, err := os.ReadFile(filepath.Join(otherCertDir, "client.crt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lb.ClientCertPath, otherClient, 0o600); err != nil {
		t.Fatal(err)
	}
	err = VerifyClientChainAgainstCA(lb.CAPath, lb.ClientCertPath)
	if err == nil {
		t.Fatal("CA mismatch MUST be blocked by the pre-flight")
	}
	if !strings.Contains(err.Error(), "does NOT verify") {
		t.Errorf("mismatch error must be explicit: %v", err)
	}
}

// writeTarMembers packs a map into a .tgz (test helper for malformed
// bundles).
func writeTarMembers(t *testing.T, path string, members map[string][]byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, data := range members {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}
