package collect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfg "github.com/o3willard-AI/SSSonector/internal/config"
)

// writeFixture writes a YAML file under the temp config root.
func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

const validInstanceYAML = `metadata:
  schema_version: "2.0.0"
type: server
config:
  mode: server
  monitor:
    enabled: true
    prometheus:
      enabled: true
      port: 9443
      path: /metrics
  auth:
    cert_file: /etc/sssonector/instances/client-a/certs/server.crt
    key_file: /etc/sssonector/instances/client-a/certs/server.key
    ca_file: /etc/sssonector/instances/client-a/certs/ca.crt
`

func TestResolveInstanceConfig_Valid(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "instances/client-a/config.yaml", validInstanceYAML)

	ic, err := ResolveInstanceConfig(ConfigPaths{ConfigRoot: root}, "client-a")
	if err != nil {
		t.Fatalf("ResolveInstanceConfig: %v", err)
	}
	if ic.Name != "client-a" {
		t.Errorf("name: %q", ic.Name)
	}
	if ic.File != filepath.Join(root, "instances/client-a/config.yaml") {
		t.Errorf("file: %q", ic.File)
	}
	if ic.Mode != "server" {
		t.Errorf("mode: %q", ic.Mode)
	}
	if !ic.Prometheus.Enabled || ic.Prometheus.Port != 9443 || ic.Prometheus.Path != "/metrics" {
		t.Errorf("prometheus: %+v", ic.Prometheus)
	}
	want := CertPaths{
		CertFile: "/etc/sssonector/instances/client-a/certs/server.crt",
		KeyFile:  "/etc/sssonector/instances/client-a/certs/server.key",
		CAFile:   "/etc/sssonector/instances/client-a/certs/ca.crt",
	}
	if ic.CertPaths != want {
		t.Errorf("cert paths: %+v, want %+v", ic.CertPaths, want)
	}
}

func TestResolveInstanceConfig_InstanceWinsOverLegacy(t *testing.T) {
	root := t.TempDir()
	// Both files exist; instance file must win.
	writeFixture(t, root, "config.yaml", strings.Replace(validInstanceYAML, "port: 9443", "port: 1111", 1))
	writeFixture(t, root, "instances/client-a/config.yaml", validInstanceYAML)

	ic, err := ResolveInstanceConfig(ConfigPaths{ConfigRoot: root}, "client-a")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if ic.Prometheus.Port != 9443 || !strings.Contains(ic.File, "instances"+string(filepath.Separator)+"client-a") {
		t.Errorf("instance file must win: file=%q port=%d", ic.File, ic.Prometheus.Port)
	}
}

func TestResolveInstanceConfig_LegacyFallback(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "config.yaml", validInstanceYAML)

	ic, err := ResolveInstanceConfig(ConfigPaths{ConfigRoot: root}, "default")
	if err != nil {
		t.Fatalf("legacy fallback: %v", err)
	}
	if ic.File != filepath.Join(root, "config.yaml") {
		t.Errorf("file: %q", ic.File)
	}
	if ic.Mode != "server" {
		t.Errorf("mode: %q", ic.Mode)
	}
}

func TestResolveInstanceConfig_MissingFile(t *testing.T) {
	root := t.TempDir()
	_, err := ResolveInstanceConfig(ConfigPaths{ConfigRoot: root}, "ghost")
	if err == nil {
		t.Fatal("want error for missing config, got nil")
	}
	if !strings.Contains(err.Error(), "no config file found") {
		t.Errorf("want explicit resolution error, got: %v", err)
	}
}

func TestResolveInstanceConfig_InvalidSchema_LoaderErrorVerbatim(t *testing.T) {
	root := t.TempDir()
	bad := "config:\n  mode: [unclosed\n"
	writeFixture(t, root, "instances/client-a/config.yaml", bad)

	_, resolveErr := ResolveInstanceConfig(ConfigPaths{ConfigRoot: root}, "client-a")
	if resolveErr == nil {
		t.Fatal("want loader error, got nil")
	}

	// Prove verbatim: the direct loader error must match what the resolver
	// returned (same message, no wrapping text added).
	_, loaderErr := cfg.LoadConfigFile(filepath.Join(root, "instances", "client-a", "config.yaml"))
	if loaderErr == nil {
		t.Fatal("loader itself must error on this fixture")
	}
	if resolveErr.Error() != loaderErr.Error() {
		t.Errorf("error must be verbatim:\nresolver: %v\nloader:   %v", resolveErr, loaderErr)
	}
}

func TestResolveInstanceConfig_RealConfigFixtures(t *testing.T) {
	// Round-trip the repo's real config fixtures through the resolver by
	// pointing ConfigRoot at a temp dir that contains copies.
	root := t.TempDir()
	for _, src := range []string{"configs/server.yaml", "configs/client.yaml"} {
		data, err := os.ReadFile(filepath.Join("..","..","..",src))
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		name := "default"
		if strings.Contains(src, "client") {
			name = "client-x"
		}
		writeFixture(t, root, "instances/"+name+"/config.yaml", string(data))
	}

	t.Run("server.yaml", func(t *testing.T) {
		ic, err := ResolveInstanceConfig(ConfigPaths{ConfigRoot: root}, "default")
		if err != nil {
			t.Fatalf("server.yaml: %v", err)
		}
		if ic.Mode != "server" {
			t.Errorf("server mode: %q", ic.Mode)
		}
		if !ic.Prometheus.Enabled || ic.Prometheus.Port != 9090 || ic.Prometheus.Path != "/metrics" {
			t.Errorf("server prometheus: %+v", ic.Prometheus)
		}
		// configs/server.yaml keeps cert paths empty (template) — must stay
		// empty, never fabricated.
		if ic.CertPaths.CertFile != "" || ic.CertPaths.KeyFile != "" || ic.CertPaths.CAFile != "" {
			t.Errorf("server cert paths must be empty as in the file: %+v", ic.CertPaths)
		}
	})

	t.Run("client", func(t *testing.T) {
		ic, err := ResolveInstanceConfig(ConfigPaths{ConfigRoot: root}, "client-x")
		if err != nil {
			t.Fatalf("client: %v", err)
		}
		if ic.Mode != "client" {
			t.Errorf("client mode: %q", ic.Mode)
		}
		if ic.CertPaths.CertFile != "certs/client.crt" || ic.CertPaths.KeyFile != "certs/client.key" || ic.CertPaths.CAFile != "certs/ca.crt" {
			t.Errorf("client cert paths: %+v", ic.CertPaths)
		}
	})
}

// loadDirect calls the daemon loader directly for the verbatim-error check.
func loadDirect(path string) (*cfg.AppConfig, error) {
	return cfg.LoadConfigFile(path)
}
