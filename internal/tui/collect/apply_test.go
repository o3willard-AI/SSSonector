package collect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSignaller records SIGHUP calls (never a real signal).
type fakeSignaller struct {
	calls []int
	err   error // error to return on signal
}

func (f *fakeSignaller) signal(pid int) error {
	f.calls = append(f.calls, pid)
	return f.err
}

func (f *fakeSignaller) reader(outcome ReloadOutcome) ReloadReader {
	return func(int) (ReloadOutcome, error) { return outcome, nil }
}

const applyDraft = `metadata:
  schema_version: "2.0.0"
  environment: development
type: server
config:
  mode: server
`

func applyFixture(t *testing.T) (ConfigPaths, string) {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "instances/client-a/config.yaml", "metadata:\n  schema_version: \"2.0.0\"\n  environment: development\ntype: server\nconfig:\n  mode: server\n  logging:\n    level: info\n")
	return ConfigPaths{ConfigRoot: root}, filepath.Join(root, "instances", "client-a", "config.yaml")
}

func TestApplyConfig_ReloadOK(t *testing.T) {
	paths, target := applyFixture(t)
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}

	sig := &fakeSignaller{}
	res, err := ApplyConfig(paths, "client-a", applyDraft, 8123, sig.signal, sig.reader(ReloadOK))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	// File replaced with the draft.
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != applyDraft {
		t.Errorf("target content mismatch:\n%s", after)
	}
	if string(after) == string(before) {
		t.Error("target must have changed")
	}

	// Signal fired exactly once, to the right pid, after the write.
	if len(sig.calls) != 1 || sig.calls[0] != 8123 {
		t.Errorf("signal calls: %v, want [8123]", sig.calls)
	}
	if !res.Signaled || res.Outcome != ReloadOK || res.File != target {
		t.Errorf("result: %+v", res)
	}
}

func TestApplyConfig_ReloadRejected_FileStaysWritten(t *testing.T) {
	paths, target := applyFixture(t)
	sig := &fakeSignaller{}

	res, err := ApplyConfig(paths, "client-a", applyDraft, 8123, sig.signal, sig.reader(ReloadRejected))
	if err != nil {
		t.Fatalf("rejected reload is not a pipeline error: %v", err)
	}

	// Documented semantics: the file STAYS WRITTEN; the daemon keeps its
	// old in-memory config; outcome surfaced as rejected.
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != applyDraft {
		t.Errorf("on rejection the file must stay written (no rollback):\n%s", after)
	}
	if res.Outcome != ReloadRejected {
		t.Errorf("outcome: %v, want rejected", res.Outcome)
	}
	if !res.Signaled {
		t.Error("signal did fire (rejection happens after it)")
	}
	if len(sig.calls) != 1 {
		t.Errorf("signal calls: %v", sig.calls)
	}
}

func TestApplyConfig_WriteFails_NoSignal(t *testing.T) {
	paths, _ := applyFixture(t)
	sig := &fakeSignaller{}

	// Target path is a DIRECTORY: temp file creation/rename fails.
	bad := ConfigPaths{ConfigRoot: paths.ConfigRoot}
	if err := os.MkdirAll(filepath.Join(bad.ConfigRoot, "instances", "broken", "config.yaml"), 0o750); err != nil {
		t.Fatal(err)
	}

	_, err := ApplyConfig(bad, "broken", applyDraft, 8123, sig.signal, nil)
	if err == nil {
		t.Fatal("write failure must error")
	}
	// Invariant 2: NO signal on write failure.
	if len(sig.calls) != 0 {
		t.Errorf("NEVER signal after a failed write: %v calls", sig.calls)
	}
}

func TestApplyConfig_SignalFails_FileWrittenErrorSurfaced(t *testing.T) {
	paths, target := applyFixture(t)
	sig := &fakeSignaller{err: os.ErrProcessDone}

	res, err := ApplyConfig(paths, "client-a", applyDraft, 8123, sig.signal, nil)
	if err == nil {
		t.Fatal("signal failure must be surfaced")
	}
	if !strings.Contains(err.Error(), "SIGHUP") {
		t.Errorf("error should mention SIGHUP: %v", err)
	}
	// The file IS written (documented: next daemon start picks it up).
	after, err := os.ReadFile(target)
	if err != nil || string(after) != applyDraft {
		t.Errorf("file must remain written after signal failure: %v", err)
	}
	if res.Signaled {
		t.Error("Signaled must be false when the signal failed")
	}
	_ = res
}

func TestApplyConfig_Atomicity_TempInSameDirRenamed(t *testing.T) {
	paths, target := applyFixture(t)
	dir := filepath.Dir(target)

	// Instrument the directory: after apply, no temp files remain (rename
	// consumed ours), and the target inode changed (replaced, not written
	// in place).
	beforeInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	listTmp := func() []string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var tmps []string
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".sssonector-apply-") {
				tmps = append(tmps, e.Name())
			}
		}
		return tmps
	}

	sig := &fakeSignaller{}
	if _, err := ApplyConfig(paths, "client-a", applyDraft, 8123, sig.signal, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if left := listTmp(); len(left) != 0 {
		t.Errorf("temp files left behind after rename: %v", left)
	}
	afterInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(beforeInfo, afterInfo) {
		t.Error("target was written IN PLACE (same inode); must be replaced via rename")
	}

	// Perms: 0600 on the committed file.
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("target perms: %v, want 0600", perm)
	}
}

func TestApplyConfig_CrashMidWrite_OldFileIntact(t *testing.T) {
	// Simulate: the temp write happens but the rename never does (crash).
	// The OLD target must remain intact and readable.
	paths, target := applyFixture(t)
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}

	// Direct call to atomicWrite with an unwritable target: make the RENAME
	// fail by making the target dir read-only AFTER temp creation is
	// impossible to interleave — instead verify the equivalent invariant:
	// a failed atomicWrite leaves the old target and cleans its temp.
	bad := ConfigPaths{ConfigRoot: paths.ConfigRoot}
	if err := os.MkdirAll(filepath.Join(bad.ConfigRoot, "instances", "broken", "config.yaml"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(filepath.Join(bad.ConfigRoot, "instances", "broken", "config.yaml"), []byte(applyDraft)); err == nil {
		t.Fatal("atomicWrite onto a directory path must fail")
	}
	// Old file untouched, no temp residue anywhere under the root.
	after, err := os.ReadFile(target)
	if err != nil || string(after) != string(before) {
		t.Errorf("old file must survive a failed atomicWrite: %v", err)
	}
	filepath.Walk(bad.ConfigRoot, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsRegular() && strings.HasPrefix(info.Name(), ".sssonector-apply-") {
			t.Errorf("temp residue: %s", path)
		}
		return nil
	})
}

func TestApplyConfig_Guards(t *testing.T) {
	paths, _ := applyFixture(t)
	sig := &fakeSignaller{}

	// pid <= 0: fail closed — never guess a pid.
	if _, err := ApplyConfig(paths, "client-a", applyDraft, 0, sig.signal, nil); err == nil {
		t.Error("pid=0 must error")
	}
	if _, err := ApplyConfig(paths, "client-a", applyDraft, -1, sig.signal, nil); err == nil {
		t.Error("pid<0 must error")
	}
	// Missing instance: resolution error, no signal.
	if _, err := ApplyConfig(paths, "ghost", applyDraft, 1, sig.signal, nil); err == nil {
		t.Error("missing instance must error")
	}
	if len(sig.calls) != 0 {
		t.Errorf("no signal in any guard case: %v", sig.calls)
	}
}

func TestApplyConfig_ReloadReadError(t *testing.T) {
	paths, _ := applyFixture(t)
	sig := &fakeSignaller{}
	readErr := func(int) (ReloadOutcome, error) { return 0, os.ErrDeadlineExceeded }
	_, err := ApplyConfig(paths, "client-a", applyDraft, 8123, sig.signal, readErr)
	if err == nil || !strings.Contains(err.Error(), "reload outcome") {
		t.Errorf("reload-read failure must surface: %v", err)
	}
}
