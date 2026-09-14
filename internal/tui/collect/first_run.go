package collect

import (
	"os"
	"path/filepath"
)

// DiscoveryState is the answer to "what does this host look like?" for
// first-run mode selection (WI 5.1). It carries exactly what the dashboard
// would have discovered — the wizard detection reuses the same collector
// and config fallback, never a second discovery path.
type DiscoveryState struct {
	// Units is the discovered sssonector@* (and legacy) unit list. Empty
	// when discovery failed.
	Units []InstanceState
	// DiscoverErr is the discovery error (systemctl unavailable etc.).
	DiscoverErr error
	// ConfigCount is the number of instance configs found under the root:
	// 1 for the legacy single-instance config (counted as the "default"
	// config), plus each instances/<n>/config.yaml.
	ConfigCount int
	// Ambiguous is true when the host has 2+ configs (or the legacy +
	// instance mix) — an explicit error the DASHBOARD renders, never a
	// wizard trigger.
	Ambiguous bool
}

// DiscoverForModeSelection answers the mode-selection question using the
// SAME discovery the dashboard uses: the systemd collector (units) plus
// the exactly-one-config config-root scan (configs). It does not start a
// second discovery path — it calls the collector's Discover and counts
// configs with the same layout rules implicitInstance uses.
func DiscoverForModeSelection(sys SystemdCollector, configRoot string) (DiscoveryState, error) {
	st := DiscoveryState{}
	states, err := sys.Discover()
	if err != nil {
		st.DiscoverErr = err
	} else {
		st.Units = states
	}
	st.ConfigCount, st.Ambiguous = countInstanceConfigs(configRoot)
	return st, nil
}

// countInstanceConfigs counts instance configs under the config root with
// the same layout rules implicitInstance applies: /config.yaml (legacy,
// "default") and instances/<n>/config.yaml. Ambiguous mirrors
// implicitInstance's error cases (both layouts, or 2+ instance configs).
func countInstanceConfigs(configRoot string) (int, bool) {
	legacy := filepath.Join(configRoot, "config.yaml")
	_, legacyErr := os.Stat(legacy)

	var multi []string
	if entries, err := os.ReadDir(filepath.Join(configRoot, "instances")); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			cfg := filepath.Join(configRoot, "instances", e.Name(), "config.yaml")
			if _, err := os.Stat(cfg); err == nil {
				multi = append(multi, e.Name())
			}
		}
	}

	count := len(multi)
	if legacyErr == nil {
		count++
	}
	ambiguous := (legacyErr == nil && len(multi) > 0) || len(multi) > 1
	return count, ambiguous
}
