package collect

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	cfg "github.com/o3willard-AI/SSSonector/internal/config"
)

// ConfigOrigin marks where an effective-config value came from.
type ConfigOrigin int

const (
	// OriginUser: explicitly present in the instance file (even when the
	// value equals the default).
	OriginUser ConfigOrigin = iota
	// OriginDefault: filled in by the loader's schema defaults.
	OriginDefault
)

// String renders the origin tag.
func (o ConfigOrigin) String() string {
	if o == OriginUser {
		return "user"
	}
	return "default"
}

// ConfigLine is one rendered effective-config field.
type ConfigLine struct {
	// Path is the dotted config path (e.g. "monitor.prometheus.port",
	// "auth.cert_file").
	Path string
	// Value is the effective (merged) value, rendered as text.
	Value string
	// Origin marks user-set vs default-filled.
	Origin ConfigOrigin
}

// EffectiveConfigDump renders the effective (schema-defaults-merged)
// config for an instance as deterministic lines with per-line origin.
//
// Values come from the daemon's own loader (cfg.LoadConfigFile — the merge
// is never re-implemented). Origin is determined by intersecting the raw
// instance file's YAML keys with the rendered paths: a field explicitly
// present in the file is user-set EVEN IF its value equals the default.
func EffectiveConfigDump(paths ConfigPaths, name string) ([]ConfigLine, error) {
	file, err := resolveConfigFile(paths, name)
	if err != nil {
		return nil, err
	}

	// Merged values: the daemon's own loader (source of truth).
	app, err := cfg.LoadConfigFile(file)
	if err != nil {
		return nil, err // loader error surfaced verbatim
	}

	// Explicit-in-file key set (leaf paths, dotted).
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", file, err)
	}
	explicit, err := explicitYAMLKeys(raw)
	if err != nil {
		return nil, fmt.Errorf("config: %s: %w", file, err)
	}
	// The file wraps fields under "config:"; the dump paths are relative
	// to that wrapper. Also map legacy top-level "mode" (client.yaml).
	normalized := make(map[string]bool, len(explicit))
	for k := range explicit {
		if strings.HasPrefix(k, "config.") {
			normalized[strings.TrimPrefix(k, "config.")] = true
		} else {
			normalized[k] = true
		}
	}
	explicit = normalized

	lines := flattenAppConfig("", reflect.ValueOf(app.Config).Elem())
	sort.Slice(lines, func(i, j int) bool { return lines[i].Path < lines[j].Path })
	for i := range lines {
		if explicit[lines[i].Path] {
			lines[i].Origin = OriginUser
		} else {
			lines[i].Origin = OriginDefault
		}
	}
	return lines, nil
}

// resolveConfigFile locates the instance's config file (instances/<n>/
// config.yaml, legacy <root>/config.yaml).
func resolveConfigFile(paths ConfigPaths, name string) (string, error) {
	instanceFile := paths.ConfigRoot + "/instances/" + name + "/config.yaml"
	legacyFile := paths.ConfigRoot + "/config.yaml"
	if _, err := os.Stat(instanceFile); err == nil {
		return instanceFile, nil
	}
	if _, err := os.Stat(legacyFile); err == nil {
		return legacyFile, nil
	}
	return "", fmt.Errorf("config: instance %q: no config file found (tried %s and %s)",
		name, instanceFile, legacyFile)
}

// explicitYAMLKeys parses raw YAML and returns every leaf key path
// (dotted, mapping keys only; sequences are not descended into).
func explicitYAMLKeys(raw []byte) (map[string]bool, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return map[string]bool{}, nil
	}
	keys := map[string]bool{}
	collectLeafKeys(root.Content[0], "", keys)
	return keys, nil
}

// collectLeafKeys walks a mapping node accumulating dotted leaf paths.
// A leaf is a key whose value is a scalar (non-mapping, non-sequence) —
// but a mapping that contains a sequence leaf still records the sequence
// key itself.
func collectLeafKeys(node *yaml.Node, prefix string, out map[string]bool) {
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		switch val.Kind {
		case yaml.MappingNode:
			collectLeafKeys(val, path, out)
		default:
			// scalar or sequence: the key itself is the leaf
			out[path] = true
		}
	}
}

// flattenAppConfig renders the merged AppConfig into lines via reflection
// (struct fields in declaration order; yaml tags give the path segments).
// Durations render as their original text form; nested structs recurse;
// pointers are dereferenced.
func flattenAppConfig(prefix string, v reflect.Value) []ConfigLine {
	var out []ConfigLine
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "-" || tag == "" {
			tag = strings.ToLower(field.Name)
		} else {
			tag = strings.Split(tag, ",")[0]
		}
		path := tag
		if prefix != "" {
			path = prefix + "." + tag
		}

		switch fv.Kind() {
		case reflect.Struct:
			// Skip non-config leaf structs (time.Time etc. render inline).
			if fv.Type().String() == "time.Time" {
				out = append(out, ConfigLine{Path: path, Value: fmt.Sprintf("%v", fv.Interface())})
				continue
			}
			out = append(out, flattenAppConfig(path, fv)...)
		case reflect.Ptr:
			if fv.IsNil() {
				out = append(out, ConfigLine{Path: path, Value: "<nil>"})
				continue
			}
			out = append(out, flattenAppConfig(path, fv.Elem())...)
		default:
			out = append(out, ConfigLine{Path: path, Value: renderScalar(fv)})
		}
	}
	return out
}

// renderScalar renders a leaf value as deterministic text.
func renderScalar(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// time.Duration renders in its text form (e.g. 10s).
		if v.Type().String() == "time.Duration" {
			return timeDurationString(v.Int())
		}
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Slice, reflect.Array:
		parts := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			parts = append(parts, renderScalar(v.Index(i)))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case reflect.Map:
		// Sorted map rendering for determinism.
		keys := v.MapKeys()
		sort.Slice(keys, func(a, b int) bool {
			return fmt.Sprintf("%v", keys[a].Interface()) < fmt.Sprintf("%v", keys[b].Interface())
		})
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%v=%v", k.Interface(), v.MapIndex(k).Interface()))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case reflect.Interface:
		if v.IsNil() {
			return "<nil>"
		}
		return fmt.Sprintf("%v", v.Interface())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// timeDurationString renders a duration like Go's time.Duration.String.
func timeDurationString(nanos int64) string {
	d := make([]string, 0, 3)
	h := nanos / int64(3600e9)
	m := (nanos % int64(3600e9)) / int64(60e9)
	s := (nanos % int64(60e9)) / 1e9
	ms := (nanos % 1e9) / 1e6
	if h > 0 {
		d = append(d, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		d = append(d, fmt.Sprintf("%dm", m))
	}
	if s > 0 || len(d) == 0 {
		if ms > 0 {
			d = append(d, fmt.Sprintf("%d.%03ds", s, ms))
		} else {
			d = append(d, fmt.Sprintf("%ds", s))
		}
	}
	return strings.Join(d, "")
}

// FormatConfigDump renders ConfigLines as deterministic plain text
// (origin-tagged; the bright/dim styling is applied in WI 3.4).
func FormatConfigDump(lines []ConfigLine) string {
	var b strings.Builder
	b.WriteString("effective config:\n")
	for _, l := range lines {
		fmt.Fprintf(&b, "  %-52s = %-20s [%s]\n", l.Path, l.Value, l.Origin)
	}
	return b.String()
}
