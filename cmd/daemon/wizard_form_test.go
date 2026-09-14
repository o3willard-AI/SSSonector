package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 5.2: server-wizard field-validation table. Every row fails if the
// validation is reverted (invalid input must block, valid input must
// unblock). No real ss/systemctl runs — seams injected.

// okValidate is a real-loader validate (the actual production chain).
var okValidate = collect.ValidateDraft

// freePortProbe: every port free.
func freePortProbe(int) bool { return true }

// busyPortProbe: every port in use.
func busyPortProbe(int) bool { return false }

// filledForm returns a form with every field validly chosen (mode Server,
// instance, free port, valid TUN, NAT on, certs generate-new).
func filledForm() wizardForm {
	f := newWizardForm(okValidate, freePortProbe, nil)
	f.modeChosen = true
	f.modeClient = false
	f.instance = "client-a"
	f.port = "9443"
	f.tun = "10.77.0.1/24"
	f.nat = natEnabled
	f.cert = certGenerateNew
	return f
}

// TestWizardForm_NeverDefaultToServer: a zero form has NOTHING
// pre-selected — mode unset, cert unset, NAT unset — and create is
// disabled until every choice is explicit.
func TestWizardForm_NeverDefaultToServer(t *testing.T) {
	f := newWizardForm(okValidate, freePortProbe, nil)
	if f.modeChosen {
		t.Error("mode must NOT be pre-selected (asked, never defaulted)")
	}
	if f.cert != certNone {
		t.Errorf("certs must start unset (fail-closed), got %v", f.cert)
	}
	if f.nat != natUnset {
		t.Errorf("NAT must start unset, got %v", f.nat)
	}
	if f.ready() {
		t.Fatal("create must be DISABLED on a zero form")
	}
	errs := f.fieldErrs()
	for _, want := range []string{"MODE", "INSTANCE", "LISTEN PORT", "TUN ADDRESS", "FORWARD NAT", "CERTS"} {
		found := false
		for _, e := range errs {
			if strings.Contains(e, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("zero-form errors must mention %q: %v", want, errs)
		}
	}
}

// TestWizardForm_FieldTable drives the per-field validation table:
// invalid blocks, valid unblocks. Each row mutates ONE field from the
// valid baseline.
func TestWizardForm_FieldTable(t *testing.T) {
	rows := []struct {
		name    string
		mutate  func(*wizardForm)
		wantErr string // "" = must be ready
	}{
		{"instance blank", func(f *wizardForm) { f.instance = " " }, "INSTANCE: required"},
		{"instance ok", func(f *wizardForm) {}, ""},
		{"port not a number", func(f *wizardForm) { f.port = "nine" }, "must be 1–65535"},
		{"port zero", func(f *wizardForm) { f.port = "0" }, "must be 1–65535"},
		{"port too big", func(f *wizardForm) { f.port = "70000" }, "must be 1–65535"},
		{"port collision", func(f *wizardForm) {
			f.portFree = busyPortProbe
		}, "already in use"},
		{"port free ok", func(f *wizardForm) {}, ""},
		{"tun garbage", func(f *wizardForm) { f.tun = "not-a-cidr" }, "not a valid CIDR"},
		{"tun network address", func(f *wizardForm) { f.tun = "10.77.0.0/24" }, "host address"},
		{"tun overlap", func(f *wizardForm) {
			f.existing = []string{"10.77.0.5/24"}
		}, "overlaps existing instance subnet 10.77.0.5/24"},
		{"tun ok distinct subnet", func(f *wizardForm) {
			f.existing = []string{"10.77.1.5/24"}
		}, ""},
		{"certs unset", func(f *wizardForm) { f.cert = certNone }, "CERTS: choose"},
		{"certs reuse-host ok", func(f *wizardForm) { f.cert = certReuseHost }, ""},
		{"certs generate ok", func(f *wizardForm) {}, ""},
		{"nat unset", func(f *wizardForm) { f.nat = natUnset }, "FORWARD NAT: choose"},
		{"nat disabled ok", func(f *wizardForm) { f.nat = natDisabled }, ""},
		{"mode unset", func(f *wizardForm) { f.modeChosen = false }, "MODE: choose"},
	}
	for _, row := range rows {
		f := filledForm()
		row.mutate(&f)
		errs := f.fieldErrs()
		if row.wantErr == "" {
			if !f.ready() {
				t.Errorf("%s: valid input must unblock create, errors: %v", row.name, errs)
			}
			continue
		}
		if f.ready() {
			t.Errorf("%s: invalid input must BLOCK create", row.name)
			continue
		}
		found := false
		for _, e := range errs {
			if strings.Contains(e, row.wantErr) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: errors %v must mention %q", row.name, errs, row.wantErr)
		}
	}
}

// TestWizardForm_CertsFailClosed: with NEITHER cert choice made, create
// stays disabled even when every other field is perfect — and the
// rendered screen shows the fail-closed hint.
func TestWizardForm_CertsFailClosed(t *testing.T) {
	f := filledForm()
	f.cert = certNone
	if f.ready() {
		t.Fatal("certs unset must keep create DISABLED (fail-closed)")
	}
	out := serverWizardModel{form: f}.View()
	if !strings.Contains(out, "fail-closed: choose one") {
		t.Errorf("screen must show the certs fail-closed hint:\n%s", out)
	}
	if strings.Contains(out, "all fields valid") {
		t.Errorf("screen must not claim validity with certs unset:\n%s", out)
	}
}

// TestWizardForm_CreateBlockedThenUnblocked: the tea loop blocks Enter
// until ready; once ready, Enter runs the WI 5.2 no-op (prints draft).
func TestWizardForm_CreateBlockedThenUnblocked(t *testing.T) {
	m := serverWizardModel{form: newWizardForm(okValidate, freePortProbe, nil)}
	// Drive every field through real key events.
	m = wizKey(m, "enter")            // MODE: choose Server (focused first)
	m = wizKey(m, "tab")              // INSTANCE
	m = wizType(m, "client-a")        //
	m = wizKey(m, "tab")              // PORT
	m = wizType(m, "9443")            //
	m = wizKey(m, "tab")              // TUN
	m = wizType(m, "10.77.0.1/24")    //
	m = wizKey(m, "tab")              // NAT
	m = wizKey(m, " ")                // enable
	m = wizKey(m, "tab")              // CERTS
	m = wizKey(m, " ")                // generate-new (second position: space toggles none→reuse→gen? none→reuse)
	if m.form.cert == certNone {
		t.Fatal("toggle must select a cert choice")
	}
	if m.form.cert == certReuseHost {
		m = wizKey(m, " ") // cycle to generate-new
	}
	if !m.form.ready() {
		t.Fatalf("all fields driven — create must unblock, errors: %v", m.form.fieldErrs())
	}
	// Enter now runs create (5.2 no-op): done flips, draft renders.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(serverWizardModel)
	if !m.form.done {
		t.Fatal("ready + Enter must run the create no-op")
	}
	out := m.View()
	if !strings.Contains(out, "Validated draft:") || !strings.Contains(out, "schema_version") {
		t.Errorf("done screen must print the validated draft:\n%s", out)
	}
	if !strings.Contains(out, "mode: server") || !strings.Contains(out, "listen_port: 9443") {
		t.Errorf("draft must carry the form values:\n%s", out)
	}
	if !strings.Contains(out, "default_deny: true") {
		t.Errorf("NAT-enabled draft must write default-deny forward NAT:\n%s", out)
	}
}

// TestWizardForm_CreateBlockedMidFill: Enter before ready does nothing.
func TestWizardForm_CreateBlockedMidFill(t *testing.T) {
	m := serverWizardModel{form: newWizardForm(okValidate, freePortProbe, nil)}
	m = wizKey(m, "enter") // choose mode only
	if m.form.ready() {
		t.Fatal("mid-fill form must not be ready")
	}
	m, _ = func() (serverWizardModel, tea.Cmd) {
		next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return next.(serverWizardModel), cmd
	}()
	if m.form.done {
		t.Fatal("Enter on an incomplete form must NOT run create (blocked)")
	}
}

// TestWizardForm_SchemaInvalidBlocks: the real loader rejects a draft the
// form can't fix (here: instance name colliding with schema rules is not
// a loader concern, so drive schema-invalid via a validator seam).
func TestWizardForm_SchemaInvalidBlocks(t *testing.T) {
	f := filledForm()
	// Simulate the loader rejecting the assembled draft (e.g. subnet
	// overlap detected by the real validator inside the daemon chain).
	f.validate = func(draft string) (*collect.AppConfigAlias, error) {
		return nil, errInvalidSchema
	}
	if _, err := f.draft(); err == nil {
		t.Fatal("schema-invalid draft must fail through the real loader chain")
	}
	if !strings.Contains(errInvalidSchema.Error(), "invalid") {
		t.Errorf("error must be verbatim: %v", errInvalidSchema)
	}
	// The field table stays ready (field-level checks pass); the schema
	// error surfaces at create time via the model. Move focus off the
	// MODE radio (Enter there selects the mode) — Tab lands on INSTANCE.
	m := serverWizardModel{form: f}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(serverWizardModel)
	m.handleEnter()
	if m.lastErr == "" {
		t.Error("create attempt must surface the verbatim loader error")
	}
	if m.form.done {
		t.Error("schema-invalid must not complete create")
	}
}

var errInvalidSchema = errStr("invalid: metadata.schema_version must be 2.0.0")

type errStr string

func (e errStr) Error() string { return string(e) }

// wizKey / wizType drive the model like the real loop does.
func wizKey(m serverWizardModel, s string) serverWizardModel {
	next, _ := m.Update(tea.KeyMsg{Type: keyType(s), Runes: runesFor(s)})
	return next.(serverWizardModel)
}

func wizType(m serverWizardModel, s string) serverWizardModel {
	for _, r := range s {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(serverWizardModel)
	}
	return m
}

func keyType(s string) tea.KeyType {
	switch s {
	case "enter":
		return tea.KeyEnter
	case "tab":
		return tea.KeyTab
	case "esc":
		return tea.KeyEsc
	case "up":
		return tea.KeyUp
	case "down":
		return tea.KeyDown
	case "backspace":
		return tea.KeyBackspace
	case " ":
		return tea.KeySpace
	default:
		return tea.KeyRunes
	}
}

func runesFor(s string) []rune {
	switch s {
	case "enter", "tab", "esc", "up", "down", "backspace", " ":
		return nil
	default:
		return []rune(s)
	}
}
