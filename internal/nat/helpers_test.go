package nat

import (
	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// testNATConfig returns an enabled forward-NAT config with one permissive rule.
func testNATConfig() config.NATConfig {
	return config.NATConfig{
		Enabled: true,
		Forward: config.NATForwardConfig{
			Enabled: true,
			Rules: []config.NATForwardRule{
				{
					Comment: "test allow",
					SrcCIDR: "10.77.0.0/24",
					DstCIDR: "192.168.10.0/24",
					Ports:   []int{80, 443},
				},
			},
		},
	}
}

func testLogger() *zap.Logger {
	return zap.NewNop()
}
