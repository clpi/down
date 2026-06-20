package database

import "strings"

// FieldDef describes a database column.
type FieldDef struct {
	Type      string   `json:"type"`
	Options   []string `json:"options,omitempty"`
	Formula   string   `json:"formula,omitempty"`
	Relation  string   `json:"relation,omitempty"`
	Target    string   `json:"target,omitempty"`
	Aggregate string   `json:"aggregate,omitempty"`
}

// NormalizeSchema converts frontmatter schema maps into FieldDef values.
func NormalizeSchema(raw map[string]any) map[string]FieldDef {
	if raw == nil {
		return nil
	}
	if cols, ok := raw["columns"]; ok {
		if m, ok := cols.(map[string]any); ok {
			raw = m
		}
	}
	out := make(map[string]FieldDef, len(raw))
	for key, def := range raw {
		switch t := def.(type) {
		case string:
			out[key] = FieldDef{Type: t}
		case map[string]any:
			fd := FieldDef{Type: "text"}
			if v, ok := t["type"]; ok {
				fd.Type = asString(v)
			}
			if v, ok := t["formula"]; ok {
				fd.Formula = asString(v)
			}
			if v, ok := t["expression"]; ok && fd.Formula == "" {
				fd.Formula = asString(v)
			}
			if v, ok := t["relation"]; ok {
				fd.Relation = asString(v)
			}
			if v, ok := t["rollup_relation"]; ok && fd.Relation == "" {
				fd.Relation = asString(v)
			}
			if v, ok := t["target"]; ok {
				fd.Target = asString(v)
			}
			if v, ok := t["rollup_target"]; ok && fd.Target == "" {
				fd.Target = asString(v)
			}
			if v, ok := t["aggregate"]; ok {
				fd.Aggregate = asString(v)
			}
			if opts, ok := t["options"]; ok {
				fd.Options = stringSlice(opts)
			}
			if fd.Type == "" {
				fd.Type = "text"
			}
			out[key] = fd
		}
	}
	return out
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, asString(item))
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

// DefaultValue returns an empty cell value for a field type.
func DefaultValue(fd FieldDef) string {
	switch fd.Type {
	case "checkbox":
		return "No"
	case "number":
		return "0"
	default:
		return ""
	}
}

// SchemaFromHeaders builds a schema from table headers when no YAML schema exists.
func SchemaFromHeaders(headers []string) map[string]FieldDef {
	out := make(map[string]FieldDef, len(headers))
	for _, h := range headers {
		key := strings.TrimSpace(h)
		if key == "" {
			continue
		}
		kind := "text"
		switch strings.ToLower(key) {
		case "title", "name":
			kind = "title"
		case "status":
			kind = "status"
		case "date", "due", "due_date", "created", "updated":
			kind = "date"
		case "tags":
			kind = "multi_select"
		case "priority":
			kind = "select"
		case "done", "complete", "completed":
			kind = "checkbox"
		case "url", "link":
			kind = "url"
		case "email":
			kind = "email"
		}
		out[key] = FieldDef{Type: kind}
	}
	return out
}
