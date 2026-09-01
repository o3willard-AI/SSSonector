package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// natBaseConfig returns a config that passes all non-NAT validation, so
// NAT-specific tests can vary only the nat section.
func natBaseConfig(mode string) *AppConfig {
	cfg := NewAppConfig(Type(mode))
	cfg.Config.Mode = mode
	cfg.Config.Logging.Level = "info"
	cfg.Config.Network.Interface = "tun0"
	cfg.Config.Network.MTU = 1500
	cfg.Config.Network.Address = "10.77.0.1/24"
	if mode == ModeServer {
		cfg.Config.Tunnel.ListenAddress = "0.0.0.0"
		cfg.Config.Tunnel.ListenPort = 8443
	} else {
		cfg.Config.Tunnel.ServerAddress = "192.0.2.10"
		cfg.Config.Tunnel.ServerPort = 8443
	}
	cfg.Config.Security.TLS.MinVersion = "1.2"
	cfg.Config.Security.TLS.MaxVersion = "1.3"
	return cfg
}

func validForwardRule() NATForwardRule {
	return NATForwardRule{
		Comment: "client may reach server LAN web",
		SrcCIDR: "10.77.0.0/24",
		DstCIDR: "192.168.10.0/24",
		Ports:   []int{80, 443},
	}
}

func validListenerRule() NATListenerRule {
	return NATListenerRule{
		Comment:      "publish home web server",
		ListenPort:   8080,
		Dst:          "10.77.0.2:80",
		AllowedCIDRs: []string{"0.0.0.0/0"},
	}
}

// TestNATDisabledByDefault verifies the fail-closed zero value: a config
// that never mentions NAT must validate, and must carry NAT disabled.
func TestNATDisabledByDefault(t *testing.T) {
	cfg := natBaseConfig(ModeServer)
	if cfg.Config.NAT.Enabled {
		t.Fatal("zero-value NATConfig must be disabled")
	}
	if cfg.Config.NAT.Forward.Enabled || cfg.Config.NAT.Reverse.Enabled {
		t.Fatal("zero-value NAT sub-sections must be disabled")
	}
	v := NewValidator()
	if err := v.Validate(cfg); err != nil {
		t.Fatalf("default config with absent NAT section must validate: %v", err)
	}
}

// TestNATDisabledSubsectionConflict verifies sub-sections cannot claim
// enabled while the master switch is off.
func TestNATDisabledSubsectionConflict(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*NATConfig)
		wantErr string
	}{
		{
			name:    "forward enabled without master",
			mutate:  func(n *NATConfig) { n.Forward.Enabled = true },
			wantErr: "nat.forward.enabled requires nat.enabled",
		},
		{
			name:    "reverse enabled without master",
			mutate:  func(n *NATConfig) { n.Reverse.Enabled = true },
			wantErr: "nat.reverse.enabled requires nat.enabled",
		},
		{
			name: "forward rules defined without switches",
			mutate: func(n *NATConfig) {
				n.Forward.Rules = []NATForwardRule{validForwardRule()}
			},
			wantErr: "nat.forward.rules defined but nat.forward.enabled is false",
		},
		{
			name: "reverse listeners defined without switches",
			mutate: func(n *NATConfig) {
				n.Reverse.Listeners = []NATListenerRule{validListenerRule()}
			},
			wantErr: "nat.reverse.listeners defined but nat.reverse.enabled is false",
		},
	}

	v := NewValidator()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := natBaseConfig(ModeServer)
			tc.mutate(&cfg.Config.NAT)
			err := v.Validate(cfg)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestNATForwardRuleValidation is table-driven over malformed rules.
func TestNATForwardRuleValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*NATForwardRule)
		wantErr string
	}{
		{
			name:    "missing src_cidr",
			mutate:  func(r *NATForwardRule) { r.SrcCIDR = "" },
			wantErr: "src_cidr is required",
		},
		{
			name:    "invalid src_cidr",
			mutate:  func(r *NATForwardRule) { r.SrcCIDR = "10.77.0.0/33" },
			wantErr: "invalid src_cidr",
		},
		{
			name:    "missing dst_cidr",
			mutate:  func(r *NATForwardRule) { r.DstCIDR = "" },
			wantErr: "dst_cidr is required",
		},
		{
			name:    "invalid dst_cidr",
			mutate:  func(r *NATForwardRule) { r.DstCIDR = "not-a-cidr" },
			wantErr: "invalid dst_cidr",
		},
		{
			name:    "dst inside tunnel subnet (loop hazard)",
			mutate:  func(r *NATForwardRule) { r.DstCIDR = "10.77.0.0/24" },
			wantErr: "loop hazard",
		},
		{
			name:    "dst inside tunnel subnet via wider CIDR",
			mutate:  func(r *NATForwardRule) { r.DstCIDR = "10.0.0.0/8" },
			wantErr: "loop hazard",
		},
		{
			name:    "port out of range high",
			mutate:  func(r *NATForwardRule) { r.Ports = []int{80, 65536} },
			wantErr: "invalid port 65536",
		},
		{
			name:    "port out of range zero",
			mutate:  func(r *NATForwardRule) { r.Ports = []int{0} },
			wantErr: "invalid port 0",
		},
		{
			name:    "ports empty fails closed at validation (rule permitted but matches nothing)",
			mutate:  func(r *NATForwardRule) { r.Ports = nil },
			wantErr: "",
		},
	}

	v := NewValidator()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := natBaseConfig(ModeServer)
			rule := validForwardRule()
			tc.mutate(&rule)
			cfg.Config.NAT.Enabled = true
			cfg.Config.NAT.Forward.Enabled = true
			cfg.Config.NAT.Forward.Rules = []NATForwardRule{rule}

			err := v.Validate(cfg)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected valid, got: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestNATForwardDuplicateRules rejects duplicate rules.
func TestNATForwardDuplicateRules(t *testing.T) {
	cfg := natBaseConfig(ModeServer)
	r := validForwardRule()
	cfg.Config.NAT.Enabled = true
	cfg.Config.NAT.Forward.Enabled = true
	cfg.Config.NAT.Forward.Rules = []NATForwardRule{r, r}

	v := NewValidator()
	err := v.Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "duplicate rule") {
		t.Fatalf("expected duplicate-rule error, got: %v", err)
	}
}

// TestNATReverseListenerValidation is table-driven over malformed listeners.
func TestNATReverseListenerValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*NATListenerRule)
		wantErr string
	}{
		{
			name:    "listen_port zero",
			mutate:  func(l *NATListenerRule) { l.ListenPort = 0 },
			wantErr: "invalid listen_port 0",
		},
		{
			name:    "listen_port out of range",
			mutate:  func(l *NATListenerRule) { l.ListenPort = 70000 },
			wantErr: "invalid listen_port 70000",
		},
		{
			name:    "missing dst",
			mutate:  func(l *NATListenerRule) { l.Dst = "" },
			wantErr: "dst is required",
		},
		{
			name:    "dst without port",
			mutate:  func(l *NATListenerRule) { l.Dst = "10.77.0.2" },
			wantErr: "invalid dst (want host:port)",
		},
		{
			name:    "dst with hostname instead of IP",
			mutate:  func(l *NATListenerRule) { l.Dst = "home.example.com:80" },
			wantErr: "not an IP",
		},
		{
			name:    "dst port out of range",
			mutate:  func(l *NATListenerRule) { l.Dst = "10.77.0.2:99999" },
			wantErr: "invalid dst port",
		},
		{
			name:    "empty allowed_cidrs fails closed",
			mutate:  func(l *NATListenerRule) { l.AllowedCIDRs = nil },
			wantErr: "allowed_cidrs is required",
		},
		{
			name:    "invalid allowed_cidrs entry",
			mutate:  func(l *NATListenerRule) { l.AllowedCIDRs = []string{"10.0.0.0/8", "bogus"} },
			wantErr: "invalid CIDR notation",
		},
	}

	v := NewValidator()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := natBaseConfig(ModeServer)
			l := validListenerRule()
			tc.mutate(&l)
			cfg.Config.NAT.Enabled = true
			cfg.Config.NAT.Reverse.Enabled = true
			cfg.Config.NAT.Reverse.Listeners = []NATListenerRule{l}

			err := v.Validate(cfg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestNATReverseDuplicateListenPorts rejects two listeners on one port.
func TestNATReverseDuplicateListenPorts(t *testing.T) {
	cfg := natBaseConfig(ModeServer)
	l1 := validListenerRule()
	l2 := validListenerRule()
	l2.Dst = "10.77.0.2:8080"
	cfg.Config.NAT.Enabled = true
	cfg.Config.NAT.Reverse.Enabled = true
	cfg.Config.NAT.Reverse.Listeners = []NATListenerRule{l1, l2}

	v := NewValidator()
	err := v.Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "duplicate listen_port") {
		t.Fatalf("expected duplicate listen_port error, got: %v", err)
	}
}

// TestNATFullyValidConfig verifies both paths validate together and on
// the client role too.
func TestNATFullyValidConfig(t *testing.T) {
	v := NewValidator()
	for _, mode := range []string{ModeServer, ModeClient} {
		cfg := natBaseConfig(mode)
		cfg.Config.NAT.Enabled = true
		cfg.Config.NAT.Forward.Enabled = true
		cfg.Config.NAT.Forward.Rules = []NATForwardRule{validForwardRule()}
		cfg.Config.NAT.Reverse.Enabled = true
		cfg.Config.NAT.Reverse.Listeners = []NATListenerRule{validListenerRule()}

		if err := v.Validate(cfg); err != nil {
			t.Fatalf("mode %s: expected valid NAT config, got: %v", mode, err)
		}
	}
}

// TestNATClientWithoutTunnelAddress verifies loop-hazard checking is
// skipped (not failed) when the tunnel address is unset.
func TestNATClientWithoutTunnelAddress(t *testing.T) {
	cfg := natBaseConfig(ModeClient)
	cfg.Config.Network.Address = ""
	cfg.Config.NAT.Enabled = true
	cfg.Config.NAT.Forward.Enabled = true
	cfg.Config.NAT.Forward.Rules = []NATForwardRule{validForwardRule()}

	v := NewValidator()
	if err := v.Validate(cfg); err != nil {
		t.Fatalf("expected valid config without tunnel address, got: %v", err)
	}
}

// TestNATRoundTripYAML ensures the nat section serializes and deserializes
// with the documented field names.
func TestNATRoundTripYAML(t *testing.T) {
	cfg := natBaseConfig(ModeServer)
	cfg.Config.NAT.Enabled = true
	cfg.Config.NAT.Forward.Enabled = true
	cfg.Config.NAT.Forward.Rules = []NATForwardRule{validForwardRule()}
	cfg.Config.NAT.Reverse.Enabled = true
	cfg.Config.NAT.Reverse.Listeners = []NATListenerRule{validListenerRule()}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	text := string(out)

	for _, want := range []string{"nat:", "enabled:", "forward:", "reverse:", "src_cidr:", "dst_cidr:", "listen_port:", "allowed_cidrs:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("marshaled config missing %q:\n%s", want, text)
		}
	}

	loaded, err := LoadConfigString(text, "yaml")
	if err != nil {
		t.Fatalf("round-trip load failed: %v", err)
	}
	if !loaded.Config.NAT.Enabled || !loaded.Config.NAT.Forward.Enabled || !loaded.Config.NAT.Reverse.Enabled {
		t.Fatal("round-trip lost NAT enabled flags")
	}
	if len(loaded.Config.NAT.Forward.Rules) != 1 || loaded.Config.NAT.Forward.Rules[0].SrcCIDR != "10.77.0.0/24" {
		t.Fatalf("round-trip lost forward rule: %+v", loaded.Config.NAT.Forward.Rules)
	}
	if len(loaded.Config.NAT.Reverse.Listeners) != 1 || loaded.Config.NAT.Reverse.Listeners[0].ListenPort != 8080 {
		t.Fatalf("round-trip lost listener: %+v", loaded.Config.NAT.Reverse.Listeners)
	}
}
