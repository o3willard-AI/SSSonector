package collect

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SystemdPaths is the filesystem layout the collector probes. Overridable
// in tests; defaults match the documented install layout.
type SystemdPaths struct {
	// ConfigRoot is /etc/sssonector on a real host.
	ConfigRoot string
}

// DefaultSystemdPaths returns the real host layout.
func DefaultSystemdPaths() SystemdPaths {
	return SystemdPaths{ConfigRoot: "/etc/sssonector"}
}

// InstanceState is the systemd state of one sssonector instance.
type InstanceState struct {
	// Name is the template unit's %i (instance name), or "default" for the
	// legacy single-instance unit sssonector.service.
	Name string
	// Unit is the full systemd unit name (e.g. "sssonector@client-a.service").
	Unit string
	// ActiveState is systemd's ActiveState ("active", "inactive", "failed", ...).
	ActiveState string
	// SubState is systemd's SubState ("running", "dead", "exited", ...).
	SubState string
	// MainPID is the unit's main process PID, 0 when not running.
	MainPID int
}

// Running reports whether the unit's main process is up.
func (s InstanceState) Running() bool {
	return s.ActiveState == "active" && s.MainPID > 0
}

// CommandRunner executes a systemctl-style command and returns its stdout.
// All systemd collectors run through this interface so tests inject fake
// output and never exec a real systemctl.
type CommandRunner interface {
	// Run returns combined-trimmed stdout for the given argv.
	Run(args ...string) (string, error)
}

// OSCommandRunner is the production CommandRunner: real os/exec.
type OSCommandRunner struct{}

// Run executes argv via os/exec and returns trimmed stdout.
func (OSCommandRunner) Run(args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("exec: empty argv")
	}
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// SystemdCollector discovers sssonector instances via systemctl.
type SystemdCollector struct {
	Runner CommandRunner
	Paths  SystemdPaths
}

// discovery choice (explicit): `systemctl list-units 'sssonector@*' --all`
// is used because plain list-units hides inactive units, and the dashboard
// must show stopped ("down") instances. --all includes inactive/failed
// units that are still loaded. Units that are not loaded at all (never
// started since boot) are out of scope here: they appear in WI 1.2's
// config resolution, not from systemd.
const listUnitsArgs = "list-units sssonector@* --all --plain --no-legend"

// listUnitsArgv is the exact argv Discover passes to the runner.
var listUnitsArgv = []string{"systemctl", "list-units", "sssonector@*", "--all", "--plain", "--no-legend"}

// Discover returns the states of all sssonector@* template instances plus
// the legacy sssonector.service unit when it exists. On runner failure it
// applies the spec §7 fail-closed degradation: a single implicit instance
// ONLY when exactly one config exists under the config root; otherwise an
// error. It never fabricates an instance list.
func (c SystemdCollector) Discover() ([]InstanceState, error) {
	out, listErr := c.Runner.Run(listUnitsArgv...)
	if listErr == nil {
		states, err := parseListUnits(out)
		if err != nil {
			return nil, fmt.Errorf("systemd: parse list-units: %w", err)
		}
		// The legacy single-instance unit is discovered separately so both
		// layouts can coexist on a migrating host.
		if legacy, err := c.discoverLegacy(); err == nil && legacy != nil {
			states = append(states, *legacy)
		}
		return c.fillStates(states)
	}

	// Runner failed: degrade only on the exactly-one-config rule.
	implicit, err := implicitInstance(c.Paths.ConfigRoot)
	if err != nil {
		return nil, fmt.Errorf("systemd: %w (systemctl unavailable: %v)", err, listErr)
	}
	return []InstanceState{*implicit}, nil
}

// discoverLegacy returns the legacy unit state if sssonector.service is
// loaded; nil when absent.
func (c SystemdCollector) discoverLegacy() (*InstanceState, error) {
	out, err := c.Runner.Run("systemctl", "show", "sssonector.service",
		"-p", "ActiveState", "-p", "SubState", "-p", "MainPID")
	if err != nil {
		return nil, err
	}
	st, err := parseShow(out)
	if err != nil {
		return nil, err
	}
	if st.ActiveState == "" || st.ActiveState == "not-found" {
		return nil, nil
	}
	st.Name = "default"
	st.Unit = "sssonector.service"
	return &st, nil
}

// fillStates queries per-unit detail for every discovered instance.
func (c SystemdCollector) fillStates(states []InstanceState) ([]InstanceState, error) {
	for i := range states {
		out, err := c.Runner.Run("systemctl", "show", states[i].Unit,
			"-p", "ActiveState", "-p", "SubState", "-p", "MainPID")
		if err != nil {
			return nil, fmt.Errorf("systemd: show %s: %w", states[i].Unit, err)
		}
		st, err := parseShow(out)
		if err != nil {
			return nil, fmt.Errorf("systemd: show %s: %w", states[i].Unit, err)
		}
		states[i].ActiveState = st.ActiveState
		states[i].SubState = st.SubState
		states[i].MainPID = st.MainPID
	}
	return states, nil
}

// ImplicitInstanceForTest exposes the exactly-one-config degradation rule
// for WI 4.4's degradation-matrix tests (same logic Discover uses).
func ImplicitInstanceForTest(configRoot string) (*InstanceState, error) {
	return implicitInstance(configRoot)
}

// implicitInstance implements the exactly-one-config degradation rule.
// Exactly one of: /etc/sssonector/config.yaml ("default"), or exactly one
// /etc/sssonector/instances/<n>/config.yaml (<n>). Zero or 2+ => error.
func implicitInstance(configRoot string) (*InstanceState, error) {
	legacy := filepath.Join(configRoot, "config.yaml")
	_, legacyErr := os.Stat(legacy)

	instancesDir := filepath.Join(configRoot, "instances")
	var multi []string
	if entries, err := os.ReadDir(instancesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			cfg := filepath.Join(instancesDir, e.Name(), "config.yaml")
			if _, err := os.Stat(cfg); err == nil {
				multi = append(multi, e.Name())
			}
		}
	}

	switch {
	case legacyErr == nil && len(multi) > 0:
		return nil, errors.New("degradation ambiguous: both legacy config and instance configs present")
	case legacyErr == nil && len(multi) == 0:
		return &InstanceState{Name: "default", Unit: "sssonector.service", ActiveState: "unknown", SubState: "unknown"}, nil
	case legacyErr != nil && len(multi) == 1:
		return &InstanceState{Name: multi[0], Unit: "sssonector@" + multi[0] + ".service", ActiveState: "unknown", SubState: "unknown"}, nil
	case legacyErr != nil && len(multi) == 0:
		return nil, errors.New("degradation impossible: systemctl unavailable and no sssonector config found")
	default:
		return nil, fmt.Errorf("degradation ambiguous: %d instance configs present (%v)", len(multi), multi)
	}
}

// parseListUnits parses `systemctl list-units 'sssonector@*' --all --plain
// --no-legend` output: whitespace-separated columns UNIT LOAD ACTIVE SUB
// DESCRIPTION. Only the UNIT column is authoritative here; state comes
// from `show` (list-units ACTIVE/SUB may be stale during transitions).
func parseListUnits(out string) ([]InstanceState, error) {
	var states []InstanceState
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			return nil, fmt.Errorf("malformed list-units line %q", line)
		}
		unit := fields[0]
		name, ok := instanceNameFromUnit(unit)
		if !ok {
			continue // not a template unit row (e.g. filtered legacy row)
		}
		states = append(states, InstanceState{Name: name, Unit: unit})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return states, nil
}

// instanceNameFromUnit extracts %i from "sssonector@<name>.service".
func instanceNameFromUnit(unit string) (string, bool) {
	const prefix = "sssonector@"
	const suffix = ".service"
	if !strings.HasPrefix(unit, prefix) || !strings.HasSuffix(unit, suffix) {
		return "", false
	}
	name := unit[len(prefix) : len(unit)-len(suffix)]
	if name == "" {
		return "", false
	}
	return name, true
}

// parseShow parses `systemctl show <unit> -p ActiveState -p SubState
// -p MainPID` output: key=value lines in any order.
func parseShow(out string) (InstanceState, error) {
	var st InstanceState
	sc := bufio.NewScanner(strings.NewReader(out))
	seen := 0
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return st, fmt.Errorf("malformed show line %q", line)
		}
		switch k {
		case "ActiveState":
			st.ActiveState = v
			seen++
		case "SubState":
			st.SubState = v
			seen++
		case "MainPID":
			pid, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return st, fmt.Errorf("malformed MainPID %q", v)
			}
			st.MainPID = pid
			seen++
		default:
			// Other properties are ignored; systemctl may emit extras.
		}
	}
	if err := sc.Err(); err != nil {
		return st, err
	}
	if seen < 3 {
		return st, fmt.Errorf("incomplete show output: got %d of 3 expected keys", seen)
	}
	return st, nil
}
