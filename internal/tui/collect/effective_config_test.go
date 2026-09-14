package collect

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	cfg "github.com/o3willard-AI/SSSonector/internal/config"
)

func dumpFixture(t *testing.T, yaml string) []ConfigLine {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "instances/client-a/config.yaml", yaml)
	lines, err := EffectiveConfigDump(ConfigPaths{ConfigRoot: root}, "client-a")
	if err != nil {
		t.Fatalf("EffectiveConfigDump: %v", err)
	}
	return lines
}

func findLine(t *testing.T, lines []ConfigLine, path string) ConfigLine {
	t.Helper()
	for _, l := range lines {
		if l.Path == path {
			return l
		}
	}
	t.Fatalf("path %q missing from dump (%d lines)", path, len(lines))
	return ConfigLine{}
}

const mergeFixtureYAML = `metadata:
  schema_version: "2.0.0"
type: server
config:
  mode: server
  logging:
    level: debug
  monitor:
    enabled: true
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
  auth:
    cert_file: certs/server.crt
`

func TestEffectiveConfig_OriginMarking(t *testing.T) {
	lines := dumpFixture(t, mergeFixtureYAML)

	// Explicit fields are user-set.
	for _, path := range []string{"mode", "logging.level", "monitor.enabled",
		"monitor.prometheus.enabled", "monitor.prometheus.port", "monitor.prometheus.path", "auth.cert_file"} {
		if l := findLine(t, lines, path); l.Origin != OriginUser {
			t.Errorf("%s: origin %v, want user (explicit in file)", path, l.Origin)
		}
	}
	// Non-explicit fields are default.
	for _, path := range []string{"logging.format", "network.mtu", "tunnel.listen_port",
		"auth.key_file", "auth.ca_file", "monitor.interval"} {
		if l := findLine(t, lines, path); l.Origin != OriginDefault {
			t.Errorf("%s: origin %v, want default (absent from file)", path, l.Origin)
		}
	}
	// Values come from the merged loader output.
	if l := findLine(t, lines, "monitor.prometheus.port"); l.Value != "9090" {
		t.Errorf("port value: %q", l.Value)
	}
	// The loader leaves unset fields at their zero value (schema defaults
	// like reconnect delays are applied at validation time via Normalized(),
	// not by the loader) — the dump reports the loader's truth faithfully.
	if l := findLine(t, lines, "network.mtu"); l.Value != "0" {
		t.Errorf("default mtu value: %q (loader default is 0)", l.Value)
	}
}

func TestEffectiveConfig_ExplicitDefaultIsUserSet(t *testing.T) {
	// logging.level explicitly set to the DEFAULT value ("info") must
	// still be user-set; an absent sibling stays default.
	yaml := `metadata:
  schema_version: "2.0.0"
type: server
config:
  mode: server
  logging:
    level: info
`
	lines := dumpFixture(t, yaml)
	if l := findLine(t, lines, "logging.level"); l.Origin != OriginUser {
		t.Errorf("explicitly-set default must be user-set: %v", l.Origin)
	}
	if l := findLine(t, lines, "logging.level"); l.Value != "info" {
		t.Errorf("explicit-default value: %q", l.Value)
	}
	if l := findLine(t, lines, "logging.format"); l.Origin != OriginDefault {
		t.Errorf("absent sibling must stay default: %v", l.Origin)
	}
}

func TestEffectiveConfig_DriftCheckAgainstDefaultConfig(t *testing.T) {
	// Every leaf path in the dump of a MINIMAL file must exist in
	// internal/config DefaultConfig() (no field silently invented), and
	// DefaultConfig's own leaf paths must all be covered by the dump
	// renderer (no field silently dropped from the view).
	minimal := `metadata:
  schema_version: "2.0.0"
type: server
config:
  mode: server
`
	lines := dumpFixture(t, minimal)
	dumpPaths := map[string]bool{}
	for _, l := range lines {
		dumpPaths[l.Path] = true
	}

	// DefaultConfig leaves (same reflection walk over the defaults struct).
	defLines := flattenAppConfig("", reflect.ValueOf(cfg.DefaultConfig().Config).Elem())
	defPaths := map[string]bool{}
	for _, l := range defLines {
		defPaths[l.Path] = true
	}

	// Drift check 1: dump must not invent paths the defaults don't have.
	for p := range dumpPaths {
		if !defPaths[p] {
			t.Errorf("dump path %q absent from DefaultConfig — invented field?", p)
		}
	}
	// Drift check 2: every default leaf must be covered by the dump.
	missing := 0
	for p := range defPaths {
		if !dumpPaths[p] {
			missing++
			t.Errorf("DefaultConfig leaf %q missing from dump — silently omitted", p)
		}
	}
	if missing > 0 {
		t.Errorf("%d default leaves missing", missing)
	}

	// All minimal-dump lines (except the wrapper metadata paths) are
	// default-filled by construction.
	for _, l := range lines {
		if l.Origin != OriginUser && l.Origin != OriginDefault {
			t.Errorf("path %q: bad origin %v", l.Path, l.Origin)
		}
	}
}

func TestEffectiveConfig_Deterministic(t *testing.T) {
	a := dumpFixture(t, mergeFixtureYAML)
	b := dumpFixture(t, mergeFixtureYAML)
	if len(a) != len(b) {
		t.Fatalf("length drift: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("line %d drifted: %+v vs %+v", i, a[i], b[i])
		}
	}
	// Sorted by path.
	for i := 1; i < len(a); i++ {
		if a[i-1].Path >= a[i].Path {
			t.Errorf("not sorted at %d: %q >= %q", i, a[i-1].Path, a[i].Path)
		}
	}
}

func TestEffectiveConfig_FormatStable(t *testing.T) {
	lines := dumpFixture(t, mergeFixtureYAML)
	out := FormatConfigDump(lines)
	if !strings.HasPrefix(out, "effective config:\n") {
		t.Errorf("format header missing: %q", out[:40])
	}
	// Every line tagged with an origin.
	for _, l := range lines {
		needle := "[" + l.Origin.String() + "]"
		if !strings.Contains(out, needle) {
			t.Errorf("origin tag %s missing for %s", needle, l.Path)
		}
	}
}

func TestEffectiveConfig_MissingFileErrors(t *testing.T) {
	_, err := EffectiveConfigDump(ConfigPaths{ConfigRoot: t.TempDir()}, "ghost")
	if err == nil || !strings.Contains(err.Error(), "no config file found") {
		t.Errorf("want explicit error, got %v", err)
	}
}

func TestEffectiveConfig_InvalidSchemaSurfacesLoaderError(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "instances/client-a/config.yaml", "config:\n  mode: [unclosed\n")
	_, dumpErr := EffectiveConfigDump(ConfigPaths{ConfigRoot: root}, "client-a")
	if dumpErr == nil {
		t.Fatal("want loader error")
	}
	// Verbatim vs direct loader call.
	_, loaderErr := cfg.LoadConfigFile(filepath.Join(root, "instances", "client-a", "config.yaml"))
	if loaderErr == nil || dumpErr.Error() != loaderErr.Error() {
		t.Errorf("loader error must be surfaced verbatim:\n%v\nvs\n%v", dumpErr, loaderErr)
	}
}

func TestExplicitYAMLKeys_NestedAndSequences(t *testing.T) {
	raw := []byte("a:\n  b: 1\n  c:\n    d: true\nlist:\n  - x\n  - y\n")
	keys, err := explicitYAMLKeys(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, want := range []string{"a.b", "a.c.d", "list"} {
		if !keys[want] {
			t.Errorf("key %q missing: %v", want, keys)
		}
	}
	if keys["a"] || keys["a.c"] {
		t.Errorf("non-leaf mapping keys must not be marked: %v", keys)
	}
}

func TestEffectiveConfig_UnreadableFile(t *testing.T) {
	// Unreadable YAML file: explicit-key parse error surfaces (not swallowed).
	root := t.TempDir()
	p := filepath.Join(root, "instances", "client-a", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(":\n:\n  - ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := EffectiveConfigDump(ConfigPaths{ConfigRoot: root}, "client-a"); err == nil {
		t.Fatal("want error for malformed yaml")
	}
}
