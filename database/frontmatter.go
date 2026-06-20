package database

import (
	"strconv"
	"strings"
)

// ParseFrontmatter extracts YAML frontmatter from markdown (--- delimited at file start).
func ParseFrontmatter(text string) (map[string]any, int, int, bool) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, 0, 0, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, 0, 0, false
	}
	body := strings.Join(lines[1:end], "\n")
	data := parseYAMLMap(body)
	return data, 0, end, true
}

func parseYAMLMap(body string) map[string]any {
	root := map[string]any{}
	lines := strings.Split(body, "\n")
	type frame struct {
		indent int
		value  map[string]any
	}
	stack := []frame{{indent: -1, value: root}}
	var listKey string
	var listIndent int
	var listItems []any

	flushList := func() {
		if listKey == "" {
			return
		}
		parent := stack[len(stack)-1].value
		parent[listKey] = listItems
		listKey = ""
		listItems = nil
	}

	for li, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		stripped := strings.TrimLeft(line, " \t")
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		indent := len(line) - len(stripped)

		if strings.HasPrefix(stripped, "- ") {
			item := strings.TrimSpace(stripped[2:])
			if listKey != "" && indent == listIndent {
				listItems = append(listItems, parseScalar(item))
				continue
			}
		}

		key, val, hasVal := splitYAMLKeyValue(stripped)
		if key == "" {
			continue
		}

		for len(stack) > 1 && indent <= stack[len(stack)-1].indent {
			flushList()
			stack = stack[:len(stack)-1]
		}

		if hasVal && val == "" {
			peek := ""
			for j := li + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) != "" {
					peek = lines[j]
					break
				}
			}
			if strings.HasPrefix(strings.TrimLeft(peek, " \t"), "- ") {
				flushList()
				listKey = key
				listIndent = indent + 2
				listItems = []any{}
				continue
			}
			flushList()
			child := map[string]any{}
			stack[len(stack)-1].value[key] = child
			stack = append(stack, frame{indent: indent, value: child})
			continue
		}

		if hasVal {
			flushList()
			stack[len(stack)-1].value[key] = parseScalar(val)
		}
	}

	flushList()
	return root
}

func splitYAMLKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	val := strings.TrimSpace(line[idx+1:])
	return key, val, true
}

func parseScalar(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		if inner == "" {
			return []any{}
		}
		parts := strings.Split(inner, ",")
		out := make([]any, 0, len(parts))
		for _, p := range parts {
			out = append(out, parseScalar(strings.TrimSpace(p)))
		}
		return out
	}
	return s
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}
