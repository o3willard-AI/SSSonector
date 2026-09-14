package collect

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	cfg "github.com/o3willard-AI/SSSonector/internal/config"
)

// ConfigPaths is the filesystem layout for instance configs. Overridable in
// tests; defaults match the documented install layout.
type ConfigPaths struct {
	// ConfigRoot is /etc/sssonector on a real host.
	ConfigRoot string
}

// DefaultConfigPaths returns the real host layout.
func DefaultConfigPaths() ConfigPaths {
	return ConfigPaths{ConfigRoot: "/etc/sssonector"}
}

// InstanceConfig is the typed extraction of the fields the dashboard needs
// from one instance's config file.
type InstanceConfig struct {
	// Name is the instance name the resolver was asked for.
	Name string
	// File is the config file that was loaded (resolved path).
	File string
	// Mode is config.mode (authoritative; a top-level type: exists on the
	// wrapper but config.mode is authoritative).
	Mode string
	// Prometheus endpoint settings.
	Prometheus PrometheusEndpoint
	// CertPaths are the TLS material paths from config.auth.*.
	CertPaths CertPaths
	// CertRotationInterval is config.auth.cert_rotation.interval; 0 means
	// the reader applies its default (30 days, matching internal/cert).
	CertRotationInterval time.Duration
	// TunAddr is config.network.address (the TUN address/prefix).
	TunAddr string
	// ListenPort is config.tunnel.listen_port (the server listener port).
	ListenPort int
}

// PrometheusEndpoint locates the instance's /metrics endpoint.
type PrometheusEndpoint struct {
	Enabled bool
	Port    int
	Path    string
}

// CertPaths holds the certificate/key/CA paths from config.auth.*.
type CertPaths struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// ResolveInstanceConfig locates the instance's config file and loads it
// with the daemon's own loader (config.LoadConfigFile — reused, not
// re-implemented), extracting the dashboard fields.
//
// Resolution order: instances/<name>/config.yaml first, then the legacy
// <root>/config.yaml for the "default" instance. Missing files and invalid
// schemas surface the loader's error verbatim — never swallowed, never
// defaulted (fail-closed; AGENTS.md security rules).
func ResolveInstanceConfig(paths ConfigPaths, name string) (InstanceConfig, error) {
	var resolved string

	instanceFile := filepath.Join(paths.ConfigRoot, "instances", name, "config.yaml")
	legacyFile := filepath.Join(paths.ConfigRoot, "config.yaml")

	if _, err := os.Stat(instanceFile); err == nil {
		resolved = instanceFile
	} else if _, err2 := os.Stat(legacyFile); err2 == nil {
		// Legacy fallback: single-instance layout.
		resolved = legacyFile
	} else {
		return InstanceConfig{}, fmt.Errorf("config: instance %q: no config file found (tried %s and %s)",
			name, instanceFile, legacyFile)
	}

	app, err := cfg.LoadConfigFile(resolved)
	if err != nil {
		// Surface the loader's error verbatim.
		return InstanceConfig{}, err
	}

	ic := InstanceConfig{
		Name: name,
		File: resolved,
		Mode: app.Config.Mode,
		Prometheus: PrometheusEndpoint{
			Enabled: app.Config.Monitor.Prometheus.Enabled,
			Port:    app.Config.Monitor.Prometheus.Port,
			Path:    app.Config.Monitor.Prometheus.Path,
		},
		CertPaths: CertPaths{
			CertFile: app.Config.Auth.CertFile,
			KeyFile:  app.Config.Auth.KeyFile,
			CAFile:   app.Config.Auth.CAFile,
		},
		CertRotationInterval: app.Config.Auth.CertRotation.Interval,
		TunAddr:              app.Config.Network.Address,
		ListenPort:           app.Config.Tunnel.ListenPort,
	}
	return ic, nil
}
