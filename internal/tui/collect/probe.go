package collect

import (
	"fmt"
	"strings"
	"time"
)

// FormatProbe renders a one-shot probe: the tick result plus per-instance
// log tails, as deterministic text. The second return is "fatal": true
// only when discovery itself failed (nothing could be probed at all).
//
// Per-source errors are NOT fatal — they render as explicit error/absent
// lines (never fabricated values, never stale ones — the value slots are
// empty for non-OK sources).
func FormatProbe(res TickResult, logs map[string][]LogEntry) (string, bool) {
	var b strings.Builder

	fmt.Fprintf(&b, "probe time: %s\n", time.Now().Format(time.RFC3339))

	if res.DiscoverErr != nil {
		fmt.Fprintf(&b, "discovery: ERROR: %v\n", res.DiscoverErr)
		return b.String(), true
	}
	if len(res.Instances) == 0 {
		b.WriteString("discovery: no sssonector instances found\n")
		return b.String(), false
	}

	// Deterministic order: sorted instance names.
	names := make([]string, 0, len(res.Instances))
	for name := range res.Instances {
		names = append(names, name)
	}
	sortStringsCopy(names)

	for _, name := range names {
		snap := res.Instances[name]
		fmt.Fprintf(&b, "\n== instance %s ==\n", name)
		if snap.Unit != "" {
			fmt.Fprintf(&b, "unit:        %s\n", snap.Unit)
		}
		fmt.Fprintf(&b, "systemd:     %s/%s pid=%d\n", orUnknown(snap.ActiveState), orUnknown(snap.SubState), snap.MainPID)

		// Config resolution state.
		if snap.ConfigErr != nil {
			fmt.Fprintf(&b, "config:      ERROR: %v\n", snap.ConfigErr)
		} else {
			if snap.PrometheusAddr != "" {
				fmt.Fprintf(&b, "endpoint:    %s\n", snap.PrometheusAddr)
			}
		}

		// healthz source.
		switch snap.Healthz.Status() {
		case StatusOK:
			h, _ := snap.Healthz.Get()
			fmt.Fprintf(&b, "healthz:     ok mode=%s tunnel_state=%s uptime=%ds\n", h.Mode, h.TunnelState, h.UptimeSeconds)
		case StatusError:
			fmt.Fprintf(&b, "healthz:     ERROR: %v\n", snap.Healthz.Err())
		default:
			b.WriteString("healthz:     absent\n")
		}

		// Metrics source: render the typed snapshot's fields.
		switch snap.Metrics.Status() {
		case StatusOK:
			m, _ := snap.Metrics.Get()
			fmt.Fprintf(&b, "metrics:     ok\n")
			writeSnapshotFields(&b, m)
		case StatusError:
			fmt.Fprintf(&b, "metrics:     ERROR: %v\n", snap.Metrics.Err())
		default:
			b.WriteString("metrics:     absent\n")
		}

		// Cert source.
		switch snap.Cert.Status() {
		case StatusOK:
			c, _ := snap.Cert.Get()
			fmt.Fprintf(&b, "cert:        ok issuer=%q days_remaining=%d needs_rotation=%t not_after=%s\n",
				c.Issuer, c.DaysRemaining, c.NeedsRotation, c.NotAfter.Format(time.RFC3339))
		case StatusError:
			fmt.Fprintf(&b, "cert:        ERROR: %v\n", snap.Cert.Err())
		default:
			b.WriteString("cert:        absent\n")
		}

		// Logs for this instance (up to 12 lines, journal order).
		if entries, ok := logs[name]; ok && len(entries) > 0 {
			b.WriteString("log:\n")
			for _, e := range entries {
				if e.Timestamp.IsZero() {
					fmt.Fprintf(&b, "  [%s] %s\n", e.Instance, e.Raw)
				} else {
					fmt.Fprintf(&b, "  %s [%s] %s\n", e.Timestamp.Format(time.RFC3339), e.Instance, e.Message)
				}
			}
		} else {
			b.WriteString("log:         (none this read)\n")
		}
	}
	return b.String(), false
}

// writeSnapshotFields renders the typed metrics snapshot fields.
func writeSnapshotFields(b *strings.Builder, m Snapshot) {
	field := func(label string, o OptInt) {
		if v, ok := o.Get(); ok {
			fmt.Fprintf(b, "  %-16s %d\n", label, v)
		}
	}
	ffield := func(label string, o OptFloat64) {
		if v, ok := o.Get(); ok {
			fmt.Fprintf(b, "  %-16s %g\n", label, v)
		}
	}
	field("bytes_in", m.Network.BytesIn)
	field("bytes_out", m.Network.BytesOut)
	field("packets_in", m.Network.PacketsIn)
	field("packets_out", m.Network.PacketsOut)
	ffield("byte_rate", m.Network.ByteRate)
	field("errors", m.Errors)
	field("connections_active", m.Connections.Active)
	field("connections_peak", m.Connections.Peak)
	field("throttle_hits_in", m.Throttle.HitsIn)
	field("throttle_hits_out", m.Throttle.HitsOut)
	ffield("throttle_rate", m.Throttle.EffectiveRate)
	ffield("throttle_burst", m.Throttle.BurstBytes)
	field("nat_forwarded", m.NAT.ForwardedPackets)
	field("nat_return", m.NAT.ReturnPackets)
	field("nat_dropped", m.NAT.DroppedPackets)
	field("nat_flows", m.NAT.ActiveFlows)
	field("nat_accepts", m.NAT.ListenerAccepts)
	field("nat_acl_denied", m.NAT.ACLDenies)
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

// sortStringsCopy sorts a copy of names (stdlib sort, exported nowhere).
func sortStringsCopy(names []string) {
	// simple insertion sort to avoid importing sort in this file — small N
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
}
