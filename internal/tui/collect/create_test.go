package collect

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WI 5.3: wizard write-path tests. The runner records argv (no real
// systemctl); the config file's on-disk existence is checked BEFORE the
// enable call runs to prove write-before-unit ordering.

// recordingCreateRunner records argv and can fail specific verbs.
type recordingCreateRunner struct {
	calls [][]string
	// failOn returns an error for matching argv (checked before recording
	// — a failed call is still recorded).
	failOn func(args []string) error
	// beforeCall runs prior to each call (used to assert file-on-disk
	// ordering from inside the runner).
	beforeCall func(args []string) error
}

func (r *recordingCreateRunner) Run(args ...string) (string, error) {
	if r.beforeCall != nil {
		if err := r.beforeCall(args); err != nil {
			r.calls = append(r.calls, args)
			return "", err
		}
	}
	r.calls = append(r.calls, args)
	if r.failOn != nil {
		if err := r.failOn(args); err != nil {
			return "", err
		}
	}
	return "", nil
}

func (r *recordingCreateRunner) verbs() []string {
	var v []string
	for _, c := range r.calls {
		if len(c) >= 2 {
			v = append(v, c[1])
		}
	}
	return v
}

const createDraft = "metadata:\n  schema_version: \"2.0.0\"\n  environment: development\ntype: server\nconfig:\n  mode: server\n"

// TestCreateAndStart_Ordering_FileBeforeUnit: the config file exists on
// disk BEFORE systemctl enable/start is invoked (checked from inside the
// runner), and enable precedes start.
func TestCreateAndStart_Ordering_FileBeforeUnit(t *testing.T) {
	root := t.TempDir()
	r := &recordingCreateRunner{}
	fileExistedAtEnable := false
	r.beforeCall = func(args []string) error {
		if len(args) >= 2 && args[1] == "enable" {
			_, err := os.Stat(filepath.Join(root, "instances", "client-a", "config.yaml"))
			fileExistedAtEnable = err == nil
		}
		return nil
	}
	res, err := CreateAndStartInstance(ConfigPaths{ConfigRoot: root}, "client-a", createDraft, r)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !fileExistedAtEnable {
		t.Error("§ordering: config file must exist on disk BEFORE systemctl enable runs")
	}
	if got := strings.Join(r.verbs(), ","); got != "enable,start" {
		t.Errorf("runner order: %v, want enable,start", r.verbs())
	}
	if len(r.calls) != 2 {
		t.Fatalf("runner calls: %d, want exactly 2", len(r.calls))
	}
	if got := strings.Join(r.calls[0], " "); got != "systemctl enable sssonector@client-a.service" {
		t.Errorf("enable argv: %q", got)
	}
	if got := strings.Join(r.calls[1], " "); got != "systemctl start sssonector@client-a.service" {
		t.Errorf("start argv: %q", got)
	}
	if !res.Started || res.Unit != "sssonector@client-a.service" {
		t.Errorf("result: %+v", res)
	}
	// File content matches the draft, 0600.
	info, err := os.Stat(res.File)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("config mode: %v, want 0600", info.Mode().Perm())
	}
}

// TestCreateAndStart_WriteFails_ZeroRunnerCalls: a write failure (target
// path is a directory) must NEVER invoke enable/start.
func TestCreateAndStart_WriteFails_ZeroRunnerCalls(t *testing.T) {
	root := t.TempDir()
	// Make instances/client-a/config.yaml a DIRECTORY: the rename commit
	// point fails.
	if err := os.MkdirAll(filepath.Join(root, "instances", "client-a", "config.yaml"), 0o750); err != nil {
		t.Fatal(err)
	}
	r := &recordingCreateRunner{}
	_, err := CreateAndStartInstance(ConfigPaths{ConfigRoot: root}, "client-a", createDraft, r)
	if err == nil {
		t.Fatal("write failure must error")
	}
	if len(r.calls) != 0 {
		t.Errorf("FAIL-CLOSED VIOLATED: write failed but runner was called: %v", r.calls)
	}
	if !strings.Contains(err.Error(), "create: write") {
		t.Errorf("error must surface verbatim: %v", err)
	}
}

// TestCreateAndStart_EnableFails_FileStaysWritten: enable failure leaves
// the config file on disk (documented: no rollback, matching Phase 3's
// SIGHUP semantics) and start is never invoked.
func TestCreateAndStart_EnableFails_FileStaysWritten(t *testing.T) {
	root := t.TempDir()
	r := &recordingCreateRunner{failOn: func(args []string) error {
		if len(args) >= 2 && args[1] == "enable" {
			return errors.New("systemctl enable exploded")
		}
		return nil
	}}
	_, err := CreateAndStartInstance(ConfigPaths{ConfigRoot: root}, "client-a", createDraft, r)
	if err == nil {
		t.Fatal("enable failure must error")
	}
	if !strings.Contains(err.Error(), "systemctl enable exploded") {
		t.Errorf("error must surface verbatim: %v", err)
	}
	// File stays written — no rollback.
	if _, statErr := os.Stat(filepath.Join(root, "instances", "client-a", "config.yaml")); statErr != nil {
		t.Errorf("config file must STAY WRITTEN after enable failure: %v", statErr)
	}
	// start must NOT have run after the enable failure.
	if got := strings.Join(r.verbs(), ","); got != "enable" {
		t.Errorf("verbs after enable failure: %v, want [enable]", r.verbs())
	}
	if strings.Contains(err.Error(), "left written") == false {
		t.Errorf("error must document the left-written semantics: %v", err)
	}
}

// TestCreateAndStart_StartFails_ErrorSurfaces: start failing after a good
// enable still leaves the file and stops the sequence.
func TestCreateAndStart_StartFails_ErrorSurfaces(t *testing.T) {
	root := t.TempDir()
	r := &recordingCreateRunner{failOn: func(args []string) error {
		if len(args) >= 2 && args[1] == "start" {
			return errors.New("unit failed to start")
		}
		return nil
	}}
	_, err := CreateAndStartInstance(ConfigPaths{ConfigRoot: root}, "client-a", createDraft, r)
	if err == nil {
		t.Fatal("start failure must error")
	}
	if got := strings.Join(r.verbs(), ","); got != "enable,start" {
		t.Errorf("verbs: %v, want enable,start", r.verbs())
	}
	if _, statErr := os.Stat(filepath.Join(root, "instances", "client-a", "config.yaml")); statErr != nil {
		t.Errorf("config file must STAY WRITTEN after start failure: %v", statErr)
	}
}

// TestCreateAndStart_Success: file written → enable → start in order.
func TestCreateAndStart_Success(t *testing.T) {
	root := t.TempDir()
	r := &recordingCreateRunner{}
	res, err := CreateAndStartInstance(ConfigPaths{ConfigRoot: root}, "client-a", createDraft, r)
	if err != nil {
		t.Fatalf("success path: %v", err)
	}
	if !res.Started {
		t.Error("Started must be true on the success path")
	}
	written, err := os.ReadFile(res.File)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != createDraft {
		t.Errorf("written content must be the validated draft:\n%s", written)
	}
	if res.Unit != "sssonector@client-a.service" {
		t.Errorf("unit: %q", res.Unit)
	}
}

// TestCreateAndStart_FailClosed_NilRunnerEmptyInstance.
func TestCreateAndStart_FailClosed(t *testing.T) {
	root := t.TempDir()
	if _, err := CreateAndStartInstance(ConfigPaths{ConfigRoot: root}, "client-a", createDraft, nil); err == nil {
		t.Error("nil runner must error with zero side effects")
	}
	r := &recordingCreateRunner{}
	if _, err := CreateAndStartInstance(ConfigPaths{ConfigRoot: root}, "", createDraft, r); err == nil {
		t.Error("empty instance must error")
	}
	if len(r.calls) != 0 {
		t.Errorf("empty instance must issue zero runner calls: %v", r.calls)
	}
}
