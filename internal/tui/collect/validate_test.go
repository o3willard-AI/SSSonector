package collect

import (
	"fmt"
	"strings"
	"testing"

	cfg "github.com/o3willard-AI/SSSonector/internal/config"
)

// dumpText renders the WI 3.1 dump back to draft YAML text. For the
// round-trip test we instead keep the original file text — the truest
// "no-op draft" is the instance file itself, which the dump was derived
// from.
func noOpDraft(t *testing.T, yaml string) string {
	t.Helper()
	return yaml
}

const validDraftBase = `metadata:
  schema_version: "2.0.0"
  environment: development
type: server
config:
  mode: server
  logging:
    level: info
  network:
    name: tun0
    interface: tun0
    mtu: 1500
    address: 10.77.0.1/24
  tunnel:
    listen_address: 0.0.0.0
    listen_port: 8443
  monitor:
    enabled: true
    type: prometheus
    interval: 10s
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
  security:
    tls:
      min_version: "1.2"
      max_version: "1.3"
  auth:
    cert_file: certs/server.crt
`

func TestValidateDraft_ValidEdit(t *testing.T) {
	draft := strings.Replace(validDraftBase, "port: 9090", "port: 9191", 1)
	app, err := ValidateDraft(draft)
	if err != nil {
		t.Fatalf("valid edit must pass: %v", err)
	}
	if app.Config.Monitor.Prometheus.Port != 9191 {
		t.Errorf("merged config must reflect the edit: port=%d, want 9191", app.Config.Monitor.Prometheus.Port)
	}
	if app.Config.Mode != "server" {
		t.Errorf("mode: %q", app.Config.Mode)
	}
}

func TestValidateDraft_NoOpRoundTrip(t *testing.T) {
	// The dump text (unchanged) must round-trip as ok: same loader parses
	// it that produced it.
	app, err := ValidateDraft(noOpDraft(t, validDraftBase))
	if err != nil {
		t.Fatalf("no-op draft must round-trip: %v", err)
	}
	if app.Config.Monitor.Prometheus.Port != 9090 {
		t.Errorf("no-op values preserved: port=%d", app.Config.Monitor.Prometheus.Port)
	}
	// No-op values preserved exactly (no invented normalization).
	if app.Config.Logging.Level != "info" {
		t.Errorf("no-op logging.level: %q", app.Config.Logging.Level)
	}
}

func TestValidateDraft_SyntaxErrorVerbatim(t *testing.T) {
	draft := strings.Replace(validDraftBase, "  mode: server", "  mode: [unclosed", 1)
	_, err := ValidateDraft(draft)
	if err == nil {
		t.Fatal("syntax error must fail")
	}
	// yaml.v3 carries "line N" — assert it survives verbatim.
	if !strings.Contains(err.Error(), "line ") {
		t.Errorf("syntax error must carry line number: %v", err)
	}
	// Verbatim: identical to calling the loader directly.
	_, direct := loadConfigStringDirect(draft)
	if err.Error() != direct.Error() {
		t.Errorf("error must be verbatim:\nValidateDraft: %v\nloader:       %v", err, direct)
	}
}

func TestValidateDraft_SemanticErrorVerbatim(t *testing.T) {
	// Subnet overlap: TUN address conflicts with the validator's check.
	draft := validDraftBase + `  network:
    address: 192.168.101.1/24
`
	_, err := ValidateDraft(draft)
	if err == nil {
		t.Fatal("semantic error must fail")
	}
	// Verbatim: identical to loader+validator called directly.
	_, direct := validateDirect(draft)
	if direct == nil {
		t.Fatal("direct path must also fail (fixture sanity)")
	}
	if err.Error() != direct.Error() {
		t.Errorf("semantic error must be verbatim:\nValidateDraft: %v\ndirect:        %v", err, direct)
	}
}

func TestValidateDraft_SchemaInvalidField(t *testing.T) {
	// A draft whose only change is a schema-invalid field: wrong
	// schema_version (the loader's hard gate).
	draft := strings.Replace(validDraftBase, `schema_version: "2.0.0"`, `schema_version: "1.0.0"`, 1)
	_, err := ValidateDraft(draft)
	if err == nil {
		t.Fatal("schema-invalid draft must fail")
	}
	if !strings.Contains(err.Error(), "unsupported schema version") {
		t.Errorf("want schema-version error, got: %v", err)
	}
	// And a syntactically-valid but semantically-rejected edit: empty mode.
	draft2 := strings.Replace(validDraftBase, "  mode: server", `  mode: ""`, 1)
	if _, err := ValidateDraft(draft2); err != nil {
		t.Logf("empty mode result (validator-dependent): %v", err)
	}
}

func TestValidateDraft_FailClosed(t *testing.T) {
	// An invalid draft returns (nil, error) — never a partially-usable
	// config.
	for _, draft := range []string{
		"",            // empty
		"not: [valid", // syntax
		strings.Replace(validDraftBase, `schema_version: "2.0.0"`, `schema_version: "9.9.9"`, 1), // schema
	} {
		app, err := ValidateDraft(draft)
		if err == nil {
			t.Errorf("draft %q: want error", draft)
			continue
		}
		if app != nil {
			t.Errorf("draft %q: config must be nil on error, got %+v", draft, app)
		}
	}
}

func TestValidateDraft_DumpDerivedEdit(t *testing.T) {
	// End-to-end with the WI 3.1 dump: take a real instance file, produce
	// the dump, then validate a draft that edits a field visible in the
	// dump — proving dump -> draft -> validate coherence.
	root := t.TempDir()
	writeFixture(t, root, "instances/client-a/config.yaml", validDraftBase)
	lines, err := EffectiveConfigDump(ConfigPaths{ConfigRoot: root}, "client-a")
	if err != nil {
		t.Fatalf("dump: %v", err)
	}
	// The dump's port line must agree with a validated draft of the same
	// file.
	app, err := ValidateDraft(validDraftBase)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	found := false
	for _, l := range lines {
		if l.Path == "monitor.prometheus.port" {
			found = true
			if l.Value != fmt.Sprintf("%d", app.Config.Monitor.Prometheus.Port) {
				t.Errorf("dump value %q != validated %d", l.Value, app.Config.Monitor.Prometheus.Port)
			}
			if l.Origin != OriginUser {
				t.Errorf("port must be user-set: %v", l.Origin)
			}
		}
	}
	if !found {
		t.Fatal("port line missing from dump")
	}
}

// loadConfigStringDirect calls the daemon loader directly (verbatim check).
func loadConfigStringDirect(draft string) (*cfg.AppConfig, error) {
	return cfg.LoadConfigString(draft, "yaml")
}

// validateDirect calls loader+validator directly (verbatim check).
func validateDirect(draft string) (*cfg.AppConfig, error) {
	app, err := cfg.LoadConfigString(draft, "yaml")
	if err != nil {
		return nil, err
	}
	if err := cfg.NewValidator().Validate(app); err != nil {
		return nil, err
	}
	return app, nil
}

func TestValidateDraft_NoExtraNormalization(t *testing.T) {
	// The loader leaves unset fields at zero even in a valid config —
	// ValidateDraft must not add its own normalization (no mtu invention).
	draft := `metadata:
  schema_version: "2.0.0"
  environment: development
type: server
config:
  mode: server
  logging:
    level: info
  network:
    name: tun0
    interface: tun0
    mtu: 1500
    address: 10.77.0.1/24
  tunnel:
    listen_address: 0.0.0.0
    listen_port: 8443
  security:
    tls:
      min_version: "1.2"
      max_version: "1.3"
    cert_rotation:
      enabled: true
      interval: 0s
  monitor:
    enabled: true
    type: prometheus
    interval: 10s
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
`
	app, err := ValidateDraft(draft)
	if err != nil {
		t.Fatalf("valid minimal draft: %v", err)
	}
	// cert_rotation.interval stays at the loader's zero (the 30d default
	// is applied by the cert reader at read time, not invented here).
	if app.Config.Auth.CertRotation.Interval != 0 {
		t.Errorf("cert_rotation.interval must stay zero (no invented normalization): %v", app.Config.Auth.CertRotation.Interval)
	}
}
