package nat

import (
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// ReloadRules atomically swaps the ACL rule set. Established flows keep
// their conntrack entries (existing translations are not torn down —
// that would break live connections mid-transfer); new flows are
// evaluated against the new rules immediately.
func (e *Engine) ReloadRules(newCfg *config.NATConfig) error {
	if newCfg == nil || !newCfg.Enabled {
		// Enable flips are structural and handled by the tunnel layer's
		// restart-required classification, not here.
		return errPacketTruncated
	}

	acl, err := CompileForwardACL(newCfg.Forward.Rules)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = newCfg
	e.acl = acl

	e.logger.Info("NAT rules hot-reloaded",
		zap.Int("forward_rules", len(newCfg.Forward.Rules)),
		zap.Int("reverse_listeners", len(newCfg.Reverse.Listeners)),
	)
	return nil
}

// StartGC runs the conntrack GC loop until stop is closed. It sweeps at
// one minute intervals; entries idle beyond maxIdle are removed and
// their SNAT ports returned to the pool.
func (e *Engine) StartGC(stop <-chan struct{}, maxIdle time.Duration) {
	if maxIdle <= 0 {
		maxIdle = 5 * time.Minute
	}
	ticker := time.NewTicker(time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if n := e.table.GC(time.Now(), maxIdle); n > 0 {
					e.logger.Debug("NAT conntrack GC swept flows",
						zap.Int("removed", n),
					)
				}
			}
		}
	}()
}
