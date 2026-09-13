// Package view renders the TUI data panels as PURE functions: typed
// snapshot input in, deterministic styled string out. No package state, no
// time.Now() — callers pass "now" so golden output is deterministic.
//
// Rendering honors docs/tui.md §3.2 exactly: an errored source renders
// "unreachable" (never a stale/zero value), an absent source renders the
// disabled placeholder, and degenerate inputs (no instances, unreadable
// cert) have dedicated lines. Panels are plain text — terminal styling is
// applied by the WI 2.3 assembly, keeping these functions width-stable for
// golden tests.
package view

import (
	"fmt"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// RailInput is everything the INSTANCES rail shows.
type RailInput struct {
	Instances []collect.InstanceState
	// FocusIdx is the focused row (-1 = none).
	FocusIdx int
	// TUNAddresses maps instance name -> TUN address (from config
	// resolution; missing entries render "—").
	TUNAddresses map[string]string
	// PeerCounts maps instance name -> active connections (from metrics;
	// missing entries render "—").
	PeerCounts map[string]int
	// LastPeerChange maps instance name -> last state-change time (for
	// the age column; missing entries render "—").
	LastPeerChange map[string]time.Time
}

// RenderRail renders the INSTANCES rail (shared panel).
func RenderRail(in RailInput, now time.Time) string {
	if len(in.Instances) == 0 {
		return "INSTANCES  no sssonector@* units on this host\n"
	}
	var b strings.Builder
	b.WriteString("INSTANCES\n")
	for i, st := range in.Instances {
		marker := "  "
		if i == in.FocusIdx {
			marker = "▸ "
		}
		dot := healthDot(st)
		tun := "—"
		if in.TUNAddresses != nil {
			if v, ok := in.TUNAddresses[st.Name]; ok {
				tun = v
			}
		}
		peers := "—"
		if in.PeerCounts != nil {
			if v, ok := in.PeerCounts[st.Name]; ok {
				peers = fmt.Sprintf("%d", v)
			}
		}
		age := "—"
		if in.LastPeerChange != nil {
			if t, ok := in.LastPeerChange[st.Name]; ok && !t.IsZero() {
				age = humanAge(now.Sub(t))
			}
		}
		fmt.Fprintf(&b, "%s%-12s %s  tun %-16s peers %-4s up %-8s %s\n",
			marker, st.Name, strings.TrimSuffix(st.Unit, ".service"), tun, peers, age, dot)
	}
	return b.String()
}

// healthDot picks the rail health dot from systemd state.
func healthDot(st collect.InstanceState) string {
	switch {
	case st.ActiveState == "active" && st.SubState == "running":
		return "●"
	case st.ActiveState == "failed":
		return "✖"
	case st.ActiveState == "inactive":
		return "○"
	default:
		return "⚠"
	}
}

// humanAge formats a duration compactly (2h14m / 41m / 5m / 30s).
func humanAge(d time.Duration) string {
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// RenderDaemonHeader renders the top-right daemon line from the focused
// instance's systemd state.
func RenderDaemonHeader(st collect.InstanceState) string {
	if st.ActiveState == "" {
		return "daemon: unknown"
	}
	switch {
	case st.Running():
		return fmt.Sprintf("daemon: RUNNING (pid %d)", st.MainPID)
	case st.ActiveState == "failed":
		return fmt.Sprintf("daemon: FAILED (%s)", st.SubState)
	case st.ActiveState == "inactive":
		return "daemon: STOPPED"
	default:
		return fmt.Sprintf("daemon: %s/%s (pid %d)", st.ActiveState, st.SubState, st.MainPID)
	}
}

// tunAddrOf extracts a TUN-ish address line for the TUNNEL panel. The
// exposition has no TUN address series yet, so the panel renders it from
// the caller-supplied value ("—" when unknown) — pure, no fabrication.
func tunAddrOf(tunAddr string) string {
	if tunAddr == "" {
		return "—"
	}
	return tunAddr
}

// RenderTunnel renders the TUNNEL panel from healthz + connections.
// peers comes from the metrics snapshot's connections_active (source
// statuses preserved); tunAddr is the config-resolved TUN address.
func RenderTunnel(healthz collect.SourceOf[collect.Healthz], metrics collect.SourceOf[collect.Snapshot], tunAddr string, stateSince time.Time, now time.Time) string {
	var b strings.Builder
	b.WriteString("TUNNEL\n")
	switch healthz.Status() {
	case collect.StatusOK:
		h, _ := healthz.Get()
		fmt.Fprintf(&b, "  state    %s\n", tunnelStateGlyph(h.TunnelState))
		peers := "—"
		if metrics.Status() == collect.StatusOK {
			m, _ := metrics.Get()
			if v, ok := m.Connections.Active.Get(); ok {
				peers = fmt.Sprintf("%d", v)
			}
		}
		fmt.Fprintf(&b, "  peers    %s\n", peers)
		fmt.Fprintf(&b, "  tun      %s\n", tunAddrOf(tunAddr))
		if !stateSince.IsZero() {
			fmt.Fprintf(&b, "  since    %s\n", humanAge(now.Sub(stateSince)))
		}
	case collect.StatusError:
		fmt.Fprintf(&b, "  daemon unreachable (%v)\n", healthz.Err())
	default:
		b.WriteString("  — (healthz absent)\n")
	}
	return b.String()
}

// tunnelStateGlyph maps tunnel_state to its display glyph.
func tunnelStateGlyph(state string) string {
	switch state {
	case "up":
		return "● up"
	case "listening":
		return "● listening"
	case "connecting":
		return "◌ connecting"
	case "down":
		return "✖ down"
	default:
		return state
	}
}

// RenderNAT renders the NAT panel from the metrics snapshot's NAT fields.
func RenderNAT(metrics collect.SourceOf[collect.Snapshot]) string {
	var b strings.Builder
	b.WriteString("NAT\n")
	switch metrics.Status() {
	case collect.StatusOK:
		m, _ := metrics.Get()
		field := func(label string, o collect.OptInt) string {
			if v, ok := o.Get(); ok {
				return fmt.Sprintf("%d", v)
			}
			return "—"
		}
		fmt.Fprintf(&b, "  fwd      %s\n", field("fwd", m.NAT.ForwardedPackets))
		fmt.Fprintf(&b, "  ret      %s\n", field("ret", m.NAT.ReturnPackets))
		dropped := field("dropped", m.NAT.DroppedPackets)
		acl := field("acl", m.NAT.ACLDenies)
		if dropped != "—" {
			fmt.Fprintf(&b, "  dropped  %s (ACL denies %s)\n", dropped, acl)
		} else {
			fmt.Fprintf(&b, "  dropped  —\n")
		}
		fmt.Fprintf(&b, "  flows    %s\n", field("flows", m.NAT.ActiveFlows))
		fmt.Fprintf(&b, "  accepts  %s\n", field("accepts", m.NAT.ListenerAccepts))
	case collect.StatusError:
		fmt.Fprintf(&b, "  daemon unreachable (%v)\n", metrics.Err())
	default:
		b.WriteString("  — (prometheus disabled in config)\n")
	}
	return b.String()
}

// RenderRate renders the RATE LIMITER panel from throttle metrics.
func RenderRate(metrics collect.SourceOf[collect.Snapshot]) string {
	var b strings.Builder
	b.WriteString("RATE LIMITER\n")
	switch metrics.Status() {
	case collect.StatusOK:
		m, _ := metrics.Get()
		i := func(o collect.OptInt) string {
			if v, ok := o.Get(); ok {
				return fmt.Sprintf("%d", v)
			}
			return "—"
		}
		f := func(o collect.OptFloat64) string {
			if v, ok := o.Get(); ok {
				return fmt.Sprintf("%.1f MB/s", v/(1024*1024))
			}
			return "—"
		}
		fmt.Fprintf(&b, "  hits in   %s\n", i(m.Throttle.HitsIn))
		fmt.Fprintf(&b, "  hits out  %s\n", i(m.Throttle.HitsOut))
		fmt.Fprintf(&b, "  rate      %s\n", f(m.Throttle.EffectiveRate))
		fmt.Fprintf(&b, "  burst     %s\n", f(m.Throttle.BurstBytes))
	case collect.StatusError:
		fmt.Fprintf(&b, "  daemon unreachable (%v)\n", metrics.Err())
	default:
		b.WriteString("  — (prometheus disabled in config)\n")
	}
	return b.String()
}

// RenderCert renders the shared CERTIFICATE panel from CertInfo.
func RenderCert(cert collect.SourceOf[collect.CertInfo], now time.Time) string {
	var b strings.Builder
	b.WriteString("CERTIFICATE\n")
	switch cert.Status() {
	case collect.StatusOK:
		c, _ := cert.Get()
		days := int(c.NotAfter.Sub(now) / (24 * time.Hour))
		flag := ""
		if c.NeedsRotation {
			flag = "  ⚠ rotation due"
		}
		fmt.Fprintf(&b, "  issuer    %s\n", c.Issuer)
		fmt.Fprintf(&b, "  expires   %s (%dd)%s\n", c.NotAfter.Format("2006-01-02"), days, flag)
	case collect.StatusError:
		fmt.Fprintf(&b, "  cert unreadable: %v\n", cert.Err())
	default:
		b.WriteString("  — (cert absent)\n")
	}
	return b.String()
}

// RenderLog renders the shared LOG panel, tagged per instance, newest
// last. More than n entries are trimmed from the top (oldest dropped).
func RenderLog(entries []collect.LogEntry, n int) string {
	var b strings.Builder
	b.WriteString("LOG\n")
	if len(entries) == 0 {
		b.WriteString("  (none this read)\n")
		return b.String()
	}
	if n <= 0 {
		n = 12
	}
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	for _, e := range entries {
		if e.Timestamp.IsZero() {
			fmt.Fprintf(&b, "  [%s] %s\n", e.Instance, e.Raw)
		} else {
			fmt.Fprintf(&b, "  %s [%s] %s\n", e.Timestamp.Format("15:04:05"), e.Instance, e.Message)
		}
	}
	return b.String()
}
