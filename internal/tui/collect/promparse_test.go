package collect

import (
	"math"
	"strings"
	"testing"
)

// liveFixture reproduces the exact /metrics output the daemon renders
// (internal/monitor handleMetrics): all 21 metrics including the two
// labeled throttle samples and the float-typed gauges.
const liveFixture = `# HELP sssonector_bytes_in_total Total bytes received from tunnel peers.
# TYPE sssonector_bytes_in_total counter
sssonector_bytes_in_total 123456789
# HELP sssonector_bytes_out_total Total bytes sent to tunnel peers.
# TYPE sssonector_bytes_out_total counter
sssonector_bytes_out_total 987654321
# HELP sssonector_packets_in_total Total packets received.
# TYPE sssonector_packets_in_total counter
sssonector_packets_in_total 5000
# HELP sssonector_packets_out_total Total packets sent.
# TYPE sssonector_packets_out_total counter
sssonector_packets_out_total 4998
# HELP sssonector_errors_total Total errors recorded.
# TYPE sssonector_errors_total counter
sssonector_errors_total 0
# HELP sssonector_byte_rate Current byte throughput per second.
# TYPE sssonector_byte_rate gauge
sssonector_byte_rate 1048576.500000
# HELP sssonector_connections_active Currently active tunnel connections.
# TYPE sssonector_connections_active gauge
sssonector_connections_active 3
# HELP sssonector_connections_peak Peak concurrent tunnel connections.
# TYPE sssonector_connections_peak gauge
sssonector_connections_peak 9
# HELP sssonector_throttle_hits_total Requests that had to wait for tokens.
# TYPE sssonector_throttle_hits_total counter
sssonector_throttle_hits_total{direction="in"} 812
sssonector_throttle_hits_total{direction="out"} 634
# HELP sssonector_throttle_effective_rate_bytes_per_second Effective paced rate including TCP overhead.
# TYPE sssonector_throttle_effective_rate_bytes_per_second gauge
sssonector_throttle_effective_rate_bytes_per_second 52428800.000000
# HELP sssonector_throttle_burst_bytes Burst allowance in bytes.
# TYPE sssonector_throttle_burst_bytes gauge
sssonector_throttle_burst_bytes 104857600.000000
# HELP sssonector_nat_forwarded_packets_total Forward-NAT packets translated tunnel to egress.
# TYPE sssonector_nat_forwarded_packets_total counter
sssonector_nat_forwarded_packets_total 1204551
# HELP sssonector_nat_return_packets_total Forward-NAT return packets reverse-translated egress to tunnel.
# TYPE sssonector_nat_return_packets_total counter
sssonector_nat_return_packets_total 1190003
# HELP sssonector_nat_dropped_packets_total NAT-dropped packets (ACL denies, malformed, no translation).
# TYPE sssonector_nat_dropped_packets_total counter
sssonector_nat_dropped_packets_total 12
# HELP sssonector_nat_flows_active Live NAT conntrack entries.
# TYPE sssonector_nat_flows_active gauge
sssonector_nat_flows_active 47
# HELP sssonector_nat_listener_accepts_total Reverse-PAT public connections accepted.
# TYPE sssonector_nat_listener_accepts_total counter
sssonector_nat_listener_accepts_total 63
# HELP sssonector_nat_acl_denied_total Reverse-PAT listener connections denied by ACL.
# TYPE sssonector_nat_acl_denied_total counter
sssonector_nat_acl_denied_total 8
# HELP sssonector_cpu_usage_percent Process CPU usage percentage.
# TYPE sssonector_cpu_usage_percent gauge
sssonector_cpu_usage_percent 3.750000
# HELP sssonector_memory_alloc_bytes Bytes allocated by the process heap.
# TYPE sssonector_memory_alloc_bytes gauge
sssonector_memory_alloc_bytes 8388608
# HELP sssonector_goroutines Current goroutine count.
# TYPE sssonector_goroutines gauge
sssonector_goroutines 42
# HELP sssonector_uptime_seconds Process uptime in seconds.
# TYPE sssonector_uptime_seconds gauge
sssonector_uptime_seconds 8040
`

func TestParseExposition_LiveFixture(t *testing.T) {
	fams, err := ParseExposition(liveFixture)
	if err != nil {
		t.Fatalf("parse live fixture: %v", err)
	}

	wantInt := map[string]float64{
		"sssonector_bytes_in_total":              123456789,
		"sssonector_bytes_out_total":             987654321,
		"sssonector_packets_in_total":            5000,
		"sssonector_packets_out_total":           4998,
		"sssonector_errors_total":                0,
		"sssonector_connections_active":          3,
		"sssonector_connections_peak":            9,
		"sssonector_memory_alloc_bytes":          8388608,
		"sssonector_goroutines":                  42,
		"sssonector_uptime_seconds":              8040,
		"sssonector_nat_forwarded_packets_total": 1204551,
		"sssonector_nat_return_packets_total":    1190003,
		"sssonector_nat_dropped_packets_total":   12,
		"sssonector_nat_flows_active":            47,
		"sssonector_nat_listener_accepts_total":  63,
		"sssonector_nat_acl_denied_total":        8,
	}
	for name, want := range wantInt {
		f, ok := fams[name]
		if !ok {
			t.Errorf("metric %q missing from parse result", name)
			continue
		}
		s, ok := f.Series[""]
		if !ok {
			t.Errorf("metric %q: expected one unlabeled series", name)
			continue
		}
		if s.Value != want {
			t.Errorf("metric %q: got %v, want %v", name, s.Value, want)
		}
	}

	// Float-typed gauges parse as floats.
	for name, want := range map[string]float64{
		"sssonector_byte_rate":                                1048576.5,
		"sssonector_throttle_effective_rate_bytes_per_second": 52428800,
		"sssonector_throttle_burst_bytes":                     104857600,
		"sssonector_cpu_usage_percent":                        3.75,
	} {
		if got := fams[name].Series[""].Value; got != want {
			t.Errorf("metric %q: got %v, want %v", name, got, want)
		}
	}

	// Labeled throttle hits: exactly two series keyed by direction.
	th := fams["sssonector_throttle_hits_total"]
	if th == nil || len(th.Series) != 2 {
		t.Fatalf("throttle_hits_total: want 2 labeled series, got %+v", th)
	}
	if got := th.Series[`direction="in"`]; got.Value != 812 {
		t.Errorf("throttle in: got %v, want 812", got.Value)
	}
	if got := th.Series[`direction="out"`]; got.Value != 634 {
		t.Errorf("throttle out: got %v, want 634", got.Value)
	}
	if th.Series[`direction="in"`].Labels["direction"] != "in" {
		t.Errorf("throttle in labels: %+v", th.Series[`direction="in"`].Labels)
	}

	// Types and help text survive.
	if fams["sssonector_bytes_in_total"].Type != "counter" {
		t.Errorf("bytes_in type: %q", fams["sssonector_bytes_in_total"].Type)
	}
	if fams["sssonector_byte_rate"].Type != "gauge" {
		t.Errorf("byte_rate type: %q", fams["sssonector_byte_rate"].Type)
	}
	if got := fams["sssonector_bytes_in_total"].Help; got != "Total bytes received from tunnel peers." {
		t.Errorf("bytes_in help: %q", got)
	}

	// Count families: 21 metric names in the fixture (throttle counted once).
	if len(fams) != 21 {
		t.Errorf("family count: got %d, want 21", len(fams))
	}
}

func TestParseExposition_LabelsAndEscapes(t *testing.T) {
	in := `# TYPE m counter
m{a="1",b="two words"} 3
n{k="quote\"inside"} 4
o{k="back\\slash"} 5
p{k="new\nline"} 6
q{} 7
`
	fams, err := ParseExposition(in)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := fams["m"]
	if m == nil || len(m.Series) != 1 {
		t.Fatalf("m: %+v", m)
	}
	s, ok := m.Series[`a="1",b="two words"`]
	if !ok {
		t.Fatal("m: labeled series key not found")
	}
	if s.Labels["b"] != "two words" {
		t.Errorf("b: %q", s.Labels["b"])
	}
	if fams["n"].Series[`k="quote\"inside"`].Labels["k"] != `quote"inside` {
		t.Errorf("quote escape: %+v", fams["n"].Series)
	}
	if fams["o"].Series[`k="back\\slash"`].Labels["k"] != `back\slash` {
		t.Errorf("backslash escape: %+v", fams["o"])
	}
	if fams["p"].Series[`k="new\nline"`].Labels["k"] != "new\nline" {
		t.Errorf("newline escape: %+v", fams["p"])
	}
	if _, ok := fams["q"].Series[""]; !ok {
		t.Fatalf("empty label block: series missing: %+v", fams["q"])
	}
	if fams["q"].Series[""].Value != 7 {
		t.Errorf("empty label block value: %v", fams["q"].Series[""].Value)
	}
}

func TestParseExposition_SpecialValues(t *testing.T) {
	in := `nan_metric NaN
pinf +Inf
ninf -Inf
neg -3.5
exp 1.5e3
`
	fams, err := ParseExposition(in)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !math.IsNaN(fams["nan_metric"].Series[""].Value) {
		t.Errorf("NaN: %v", fams["nan_metric"].Series[""].Value)
	}
	if !math.IsInf(fams["pinf"].Series[""].Value, 1) {
		t.Errorf("+Inf: %v", fams["pinf"].Series[""].Value)
	}
	if !math.IsInf(fams["ninf"].Series[""].Value, -1) {
		t.Errorf("-Inf: %v", fams["ninf"].Series[""].Value)
	}
	if got := fams["neg"].Series[""].Value; got != -3.5 {
		t.Errorf("neg: %v", got)
	}
	if got := fams["exp"].Series[""].Value; got != 1500 {
		t.Errorf("exp: %v", got)
	}
}

func TestParseExposition_MalformedReturnsError(t *testing.T) {
	cases := map[string]string{
		"no value":          "metric_without_value",
		"bad value":         "m abc",
		"trailing garbage":  "m 1 2",
		"unbalanced brace":  "m{k=\"v\" 1",
		"stray close brace": "m} 1",
		"unquoted label":    `m{k=v} 1`,
		"bad escape":        `m{k="a\zb"} 1`,
		"dup label":         `m{k="a",k="b"} 1`,
		"empty name":        `{k="v"} 1`,
		"bad type":          "# TYPE m weirdtype\nm 1",
		"bad label name":    `m{1bad="v"} 1`,
		"dangling escape":   `m{k="v\"} 1`,
		"dup series":        "m 1\nm 2",
		"bare hash":         "#HELP m x",
	}
	for name, in := range cases {
		in := in
		t.Run(name, func(t *testing.T) {
			got, err := ParseExposition(in) // must not panic
			if err == nil {
				t.Fatalf("input %q: expected error, got %+v", in, got)
			}
		})
	}
}

func TestParseExposition_NoPanicOnArbitraryInput(t *testing.T) {
	fuzzInputs := []string{
		"", "\n\n\n", "#", "# ", "####", `m{`, `m{}`, `m{"a"}`, `{"a"="b"} 1`,
		`m{k="`, `m{k="\`, `m{k=}`, "m NaN NaN", strings.Repeat("m 1\n", 100),
		"m -0 0", "m 0x10", "m 1_000", "\x00\xff", `m{k="unterminated} 1`,
	}
	for _, in := range fuzzInputs {
		_, _ = ParseExposition(in) // must not panic on any input
	}
}

func TestParseExposition_UnknownAndMissingMetrics(t *testing.T) {
	in := `# HELP totally_unknown Some other exporter's metric.
# TYPE totally_unknown counter
totally_unknown{job="x"} 99
sssonector_known 5
`
	fams, err := ParseExposition(in)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fams["totally_unknown"] == nil {
		t.Error("unknown metric should parse fine")
	}
	if _, ok := fams["sssonector_absent"]; ok {
		t.Error("absent metric must not be fabricated")
	}
	// Faithful parsing: a family with HELP/TYPE but no sample lines keeps
	// its metadata and has no series.
	in2 := "# HELP only_meta declared but never sampled\n# TYPE only_meta gauge\n"
	fams2, err := ParseExposition(in2)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f := fams2["only_meta"]; f == nil || f.Type != "gauge" || len(f.Series) != 0 {
		t.Errorf("metadata-only family: %+v", fams2["only_meta"])
	}
}

func TestParseExposition_CommentsAndBlankLines(t *testing.T) {
	in := "# a plain comment\n\n# another\nm 1\r\n\r\n"
	fams, err := ParseExposition(in)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(fams) != 1 || fams["m"].Series[""].Value != 1 {
		t.Errorf("comments/blank lines mishandled: %+v", fams)
	}
}
