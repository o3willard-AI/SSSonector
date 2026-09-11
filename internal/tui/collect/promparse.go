package collect

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// MetricFamily is one parsed exposition metric name with its series.
//
// Absent metrics are simply not present in the returned map — the parser
// never fabricates zero/default values (faithful parsing only).
type MetricFamily struct {
	// Type is the declared type from "# TYPE <name> <type>": "counter",
	// "gauge", "untyped", "histogram", "summary", or "" when undeclared.
	Type string
	// Help is the text from "# HELP <name> <text>", or "" when absent.
	Help string
	// Series holds one entry per rendered sample line, keyed by label set.
	// For unlabeled metrics the key is "". Labels are canonical
	// name="value" pairs sorted by name (label sets are unique per family).
	Series map[string]Sample
}

// Sample is a single metric sample.
type Sample struct {
	// Labels maps label name to raw (unescaped) label value.
	Labels map[string]string
	// Value is the parsed numeric value. NaN/+Inf/-Inf are represented
	// with the corresponding float64 values.
	Value float64
}

// ParseExposition parses Prometheus text exposition format (version 0.0.4)
// and returns the parsed families keyed by metric name.
//
// It handles: comments, # HELP / # TYPE metadata, labeled and unlabeled
// sample lines, integer and float values, NaN / +Inf / -Inf, and backslash
// and quote escaping inside label values. Unknown metric names parse fine.
// A malformed line returns an error; the parser never panics on any input.
func ParseExposition(data string) (map[string]*MetricFamily, error) {
	families := make(map[string]*MetricFamily)

	for lineNo, raw := range strings.Split(data, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case strings.HasPrefix(trimmed, "#"):
			kind, rest, err := parseMetaLine(trimmed)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
			}
			switch kind {
			case "help":
				name, text := splitMetaValue(rest)
				f := family(families, name)
				f.Help = text
			case "type":
				name, typ := splitMetaValue(rest)
				switch typ {
				case "counter", "gauge", "untyped", "histogram", "summary":
					family(families, name).Type = typ
				default:
					return nil, fmt.Errorf("line %d: unknown metric type %q", lineNo+1, typ)
				}
			}
			continue
		}

		name, labels, value, err := parseSampleLine(trimmed)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
		}
		f := family(families, name)
		key := canonicalLabelKey(labels)
		if _, dup := f.Series[key]; dup {
			return nil, fmt.Errorf("line %d: duplicate series %q for metric %q", lineNo+1, key, name)
		}
		f.Series[key] = Sample{Labels: labels, Value: value}
	}
	return families, nil
}

// family returns (creating if needed) the named metric family.
func family(m map[string]*MetricFamily, name string) *MetricFamily {
	f, ok := m[name]
	if !ok {
		f = &MetricFamily{Series: make(map[string]Sample)}
		m[name] = f
	}
	return f
}

// parseMetaLine splits a "#" line into its kind (help/type/comment) and the
// remaining text. Returns kind="" (comment) for anything not HELP/TYPE.
func parseMetaLine(line string) (kind, rest string, err error) {
	body := strings.TrimPrefix(line, "#")
	if !strings.HasPrefix(body, " ") {
		return "", "", fmt.Errorf("malformed comment %q: '#' must be followed by a space or be a bare comment", line)
	}
	body = body[1:]
	switch {
	case strings.HasPrefix(body, "HELP "):
		return "help", strings.TrimPrefix(body, "HELP "), nil
	case strings.HasPrefix(body, "TYPE "):
		return "type", strings.TrimPrefix(body, "TYPE "), nil
	default:
		return "", "", nil // ordinary comment
	}
}

// splitMetaValue splits "<name> <rest>" at the first space.
func splitMetaValue(rest string) (name, value string) {
	name = rest
	value = ""
	if i := strings.IndexByte(rest, ' '); i >= 0 {
		name, value = rest[:i], rest[i+1:]
	}
	return name, value
}

// parseSampleLine parses "<name>{k="v",...} <value>" or "<name> <value>".
func parseSampleLine(line string) (name string, labels map[string]string, value float64, err error) {
	// Find the value token: the last space-separated token that is not
	// inside braces. Metric names and label sets contain no spaces, so the
	// separator is the first space outside braces.
	braceDepth := 0
	split := -1
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\\' && inQuote:
			i++ // skip escaped char
		case c == '"':
			inQuote = !inQuote
		case inQuote:
			// inside quoted label value
		case c == '{':
			braceDepth++
		case c == '}':
			braceDepth--
			if braceDepth < 0 {
				return "", nil, 0, fmt.Errorf("unbalanced '}' in %q", line)
			}
		case c == ' ' && braceDepth == 0:
			split = i
			goto found
		}
	}
	return "", nil, 0, fmt.Errorf("missing value in sample line %q", line)
found:
	namePart := strings.TrimSpace(line[:split])
	valuePart := strings.TrimSpace(line[split+1:])
	if valuePart == "" {
		return "", nil, 0, fmt.Errorf("empty value in sample line %q", line)
	}

	if strings.HasSuffix(namePart, "}") {
		open := strings.IndexByte(namePart, '{')
		if open < 0 {
			return "", nil, 0, fmt.Errorf("unbalanced '}' in %q", namePart)
		}
		name = namePart[:open]
		if name == "" {
			return "", nil, 0, fmt.Errorf("empty metric name in %q", line)
		}
		labels, err = parseLabels(namePart[open+1 : len(namePart)-1])
		if err != nil {
			return "", nil, 0, err
		}
	} else {
		name = namePart
		if name == "" {
			return "", nil, 0, fmt.Errorf("empty metric name in %q", line)
		}
	}

	value, err = parseValue(valuePart)
	if err != nil {
		return "", nil, 0, err
	}
	return name, labels, value, nil
}

// parseLabels parses the inside of a label block: k="v",k2="v2" (may be empty).
func parseLabels(s string) (map[string]string, error) {
	labels := make(map[string]string)
	if strings.TrimSpace(s) == "" {
		return labels, nil
	}
	for _, part := range splitLabelPairs(s) {
		eq := strings.IndexByte(part, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("malformed label %q (want name=\"value\")", part)
		}
		k := strings.TrimSpace(part[:eq])
		v := strings.TrimSpace(part[eq+1:])
		if !isValidLabelName(k) {
			return nil, fmt.Errorf("invalid label name %q", k)
		}
		if len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"' {
			return nil, fmt.Errorf("label value for %q must be quoted: %q", k, v)
		}
		unescaped, err := unescapeLabelValue(v[1 : len(v)-1])
		if err != nil {
			return nil, fmt.Errorf("label %q: %w", k, err)
		}
		if _, dup := labels[k]; dup {
			return nil, fmt.Errorf("duplicate label name %q", k)
		}
		labels[k] = unescaped
	}
	return labels, nil
}

// splitLabelPairs splits on commas that are outside quoted values.
func splitLabelPairs(s string) []string {
	var parts []string
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++ // skip escaped char
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// unescapeLabelValue handles \\, \", and \n per the exposition format.
func unescapeLabelValue(v string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(v) {
			return "", fmt.Errorf("dangling escape in label value %q", v)
		}
		switch v[i] {
		case '\\':
			b.WriteByte('\\')
		case '"':
			b.WriteByte('"')
		case 'n':
			b.WriteByte('\n')
		default:
			return "", fmt.Errorf("invalid escape \\%c in label value", v[i])
		}
	}
	return b.String(), nil
}

func isValidLabelName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || (i > 0 && c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// parseValue accepts integers, floats, NaN, +Inf, -Inf (and bare Inf,
// case-insensitively per the format spec).
func parseValue(s string) (float64, error) {
	if s == "NaN" || s == "nan" {
		return math.NaN(), nil
	}
	switch s {
	case "+Inf", "Inf", "inf", "+inf":
		return math.Inf(1), nil
	case "-Inf", "-inf":
		return math.Inf(-1), nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", s)
	}
	return v, nil
}

// canonicalLabelKey builds the series key: k="v" pairs sorted by name.
func canonicalLabelKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(escapeKey(labels[k]))
		b.WriteByte('"')
	}
	return b.String()
}

// escapeKey escapes a label value for use inside the canonical key.
func escapeKey(v string) string {
	r := strings.ReplaceAll(v, `\`, `\\`)
	r = strings.ReplaceAll(r, `"`, `\"`)
	return strings.ReplaceAll(r, "\n", `\n`)
}
