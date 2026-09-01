package tunnel

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// Reloadable is implemented by tunnel modes that support SIGHUP-driven
// configuration reload.
type Reloadable interface {
	ApplyConfig(newCfg *config.AppConfig) error
}

// UpdateCertificatePaths updates certificate paths to be absolute
func UpdateCertificatePaths(cfg *config.AppConfig, baseDir string) error {
	resolvePath := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(baseDir, path)
	}

	cfg.Config.Auth.CertFile = resolvePath(cfg.Config.Auth.CertFile)
	cfg.Config.Auth.KeyFile = resolvePath(cfg.Config.Auth.KeyFile)
	cfg.Config.Auth.CAFile = resolvePath(cfg.Config.Auth.CAFile)

	return nil
}

// validateReloadTarget rejects unusable reload payloads
func validateReloadTarget(newCfg *config.AppConfig) error {
	if newCfg == nil {
		return fmt.Errorf("config is required")
	}
	if newCfg.Config == nil {
		return fmt.Errorf("config.Config is required")
	}
	return nil
}

// applyRuntimeSettings pushes the hot-reloadable subset into live objects
// and warns about anything that needs a restart instead.
func applyRuntimeSettings(logger *zap.Logger, oldCfg, newCfg *config.AppConfig, transfer *Transfer, tlsManager *TLSManager) {
	if oldCfg != nil && oldCfg.Throttle != newCfg.Throttle {
		logger.Info("Applying reloaded throttle settings",
			zap.Bool("enabled", newCfg.Throttle.Enabled),
			zap.Float64("rate", newCfg.Throttle.Rate),
			zap.Int("burst", newCfg.Throttle.Burst),
		)
		if transfer != nil {
			transfer.UpdateConfig(newCfg)
		}
	}

	if oldCfg != nil && tlsManager != nil &&
		oldCfg.Config.Auth.CertRotation.Interval != newCfg.Config.Auth.CertRotation.Interval &&
		newCfg.Config.Auth.CertRotation.Interval > 0 {
		tlsManager.SetCertTunables(newCfg.Config.Auth.CertRotation.Interval, 0)
		logger.Info("Applied reloaded certificate check interval",
			zap.Duration("interval", newCfg.Config.Auth.CertRotation.Interval),
		)
	}

	logRestartRequiredChanges(logger, oldCfg, newCfg)
}

// logRestartRequiredChanges warns about configuration differences that only
// take effect after a service restart.
func logRestartRequiredChanges(logger *zap.Logger, oldCfg, newCfg *config.AppConfig) {
	if oldCfg == nil || oldCfg.Config == nil {
		return
	}
	oldC, newC := oldCfg.Config, newCfg.Config

	warnIfChanged := func(field string, changed bool) {
		if changed {
			logger.Warn("Configuration change requires restart",
				zap.String("field", field),
			)
		}
	}

	warnIfChanged("mode", oldC.Mode != newC.Mode)
	warnIfChanged("logging.file", oldC.Logging.File != newC.Logging.File)
	warnIfChanged("logging.format", oldC.Logging.Format != newC.Logging.Format)
	warnIfChanged("network.name", oldC.Network.Name != newC.Network.Name)
	warnIfChanged("network.address", oldC.Network.Address != newC.Network.Address)
	warnIfChanged("network.mtu", oldC.Network.MTU != newC.Network.MTU)
	warnIfChanged("tunnel.listen_address", oldC.Tunnel.ListenAddress != newC.Tunnel.ListenAddress)
	warnIfChanged("tunnel.listen_port", oldC.Tunnel.ListenPort != newC.Tunnel.ListenPort)
	warnIfChanged("tunnel.server_address", oldC.Tunnel.ServerAddress != newC.Tunnel.ServerAddress)
	warnIfChanged("tunnel.server_port", oldC.Tunnel.ServerPort != newC.Tunnel.ServerPort)
	warnIfChanged("facade.enabled", oldC.Facade.Enabled != newC.Facade.Enabled)
	warnIfChanged("monitor.type", oldC.Monitor.Type != newC.Monitor.Type)
	warnIfChanged("monitor.prometheus.port", oldC.Monitor.Prometheus.Port != newC.Monitor.Prometheus.Port)
	warnIfChanged("snmp.enabled", oldC.SNMP.Enabled != newC.SNMP.Enabled)
	warnIfChanged("snmp.port", oldC.SNMP.Port != newC.SNMP.Port)
	warnIfChanged("metrics.interval", oldC.Metrics.Interval != newC.Metrics.Interval)

	// NAT enable/disable is structural (engine lifecycle); rule and
	// listener changes are hot-applied by the engine's ReloadRules.
	warnIfChanged("nat.enabled", oldC.NAT.Enabled != newC.NAT.Enabled)
	warnIfChanged("nat.forward.enabled", oldC.NAT.Forward.Enabled != newC.NAT.Forward.Enabled)
	warnIfChanged("nat.reverse.enabled", oldC.NAT.Reverse.Enabled != newC.NAT.Reverse.Enabled)
}

// sampleThrottle accumulates per-transfer hit deltas into cumulative
// counters and returns them with the current pacing values. Safe on nil
// transfer: counters persist, pacing reports the last observed values.
func sampleThrottle(
	transfer *Transfer,
	hitsIn, hitsOut *atomic.Uint64,
	lastIn, lastOut *uint64,
	mu *sync.Mutex,
) (inHits, outHits uint64, rate, burst float64) {
	if transfer != nil {
		curIn, curOut, r, b := transfer.ThrottleStats()
		mu.Lock()
		deltaIn := curIn - *lastIn
		deltaOut := curOut - *lastOut
		*lastIn = curIn
		*lastOut = curOut
		mu.Unlock()
		hitsIn.Add(deltaIn)
		hitsOut.Add(deltaOut)
		return hitsIn.Load(), hitsOut.Load(), r, b
	}
	return hitsIn.Load(), hitsOut.Load(), 0, 0
}
