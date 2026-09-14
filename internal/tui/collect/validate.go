package collect

import (
	cfg "github.com/o3willard-AI/SSSonector/internal/config"
)

// ValidateDraft runs a draft config TEXT through the daemon's EXACT
// loader + validator chain — no re-implementation, no extra normalization:
//
//  1. cfg.LoadConfigString(draft, "yaml") — schema load/merge. Parse
//     errors carry "line N" (yaml.v3) and are returned verbatim.
//  2. cfg.NewValidator().Validate(app) — the daemon's semantic checks
//     (subnet overlap, listen_port ranges, NAT default-deny invariants, …),
//     returned verbatim.
//  3. On success, the merged AppConfig is returned.
//
// Fail-closed: an invalid draft is an error and is never treated as a
// usable config — nothing is written and nothing is signaled (the write +
// SIGHUP pipeline is WI 3.3). A no-op draft (the WI 3.1 dump text,
// unchanged) round-trips as ok because the same loader that produced the
// dump re-parses it.
func ValidateDraft(draft string) (*cfg.AppConfig, error) {
	app, err := cfg.LoadConfigString(draft, "yaml")
	if err != nil {
		return nil, err // verbatim (carries "line N" for syntax errors)
	}
	if err := cfg.NewValidator().Validate(app); err != nil {
		return nil, err // verbatim semantic error
	}
	return app, nil
}
