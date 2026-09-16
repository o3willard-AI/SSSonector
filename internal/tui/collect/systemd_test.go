package collect

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner is an injectable CommandRunner returning canned output per
// subcommand. It never execs a real systemctl.
type fakeRunner struct {
	outputs map[string]string
	errs    map[string]error
	calls   [][]string
	// bootID lets logtail tests inject the boot id (empty = not set).
	bootID string
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	key := strings.Join(args, " ")
	if err, ok := f.errs[key]; ok {
		return "", err
	}
	if out, ok := f.outputs[key]; ok {
		return out, nil
	}
	return "", nil
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{}, errs: map[string]error{}}
}

func TestSystemd_Discover(t *testing.T) {
	t.Run("two running template units parse with state", func(t *testing.T) {
		r := newFakeRunner()
		r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
			"sssonector@client-a.service loaded active running SSSonector tunnel (client-a)\n" +
				"sssonector@client-b.service loaded active running SSSonector tunnel (client-b)\n"
		r.outputs["systemctl show sssonector@client-a.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"ActiveState=active\nSubState=running\nMainPID=8123\n"
		r.outputs["systemctl show sssonector@client-b.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"ActiveState=active\nSubState=running\nMainPID=8124\n"
		r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"ActiveState=not-found\nSubState=dead\nMainPID=0\nLoadState=not-found\n" // legacy absent

		c := SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()}
		got, err := c.Discover()
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d instances, want 2: %+v", len(got), got)
		}
		if got[0].Name != "client-a" || got[0].MainPID != 8123 || got[0].ActiveState != "active" || got[0].SubState != "running" {
			t.Errorf("client-a: %+v", got[0])
		}
		if !got[0].Running() {
			t.Error("client-a should be Running")
		}
		if got[1].Name != "client-b" || got[1].MainPID != 8124 {
			t.Errorf("client-b: %+v", got[1])
		}
	})

	t.Run("stopped unit included (explicit --all choice)", func(t *testing.T) {
		r := newFakeRunner()
		r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
			"sssonector@client-a.service loaded active running SSSonector tunnel (client-a)\n" +
				"sssonector@client-b.service loaded inactive dead SSSonector tunnel (client-b)\n" +
				"sssonector@client-c.service loaded failed failed SSSonector tunnel (client-c)\n"
		for _, n := range []string{"client-a", "client-b", "client-c"} {
			r.outputs["systemctl show sssonector@"+n+".service -p ActiveState -p SubState -p MainPID -p LoadState"] =
				map[string]string{
					"client-a": "ActiveState=active\nSubState=running\nMainPID=100\n",
					"client-b": "ActiveState=inactive\nSubState=dead\nMainPID=0\n",
					"client-c": "ActiveState=failed\nSubState=failed\nMainPID=0\n",
				}[n]
		}
		r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"ActiveState=not-found\nSubState=dead\nMainPID=0\n"

		c := SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()}
		got, err := c.Discover()
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d instances, want 3 (stopped/failed must be represented, not dropped): %+v", len(got), got)
		}
		byName := map[string]InstanceState{}
		for _, s := range got {
			byName[s.Name] = s
		}
		if byName["client-b"].Running() {
			t.Error("client-b (inactive) must not be Running")
		}
		if byName["client-b"].SubState != "dead" {
			t.Errorf("client-b SubState: %q", byName["client-b"].SubState)
		}
		if byName["client-c"].ActiveState != "failed" {
			t.Errorf("client-c ActiveState: %q", byName["client-c"].ActiveState)
		}
		if byName["client-c"].MainPID != 0 {
			t.Errorf("failed unit MainPID must be 0, got %d", byName["client-c"].MainPID)
		}
	})

	t.Run("legacy single-instance unit discovered as default", func(t *testing.T) {
		r := newFakeRunner()
		r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] = ""
		r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"ActiveState=active\nSubState=running\nMainPID=555\nLoadState=loaded\n"

		c := SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()}
		got, err := c.Discover()
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(got) != 1 || got[0].Name != "default" || got[0].MainPID != 555 {
			t.Fatalf("legacy discovery: %+v err=%v", got, err)
		}
	})

	t.Run("legacy unit not-found is NOT discovered (LoadState authority)", func(t *testing.T) {
		// Ubuntu 24.04 quirk: an unknown unit reports ActiveState=inactive
		// (not not-found) — LoadState=not-found is the only reliable signal.
		// Regression: a fresh host fabricated a phantom "default" instance,
		// which blocked the WI 5.1 first-run wizard route.
		r := newFakeRunner()
		r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] = ""
		r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"MainPID=0\nActiveState=inactive\nSubState=dead\nLoadState=not-found\n"

		c := SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()}
		got, err := c.Discover()
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("not-found legacy unit must not be discovered: %+v", got)
		}
	})
}

func TestSystemd_Degradation(t *testing.T) {
	// Build a fake config root on disk (os.Stat/ReadDir only — no
	// internal/config import).
	newRoot := func(t *testing.T) string {
		return t.TempDir()
	}
	writeLegacy := func(root string) {
		if err := os.MkdirAll(root, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("mode: client\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeInstance := func(root, name string) {
		dir := filepath.Join(root, "instances", name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("mode: server\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("runner error + legacy config only => implicit default", func(t *testing.T) {
		root := newRoot(t)
		writeLegacy(root)
		c := SystemdCollector{Runner: errRunner{}, Paths: SystemdPaths{ConfigRoot: root}}
		got, err := c.Discover()
		if err != nil {
			t.Fatalf("want implicit instance, got error: %v", err)
		}
		if len(got) != 1 || got[0].Name != "default" {
			t.Errorf("got %+v, want [default]", got)
		}
	})

	t.Run("runner error + exactly one instance config => implicit <n>", func(t *testing.T) {
		root := newRoot(t)
		writeInstance(root, "client-a")
		c := SystemdCollector{Runner: errRunner{}, Paths: SystemdPaths{ConfigRoot: root}}
		got, err := c.Discover()
		if err != nil {
			t.Fatalf("want implicit instance, got error: %v", err)
		}
		if len(got) != 1 || got[0].Name != "client-a" {
			t.Errorf("got %+v, want [client-a]", got)
		}
	})

	t.Run("runner error + zero configs => error", func(t *testing.T) {
		c := SystemdCollector{Runner: errRunner{}, Paths: SystemdPaths{ConfigRoot: newRoot(t)}}
		_, err := c.Discover()
		if err == nil || !strings.Contains(err.Error(), "no sssonector config found") {
			t.Errorf("want explicit error, got %v", err)
		}
	})

	t.Run("runner error + two instance configs => error", func(t *testing.T) {
		root := newRoot(t)
		writeInstance(root, "client-a")
		writeInstance(root, "client-b")
		c := SystemdCollector{Runner: errRunner{}, Paths: SystemdPaths{ConfigRoot: root}}
		_, err := c.Discover()
		if err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Errorf("want ambiguous error, got %v", err)
		}
	})

	t.Run("runner error + legacy AND instance configs => error", func(t *testing.T) {
		root := newRoot(t)
		writeLegacy(root)
		writeInstance(root, "client-a")
		c := SystemdCollector{Runner: errRunner{}, Paths: SystemdPaths{ConfigRoot: root}}
		_, err := c.Discover()
		if err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Errorf("want ambiguous error, got %v", err)
		}
	})
}

// errRunner always fails (systemd absent / non-root).
type errRunner struct{}

func (errRunner) Run(args ...string) (string, error) {
	return "", errors.New("System has not been booted with systemd")
}

func TestSystemd_MalformedOutput(t *testing.T) {
	t.Run("malformed list-units line", func(t *testing.T) {
		r := newFakeRunner()
		r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
			"sssonector@client-a.service one-single-column\n"
		c := SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()}
		_, err := c.Discover()
		if err == nil || !strings.Contains(err.Error(), "malformed list-units") {
			t.Errorf("want parse error, got %v", err)
		}
	})
	t.Run("malformed show line", func(t *testing.T) {
		r := newFakeRunner()
		r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
			"sssonector@client-a.service loaded active running x\n"
		r.outputs["systemctl show sssonector@client-a.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"ActiveState=active\nSubState=running\nMainPID=not-a-pid\n"
		c := SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()}
		_, err := c.Discover()
		if err == nil || !strings.Contains(err.Error(), "MainPID") {
			t.Errorf("want MainPID error, got %v", err)
		}
	})
	t.Run("incomplete show output", func(t *testing.T) {
		r := newFakeRunner()
		r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
			"sssonector@client-a.service loaded active running x\n"
		r.outputs["systemctl show sssonector@client-a.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
			"ActiveState=active\n"
		c := SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()}
		_, err := c.Discover()
		if err == nil || !strings.Contains(err.Error(), "incomplete show output") {
			t.Errorf("want incomplete error, got %v", err)
		}
	})
}

func TestSystemd_ParseUnits(t *testing.T) {
	t.Run("instance name extraction", func(t *testing.T) {
		if n, ok := instanceNameFromUnit("sssonector@client-a.service"); !ok || n != "client-a" {
			t.Errorf("got %q ok=%v", n, ok)
		}
		if _, ok := instanceNameFromUnit("sssonector@.service"); ok {
			t.Error("empty %i must not parse")
		}
		if _, ok := instanceNameFromUnit("sssonector.service"); ok {
			t.Error("legacy unit is not a template row")
		}
		if _, ok := instanceNameFromUnit("ssh.service"); ok {
			t.Error("non-sssonector unit must not parse")
		}
	})
	t.Run("show parser tolerates extra properties", func(t *testing.T) {
		st, err := parseShow("Id=sssonector@x.service\nActiveState=active\nSubState=running\nMainPID=9\n")
		if err != nil || st.MainPID != 9 || st.ActiveState != "active" {
			t.Errorf("got %+v err=%v", st, err)
		}
	})
	t.Run("parseShow rejects non-numeric pid", func(t *testing.T) {
		_, err := parseShow("ActiveState=active\nSubState=running\nMainPID=abc\n")
		if err == nil {
			t.Error("want error")
		}
	})
}
