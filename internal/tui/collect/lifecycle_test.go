package collect

import (
	"errors"
	"strings"
	"testing"
)

var errNoopLifecycle = errors.New("noop")

// recordingRunner captures argv and returns canned results. Reuses the
// CommandRunner seam — no real systemctl is ever invoked.
type recordingRunner struct {
	calls [][]string
	err   error
}

func (r *recordingRunner) Run(args ...string) (string, error) {
	r.calls = append(r.calls, args)
	if r.err != nil {
		return "", r.err
	}
	return "", nil
}

func TestStopInstance_RunsSystemctlStopOnUnit(t *testing.T) {
	r := &recordingRunner{}
	if err := StopInstance(r, "sssonector@client-a.service"); err != nil {
		t.Fatalf("StopInstance: %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("runner calls: %d, want exactly 1", len(r.calls))
	}
	got := strings.Join(r.calls[0], " ")
	if got != "systemctl stop sssonector@client-a.service" {
		t.Errorf("argv: %q", got)
	}
}

func TestRestartInstance_RunsSystemctlRestartOnUnit(t *testing.T) {
	r := &recordingRunner{}
	if err := RestartInstance(r, "sssonector@client-b.service"); err != nil {
		t.Fatalf("RestartInstance: %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("runner calls: %d, want exactly 1", len(r.calls))
	}
	got := strings.Join(r.calls[0], " ")
	if got != "systemctl restart sssonector@client-b.service" {
		t.Errorf("argv: %q", got)
	}
}

func TestLifecycle_FailClosed_NilRunnerEmptyUnit(t *testing.T) {
	if err := lifecycleCall(nil, LifecycleStop, "sssonector@a.service"); err == nil {
		t.Error("nil runner must error")
	}
	r := &recordingRunner{}
	if err := StopInstance(r, ""); err == nil {
		t.Error("empty unit must error without calling the runner")
	}
	if len(r.calls) != 0 {
		t.Errorf("empty unit must issue zero runner calls: %v", r.calls)
	}
}

func TestLifecycle_RunnerErrorSurfaced(t *testing.T) {
	r := &recordingRunner{err: errNoopLifecycle}
	if err := StopInstance(r, "sssonector@a.service"); err == nil {
		t.Fatal("runner error must surface")
	}
	if len(r.calls) != 1 {
		t.Errorf("calls: %d, want 1 (failed call still counts)", len(r.calls))
	}
}

func TestUnitNameFor_TemplateAndLegacy(t *testing.T) {
	if got := UnitNameFor("client-a"); got != "sssonector@client-a.service" {
		t.Errorf("template unit: %q", got)
	}
	if got := UnitNameFor("default"); got != "sssonector.service" {
		t.Errorf("legacy unit: %q", got)
	}
}
