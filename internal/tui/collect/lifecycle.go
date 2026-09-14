package collect

import (
	"fmt"
)

// LifecycleAction is a systemctl lifecycle verb.
type LifecycleAction int

const (
	// LifecycleStop is `systemctl stop <unit>`.
	LifecycleStop LifecycleAction = iota
	// LifecycleRestart is `systemctl restart <unit>`.
	LifecycleRestart
)

// String renders the verb.
func (a LifecycleAction) String() string {
	if a == LifecycleStop {
		return "stop"
	}
	return "restart"
}

// StopInstance runs `systemctl stop <unit>` via the injected runner.
// Tests inject a fake runner — no real systemctl is ever invoked from
// unit tests.
func StopInstance(runner CommandRunner, unit string) error {
	return lifecycleCall(runner, LifecycleStop, unit)
}

// RestartInstance runs `systemctl restart <unit>` via the runner.
func RestartInstance(runner CommandRunner, unit string) error {
	return lifecycleCall(runner, LifecycleRestart, unit)
}

// lifecycleCall is the shared pipeline: fail-closed on nil runner or
// empty unit; exactly one runner call per invocation.
func lifecycleCall(runner CommandRunner, action LifecycleAction, unit string) error {
	if runner == nil {
		return fmt.Errorf("lifecycle: nil command runner")
	}
	if unit == "" {
		return fmt.Errorf("lifecycle: empty unit name")
	}
	// Exactly one systemctl call: `systemctl <verb> <unit>`.
	if _, err := runner.Run("systemctl", action.String(), unit); err != nil {
		return fmt.Errorf("lifecycle: systemctl %s %s: %w", action, unit, err)
	}
	return nil
}

// UnitNameFor returns the systemd unit for an instance name. Template
// layout: sssonector@<name>.service; the legacy single-instance layout is
// sssonector.service with instance name "default". The unit is ALWAYS
// derived from the instance name — never a hardcoded default.
func UnitNameFor(name string) string {
	if name == "default" {
		return "sssonector.service"
	}
	return "sssonector@" + name + ".service"
}
