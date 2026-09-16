package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// fakeSignal records SIGHUP calls (never a real signal).
type fakeSignal struct {
	calls []int
}

func (f *fakeSignal) signal(pid int) error {
	f.calls = append(f.calls, pid)
	return nil
}

// configFixture builds a model with real config files on disk, real dump +
// real ValidateDraft (the daemon loader/validator), and FAKE signal/read —
// no real signal is ever sent.
func configFixture(t *testing.T, draftYAML string) (dashboardModel, *fakeSignal, string) {
	t.Helper()
	root := t.TempDir()
	writeCfgFixture(t, root, "instances/client-a/config.yaml", draftYAML)

	r := &cfgRunner{outputs: map[string]string{}}
	r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
		"sssonector@client-a.service loaded active running x\n"
	r.outputs["systemctl show sssonector@client-a.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
		"ActiveState=active\nSubState=running\nMainPID=8123\n"
	r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
		"ActiveState=not-found\nSubState=dead\nMainPID=0\n"

	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"client-a"})
	}, func() time.Time { return screenNow })
	m.deps = ConfigDeps{
		Dump:     collect.EffectiveConfigDump,
		Validate: validateDraftWrapper,
		Paths:    collect.ConfigPaths{ConfigRoot: root},
		Apply:    collect.ApplyConfig,
	}
	// Real dump (collect.EffectiveConfigDump), real ValidateDraft, fake
	// signal captured for assertions.
	sig := &fakeSignal{}
	m.deps.Signal = sig.signal
	m.Init()
	m = driveTick(m)
	return m, sig, root
}

func writeCfgFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := osMkdirAll(filepath.Dir(p)); err != nil {
		t.Fatal(err)
	}
	if err := osWriteFile(p, content); err != nil {
		t.Fatal(err)
	}
}

// cfgRunner is a local fake CommandRunner (collect's is unexported).
type cfgRunner struct {
	outputs map[string]string
}

func (f *cfgRunner) Run(args ...string) (string, error) {
	if out, ok := f.outputs[strings.Join(args, " ")]; ok {
		return out, nil
	}
	if len(args) > 2 && args[1] == "show" {
		return "ActiveState=not-found\nSubState=dead\nMainPID=0\n", nil
	}
	return "", nil
}

// osMkdirAll / osWriteFile / osReadFile are thin wrappers for readability.
func osMkdirAll(dir string) error { return os.MkdirAll(dir, 0o750) }
func osWriteFile(p, content string) error {
	return os.WriteFile(p, []byte(content), 0o600)
}
func osReadFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	return string(b), err
}

// ckey sends a key while in config mode.
func ckey(m dashboardModel, s string) dashboardModel {
	next, _ := m.Update(tea.KeyMsg{Type: keyTypeFor(s), Runes: runesFor(s)})
	return next.(dashboardModel)
}

const cfgValidYAML = `metadata:
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
  monitor:
    enabled: true
    type: prometheus
    interval: 10s
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
`

func TestConfigOpen_C(t *testing.T) {
	m, _, _ := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	if m.mode != modeConfig {
		t.Fatalf("c must open config mode, got %v", m.mode)
	}
	if m.config.instance != "client-a" {
		t.Errorf("config instance: %q", m.config.instance)
	}
	// The draft is YAML reconstructed from the dump's lines (display dump
	// remains available via FormatConfigDump). It must contain each dump
	// path as a YAML key.
	for _, l := range m.config.lines {
		last := l.Path
		if i := strings.LastIndex(l.Path, "."); i >= 0 {
			last = l.Path[i+1:]
		}
		if !strings.Contains(m.config.draft, last+":") {
			t.Errorf("draft missing key %q (from path %q)", last, l.Path)
		}
	}
	if !strings.Contains(m.View(), "CONFIG — instance client-a") {
		t.Errorf("config overlay must render:\n%s", m.View())
	}
}

func TestConfigEdit_EnterAndCommit(t *testing.T) {
	m, _, _ := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	// Select the logging.level line and edit it.
	idx := -1
	for i, l := range m.config.lines {
		if l.Path == "logging.level" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("logging.level line missing")
	}
	for m.config.sel != idx {
		m = ckey(m, "down")
	}
	m = ckey(m, "enter") // begin editing
	if !m.config.editing {
		t.Fatal("enter must begin editing")
	}
	m.config.editLineText("debug")
	m = ckey(m, "enter") // commit
	if m.config.editing {
		t.Error("enter must commit the edit")
	}
	if m.config.lines[idx].Value != "debug" {
		t.Errorf("edited value: %q", m.config.lines[idx].Value)
	}
	if m.config.lines[idx].Origin != collect.OriginUser {
		t.Errorf("an edit is by definition user-set: %v", m.config.lines[idx].Origin)
	}
	if !strings.Contains(m.config.draft, "debug") {
		t.Error("draft must reflect the edit")
	}
}

func TestConfigValidateFail_VerbatimBanner(t *testing.T) {
	m, _, _ := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	// Edit to an invalid value: break the TLS min version.
	for i, l := range m.config.lines {
		if l.Path == "security.tls.min_version" {
			m.config.sel = i
		}
	}
	m = ckey(m, "enter")
	m.config.editLineText("2.0")
	m = ckey(m, "enter")
	m = ckey(m, "v")
	if m.config.banner != bannerValidateFail {
		t.Fatalf("banner: %v, want validate-fail", m.config.banner)
	}
	if !strings.Contains(m.config.bannerUp, "TLS min version cannot be greater than max version") {
		t.Errorf("banner must carry the verbatim validator error: %q", m.config.bannerUp)
	}
	if !strings.Contains(m.View(), "✗ VALIDATION FAILED") {
		t.Errorf("banner must render:\n%s", m.View())
	}
}

func TestConfigValidateOK(t *testing.T) {
	m, _, _ := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	// No edit: the no-op draft round-trips (WI 3.2 guarantee).
	m = ckey(m, "v")
	if m.config.banner != bannerValidateOK {
		t.Errorf("banner: %v, want validate-ok (err=%q)", m.config.banner, m.config.bannerUp)
	}
	if !strings.Contains(m.View(), "✓ draft valid") {
		t.Errorf("ok banner must render:\n%s", m.View())
	}
}

func TestConfigApply_ValidDraft(t *testing.T) {
	m, sig, root := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	m = ckey(m, "a")
	if m.config.banner != bannerApplyOK {
		t.Fatalf("banner: %v (up=%q)", m.config.banner, m.config.bannerUp)
	}
	if len(sig.calls) != 1 || sig.calls[0] != 8100 {
		t.Errorf("signal calls: %v, want [8123]", sig.calls)
	}
	// The target file was written with the draft (validated form).
	written, err := osReadFile(filepath.Join(root, "instances", "client-a", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(written, "mode: \"server\"") || !strings.Contains(written, "config:") {
		t.Errorf("written file must be the validated YAML draft:\n%s", written)
	}
	if !strings.Contains(m.View(), "✓ applied") {
		t.Errorf("apply-ok banner must render:\n%s", m.View())
	}
}

func TestConfigApply_InvalidDraft_NeverApplies(t *testing.T) {
	m, sig, root := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	// Edit TLS min to something invalid, then press a.
	for i, l := range m.config.lines {
		if l.Path == "security.tls.min_version" {
			m.config.sel = i
		}
	}
	m = ckey(m, "enter")
	m.config.editLineText("2.0")
	m = ckey(m, "enter")
	m = ckey(m, "a") // validate fails inside apply → NEVER applies
	if m.config.banner != bannerValidateFail {
		t.Errorf("a on invalid draft must show validate-fail, got %v", m.config.banner)
	}
	if len(sig.calls) != 0 {
		t.Errorf("FAIL-CLOSED VIOLATED: invalid draft signaled: %v", sig.calls)
	}
	// The file on disk is untouched.
	written, err := osReadFile(filepath.Join(root, "instances", "client-a", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(written, "min_version: \"2.0\"") {
		t.Error("invalid draft must never reach disk")
	}
}

func TestConfigApply_RejectedOutcome(t *testing.T) {
	m, sig, _ := configFixture(t, cfgValidYAML)
	// Fake reload reader reports rejection.
	m.deps.Read = func(int) (collect.ReloadOutcome, error) { return collect.ReloadRejected, nil }
	m = ckey(m, "c")
	m = ckey(m, "a")
	if m.config.banner != bannerApplyRejected {
		t.Fatalf("banner: %v, want apply-rejected", m.config.banner)
	}
	if !strings.Contains(m.View(), "daemon keeps its old config") {
		t.Errorf("rejected banner must document semantics:\n%s", m.View())
	}
	if len(sig.calls) != 1 {
		t.Errorf("signal fired once: %v", sig.calls)
	}
}

func TestConfigEsc_DiscardNoWriteNoSignal(t *testing.T) {
	m, sig, root := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	// Make an edit, then discard.
	for i, l := range m.config.lines {
		if l.Path == "logging.level" {
			m.config.sel = i
		}
	}
	m = ckey(m, "enter")
	m.config.editLineText("debug")
	m = ckey(m, "enter")
	m = ckey(m, "esc")
	if m.mode != modeDashboard {
		t.Fatalf("esc must return to dashboard, got %v", m.mode)
	}
	if len(sig.calls) != 0 {
		t.Errorf("discard must never signal: %v", sig.calls)
	}
	written, err := osReadFile(filepath.Join(root, "instances", "client-a", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(written, "level: debug") {
		t.Error("discard must never write")
	}
	// Dashboard renders again.
	if !strings.Contains(m.View(), "TUNNEL") {
		t.Errorf("dashboard must render after esc:\n%s", m.View())
	}
}

// TestConfigGolden commits the full config-overlay render.
func TestConfigGolden(t *testing.T) {
	m, _, _ := configFixture(t, cfgValidYAML)
	m = ckey(m, "c")
	_ = ckey(m, "v")
	out := m.View()
	path := filepath.Join("testdata", "config_overlay.golden")
	if *updateScreenGoldens {
		if err := osWriteFile(path, out); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := osReadFile(path)
	if err != nil {
		t.Fatalf("golden missing (run with -update-screens): %v", err)
	}
	// The dump includes created/modified timestamps from metadata —
	// normalize those two lines for the golden compare.
	got := normalizeTimes(out)
	wantN := normalizeTimes(string(want))
	if got != wantN {
		t.Errorf("config golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, wantN)
	}
}

func normalizeTimes(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if strings.Contains(l, "metadata.created") || strings.Contains(l, "metadata.modified") ||
			strings.Contains(l, "metadata.updated_at") {
			lines[i] = l[:strings.Index(l, " = ")+3] + "TIME [user]"
		}
	}
	return strings.Join(lines, "\n")
}

var errNoop = errors.New("noop")
var _ = fmt.Sprintf
var _ = errNoop

// validateDraftWrapper adapts collect.ValidateDraft to the deps signature.
func validateDraftWrapper(d string) error {
	_, err := collect.ValidateDraft(d)
	return err
}
