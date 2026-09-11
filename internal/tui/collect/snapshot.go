package collect

import (
	"fmt"
	"math"
)

// Opt is a value-or-absent wrapper. Present distinguishes "the metric was
// reported" from "the metric was not reported": an absent Opt must never be
// rendered as a fabricated zero (anti-mock invariant, docs/tui.md §3.2).
type Opt[T any] struct {
	value   T
	present bool
}

// Some wraps v as a present value.
func Some[T any](v T) Opt[T] { return Opt[T]{value: v, present: true} }

// Absent returns the absent state for this type.
func Absent[T any]() Opt[T] { return Opt[T]{} }

// Present reports whether the value was actually reported.
func (o Opt[T]) Present() bool { return o.present }

// Get returns the value and whether it is present. When absent, value is
// the zero value of T and ok is false.
func (o Opt[T]) Get() (value T, ok bool) { return o.value, o.present }

// MustGet returns the value, panicking if absent. For render paths that
// have already checked Present.
func (o Opt[T]) MustGet() T {
	if !o.present {
		panic("collect: MustGet on absent Opt")
	}
	return o.value
}

// Snapshot is the typed view of one daemon instance's /metrics output.
// Every field is Opt: absent means the metric was missing from the
// exposition output, never a fabricated zero.
type Snapshot struct {
	Network     NetworkSnapshot
	Errors      OptInt
	Connections ConnectionsSnapshot
	Throttle    ThrottleSnapshot
	NAT         NATSnapshot
}

// NetworkSnapshot holds the tunnel byte/packet counters and rate.
type NetworkSnapshot struct {
	BytesIn    OptInt
	BytesOut   OptInt
	PacketsIn  OptInt
	PacketsOut OptInt
	ByteRate   OptFloat64
}

// ConnectionsSnapshot holds tunnel connection gauges.
type ConnectionsSnapshot struct {
	Active OptInt
	Peak   OptInt
}

// ThrottleSnapshot holds rate-limiter counters and pacing values.
type ThrottleSnapshot struct {
	HitsIn        OptInt
	HitsOut       OptInt
	EffectiveRate OptFloat64
	BurstBytes    OptFloat64
}

// NATSnapshot holds the six NAT/PAT counters.
type NATSnapshot struct {
	ForwardedPackets OptInt
	ReturnPackets    OptInt
	DroppedPackets   OptInt
	ActiveFlows      OptInt
	ListenerAccepts  OptInt
	ACLDenies        OptInt
}

// OptInt is Opt for int64 metrics.
type OptInt = Opt[int64]

// OptFloat64 is Opt for float metrics.
type OptFloat64 = Opt[float64]

// directionInLabel is the canonical series key for the inbound direction.
var directionInLabel = canonicalLabelKey(map[string]string{"direction": "in"})

// directionOutLabel is the canonical series key for the outbound direction.
var directionOutLabel = canonicalLabelKey(map[string]string{"direction": "out"})

// NewSnapshot maps parsed exposition families to a typed Snapshot. Unknown
// families are ignored; missing series/metrics leave the corresponding
// fields absent. Special values: an integral float converts exactly; a
// non-integral float mapped into an int field is an error (the daemon
// renders those metrics with %d, so such output is malformed); NaN/±Inf in
// int fields is likewise an error. Float fields pass NaN/±Inf through
// faithfully.
func NewSnapshot(families map[string]*MetricFamily) (Snapshot, error) {
	var s Snapshot
	var errs []error
	intField := func(name string, dst *OptInt) {
		f, ok := families[name]
		if !ok {
			return // absent stays absent
		}
		sample, ok := f.Series[""]
		if !ok {
			errs = append(errs, fmt.Errorf("metric %q has no unlabeled series", name))
			return
		}
		v := sample.Value
		if math.IsNaN(v) || math.IsInf(v, 0) {
			errs = append(errs, fmt.Errorf("metric %q: non-finite value %v in int field", name, v))
			return
		}
		if v != math.Trunc(v) {
			errs = append(errs, fmt.Errorf("metric %q: non-integral value %v in int field", name, v))
			return
		}
		*dst = Some(int64(v))
	}
	floatField := func(name string, dst *OptFloat64) {
		f, ok := families[name]
		if !ok {
			return
		}
		sample, ok := f.Series[""]
		if !ok {
			errs = append(errs, fmt.Errorf("metric %q has no unlabeled series", name))
			return
		}
		*dst = Some(sample.Value)
	}

	intField("sssonector_bytes_in_total", &s.Network.BytesIn)
	intField("sssonector_bytes_out_total", &s.Network.BytesOut)
	intField("sssonector_packets_in_total", &s.Network.PacketsIn)
	intField("sssonector_packets_out_total", &s.Network.PacketsOut)
	floatField("sssonector_byte_rate", &s.Network.ByteRate)

	intField("sssonector_errors_total", &s.Errors)

	intField("sssonector_connections_active", &s.Connections.Active)
	intField("sssonector_connections_peak", &s.Connections.Peak)

	// Throttle hits are labeled by direction; look up via canonical keys.
	for _, m := range []struct {
		key string
		dst *OptInt
	}{
		{directionInLabel, &s.Throttle.HitsIn},
		{directionOutLabel, &s.Throttle.HitsOut},
	} {
		f, ok := families["sssonector_throttle_hits_total"]
		if !ok {
			continue
		}
		sample, ok := f.Series[m.key]
		if !ok {
			continue // that direction absent, the other may be present
		}
		v := sample.Value
		if math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) {
			errs = append(errs, fmt.Errorf("sssonector_throttle_hits_total{%s}: non-integral or non-finite value %v in int field", m.key, v))
			continue
		}
		*m.dst = Some(int64(v))
	}

	floatField("sssonector_throttle_effective_rate_bytes_per_second", &s.Throttle.EffectiveRate)
	floatField("sssonector_throttle_burst_bytes", &s.Throttle.BurstBytes)

	intField("sssonector_nat_forwarded_packets_total", &s.NAT.ForwardedPackets)
	intField("sssonector_nat_return_packets_total", &s.NAT.ReturnPackets)
	intField("sssonector_nat_dropped_packets_total", &s.NAT.DroppedPackets)
	intField("sssonector_nat_flows_active", &s.NAT.ActiveFlows)
	intField("sssonector_nat_listener_accepts_total", &s.NAT.ListenerAccepts)
	intField("sssonector_nat_acl_denied_total", &s.NAT.ACLDenies)

	if len(errs) > 0 {
		return s, fmt.Errorf("snapshot: %d malformed metric(s): %w", len(errs), errs[0])
	}
	return s, nil
}
