package collect

import (
	"math"
	"strings"
	"testing"
)

// buildSnapshot parses fixture text and builds a Snapshot in one step.
func buildSnapshot(t *testing.T, fixture string) (Snapshot, error) {
	t.Helper()
	fams, err := ParseExposition(fixture)
	if err != nil {
		t.Fatalf("ParseExposition: %v", err)
	}
	return NewSnapshot(fams)
}

func assertInt(t *testing.T, name string, o OptInt, want int64, wantPresent bool) {
	t.Helper()
	v, ok := o.Get()
	if ok != wantPresent {
		t.Errorf("%s: present=%v, want %v", name, ok, wantPresent)
		return
	}
	if ok && v != want {
		t.Errorf("%s: got %d, want %d", name, v, want)
	}
}

func assertFloat(t *testing.T, name string, o OptFloat64, want float64, wantPresent bool) {
	t.Helper()
	v, ok := o.Get()
	if ok != wantPresent {
		t.Errorf("%s: present=%v, want %v", name, ok, wantPresent)
		return
	}
	if ok && v != want {
		t.Errorf("%s: got %v, want %v", name, v, want)
	}
}

// dropLine removes every line containing substr (used to build absence
// fixtures from the full live fixture).
func dropLine(fixture, substr string) string {
	var kept []string
	for _, line := range strings.Split(fixture, "\n") {
		if !strings.Contains(line, substr) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func TestSnapshot_RoundTrip_LiveFixture(t *testing.T) {
	s, err := buildSnapshot(t, liveFixture)
	if err != nil {
		t.Fatalf("NewSnapshot: %v", err)
	}

	// Network
	assertInt(t, "bytes_in", s.Network.BytesIn, 123456789, true)
	assertInt(t, "bytes_out", s.Network.BytesOut, 987654321, true)
	assertInt(t, "packets_in", s.Network.PacketsIn, 5000, true)
	assertInt(t, "packets_out", s.Network.PacketsOut, 4998, true)
	assertFloat(t, "byte_rate", s.Network.ByteRate, 1048576.5, true)

	// Errors — genuinely zero must stay zero AND present.
	assertInt(t, "errors", s.Errors, 0, true)

	// Connections
	assertInt(t, "connections_active", s.Connections.Active, 3, true)
	assertInt(t, "connections_peak", s.Connections.Peak, 9, true)

	// Throttle
	assertInt(t, "throttle_in", s.Throttle.HitsIn, 812, true)
	assertInt(t, "throttle_out", s.Throttle.HitsOut, 634, true)
	assertFloat(t, "throttle_rate", s.Throttle.EffectiveRate, 52428800, true)
	assertFloat(t, "throttle_burst", s.Throttle.BurstBytes, 104857600, true)

	// NAT
	assertInt(t, "nat_forwarded", s.NAT.ForwardedPackets, 1204551, true)
	assertInt(t, "nat_return", s.NAT.ReturnPackets, 1190003, true)
	assertInt(t, "nat_dropped", s.NAT.DroppedPackets, 12, true)
	assertInt(t, "nat_flows", s.NAT.ActiveFlows, 47, true)
	assertInt(t, "nat_accepts", s.NAT.ListenerAccepts, 63, true)
	assertInt(t, "nat_acl", s.NAT.ACLDenies, 8, true)
}

func TestSnapshot_Absence(t *testing.T) {
	t.Run("entire throttle family dropped", func(t *testing.T) {
		fixture := dropLine(liveFixture, "sssonector_throttle_hits_total")
		s, err := buildSnapshot(t, fixture)
		if err != nil {
			t.Fatalf("NewSnapshot: %v", err)
		}
		if s.Throttle.HitsIn.Present() {
			t.Error("HitsIn must be absent when family is missing")
		}
		if s.Throttle.HitsOut.Present() {
			t.Error("HitsOut must be absent when family is missing")
		}
		// Unrelated fields stay present (zero != absent invariant).
		assertInt(t, "errors still present", s.Errors, 0, true)
		assertInt(t, "nat_dropped still present", s.NAT.DroppedPackets, 12, true)
	})

	t.Run("one NAT family dropped", func(t *testing.T) {
		fixture := dropLine(liveFixture, "sssonector_nat_return_packets_total")
		s, err := buildSnapshot(t, fixture)
		if err != nil {
			t.Fatalf("NewSnapshot: %v", err)
		}
		if s.NAT.ReturnPackets.Present() {
			t.Error("ReturnPackets must be absent when its family is missing")
		}
		assertInt(t, "nat_forwarded still present", s.NAT.ForwardedPackets, 1204551, true)
	})

	t.Run("one labeled direction dropped", func(t *testing.T) {
		fixture := dropLine(liveFixture, `direction="out"`)
		s, err := buildSnapshot(t, fixture)
		if err != nil {
			t.Fatalf("NewSnapshot: %v", err)
		}
		if s.Throttle.HitsOut.Present() {
			t.Error("HitsOut must be absent when its series is missing")
		}
		assertInt(t, "HitsIn still present", s.Throttle.HitsIn, 812, true)
	})

	t.Run("genuinely zero stays zero and present", func(t *testing.T) {
		s, err := buildSnapshot(t, liveFixture)
		if err != nil {
			t.Fatalf("NewSnapshot: %v", err)
		}
		v, ok := s.Errors.Get()
		if !ok {
			t.Error("errors_total=0 must be PRESENT (zero != absent)")
		}
		if ok && v != 0 {
			t.Errorf("errors: got %d, want 0", v)
		}
	})
}

func TestSnapshot_LabeledDirectionLookup(t *testing.T) {
	// Sanity: canonical keys used by NewSnapshot must match what the
	// parser produces for the throttle series (guards against string
	// concatenation sneaking back in).
	fams, err := ParseExposition(`sssonector_throttle_hits_total{direction="in"} 11
sssonector_throttle_hits_total{direction="out"} 22
`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	th := fams["sssonector_throttle_hits_total"]
	if _, ok := th.Series[directionInLabel]; !ok {
		t.Errorf("parser key %q != canonical in-key %q", keys(th.Series), directionInLabel)
	}
	if _, ok := th.Series[directionOutLabel]; !ok {
		t.Errorf("parser key %q != canonical out-key %q", keys(th.Series), directionOutLabel)
	}

	s, err := buildSnapshot(t, `sssonector_throttle_hits_total{direction="in"} 11
sssonector_throttle_hits_total{direction="out"} 22
`)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	assertInt(t, "in", s.Throttle.HitsIn, 11, true)
	assertInt(t, "out", s.Throttle.HitsOut, 22, true)
}

func TestSnapshot_UnknownAndExtraMetricsIgnored(t *testing.T) {
	s, err := buildSnapshot(t, liveFixture+`
totally_unrelated_metric 1
another_one{job="x"} 2
`)
	if err != nil {
		t.Fatalf("NewSnapshot: %v", err)
	}
	assertInt(t, "bytes_in", s.Network.BytesIn, 123456789, true)
}

func TestSnapshot_NonFiniteAndNonIntegralIntFields(t *testing.T) {
	t.Run("NaN in int field is error", func(t *testing.T) {
		_, err := buildSnapshot(t, "sssonector_bytes_in_total NaN\n")
		if err == nil || !strings.Contains(err.Error(), "non-finite") {
			t.Errorf("want non-finite error, got %v", err)
		}
	})
	t.Run("non-integral in int field is error", func(t *testing.T) {
		_, err := buildSnapshot(t, "sssonector_bytes_in_total 5.5\n")
		if err == nil || !strings.Contains(err.Error(), "non-integral") {
			t.Errorf("want non-integral error, got %v", err)
		}
	})
	t.Run("NaN in float field passes through faithfully", func(t *testing.T) {
		s, err := buildSnapshot(t, "sssonector_byte_rate NaN\n")
		if err != nil {
			t.Fatalf("byte_rate NaN must be valid for a float field: %v", err)
		}
		v, ok := s.Network.ByteRate.Get()
		if !ok || !math.IsNaN(v) {
			t.Errorf("byte_rate: got %v ok=%v, want present NaN", v, ok)
		}
	})
}

// keys returns the series keys of a family for diagnostics.
func keys(m map[string]Sample) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
