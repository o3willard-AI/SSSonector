package nat

import (
	"net"
	"testing"

	"github.com/o3willard-AI/SSSonector/internal/config"
)

func mkRules(specs []struct {
	src   string
	dst   string
	ports []int
}) []config.NATForwardRule {
	rules := make([]config.NATForwardRule, 0, len(specs))
	for _, s := range specs {
		rules = append(rules, config.NATForwardRule{
			SrcCIDR: s.src,
			DstCIDR: s.dst,
			Ports:   s.ports,
		})
	}
	return rules
}

func TestACLEvaluate(t *testing.T) {
	rules := mkRules([]struct {
		src   string
		dst   string
		ports []int
	}{
		{src: "10.77.0.0/24", dst: "192.168.10.0/24", ports: []int{80, 443}},
		{src: "10.77.0.0/24", dst: "192.168.20.0/24", ports: []int{22}},
	})
	acl, err := CompileForwardACL(rules)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	tests := []struct {
		name    string
		src     string
		dst     string
		port    int
		want    ACLDecision
		wantIdx int
	}{
		{"web allowed", "10.77.0.2", "192.168.10.5", 80, ACLAllow, 0},
		{"https allowed", "10.77.0.99", "192.168.10.5", 443, ACLAllow, 0},
		{"wrong port denied", "10.77.0.2", "192.168.10.5", 22, ACLDeny, -1},
		{"wrong dst denied", "10.77.0.2", "192.168.30.5", 80, ACLDeny, -1},
		{"wrong src denied", "10.78.0.2", "192.168.10.5", 80, ACLDeny, -1},
		{"ssh allowed second rule", "10.77.0.2", "192.168.20.9", 22, ACLAllow, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, idx, _ := acl.Evaluate(net.ParseIP(tc.src), net.ParseIP(tc.dst), tc.port)
			if got != tc.want {
				t.Fatalf("decision: want %v got %v", tc.want, got)
			}
			if idx != tc.wantIdx {
				t.Fatalf("rule index: want %d got %d", tc.wantIdx, idx)
			}
		})
	}
}

func TestACLEmptyDeniesAll(t *testing.T) {
	acl, err := CompileForwardACL(nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	got, _, _ := acl.Evaluate(net.ParseIP("10.77.0.2"), net.ParseIP("192.168.10.5"), 80)
	if got != ACLDeny {
		t.Fatalf("empty ACL must deny, got %v", got)
	}
}

func TestACLRuleWithNoPortsNeverMatches(t *testing.T) {
	// Fail closed: a rule without ports permits nothing even if CIDRs match.
	acl, err := CompileForwardACL(mkRules([]struct {
		src   string
		dst   string
		ports []int
	}{
		{src: "10.77.0.0/24", dst: "192.168.10.0/24", ports: nil},
	}))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	got, _, _ := acl.Evaluate(net.ParseIP("10.77.0.2"), net.ParseIP("192.168.10.5"), 80)
	if got != ACLDeny {
		t.Fatalf("portless rule must never match, got %v", got)
	}
}

func TestACLCompileRejectsBadCIDR(t *testing.T) {
	bad := mkRules([]struct {
		src   string
		dst   string
		ports []int
	}{
		{src: "bogus", dst: "192.168.0.0/24", ports: []int{80}},
	})
	if _, err := CompileForwardACL(bad); err == nil {
		t.Fatal("expected compile error for bad src")
	}
	bad2 := mkRules([]struct {
		src   string
		dst   string
		ports []int
	}{
		{src: "10.0.0.0/8", dst: "bogus", ports: []int{80}},
	})
	if _, err := CompileForwardACL(bad2); err == nil {
		t.Fatal("expected compile error for bad dst")
	}
}

func TestListenerACL(t *testing.T) {
	acl, err := CompileListenerACL([]string{"203.0.113.0/24", "198.51.100.7/32"})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !acl.Evaluate(net.ParseIP("203.0.113.50")) {
		t.Fatal("in-subnet source must be allowed")
	}
	if !acl.Evaluate(net.ParseIP("198.51.100.7")) {
		t.Fatal("exact host source must be allowed")
	}
	if acl.Evaluate(net.ParseIP("203.0.114.1")) {
		t.Fatal("out-of-subnet source must be denied")
	}
}

func TestListenerACLEmptyDeniesAll(t *testing.T) {
	acl, err := CompileListenerACL(nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if acl.Evaluate(net.ParseIP("1.2.3.4")) {
		t.Fatal("empty allowlist must deny everything")
	}
}
