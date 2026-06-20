package database

import (
	"strconv"
	"strings"
	"time"
)

// Filter describes a single filter condition.
type Filter struct {
	Property string
	Operator string
	Value    string
}

// Sort describes a sort column.
type Sort struct {
	Property  string
	Direction string // ascending | descending
}

// FilterRows applies filters to rows.
func FilterRows(rows []Row, filters []Filter) []Row {
	if len(filters) == 0 {
		return rows
	}
	out := make([]Row, 0, len(rows))
	for _, row := range rows {
		if matchesAll(row, filters) {
			out = append(out, row)
		}
	}
	return out
}

func matchesAll(row Row, filters []Filter) bool {
	for _, f := range filters {
		if !matchOne(row, f) {
			return false
		}
	}
	return true
}

func matchOne(row Row, f Filter) bool {
	op := strings.ToLower(f.Operator)
	if op == "" {
		op = "eq"
	}
	val := row[f.Property]
	switch op {
	case "is_empty", "empty":
		return strings.TrimSpace(val) == ""
	case "is_not_empty", "not_empty":
		return strings.TrimSpace(val) != ""
	case "contains":
		return strings.Contains(strings.ToLower(val), strings.ToLower(f.Value))
	case "does_not_contain", "not_contains":
		return !strings.Contains(strings.ToLower(val), strings.ToLower(f.Value))
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(val), strings.ToLower(f.Value))
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(val), strings.ToLower(f.Value))
	case "neq", "!=", "not":
		return !equalFold(val, f.Value)
	case "gt", ">":
		return compareValues(val, f.Value) > 0
	case "gte", ">=":
		return compareValues(val, f.Value) >= 0
	case "lt", "<":
		return compareValues(val, f.Value) < 0
	case "lte", "<=":
		return compareValues(val, f.Value) <= 0
	default:
		return equalFold(val, f.Value)
	}
}

func equalFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func compareValues(a, b string) int {
	if da, ok := parseDate(a); ok {
		if db, ok2 := parseDate(b); ok2 {
			if da.Before(db) {
				return -1
			}
			if da.After(db) {
				return 1
			}
			return 0
		}
	}
	fa, erra := strconv.ParseFloat(strings.TrimSpace(a), 64)
	fb, errb := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if erra == nil && errb == nil {
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		t, err := time.Parse("2006-01-02", s[:10])
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// SortRows sorts rows by the given sort specs.
func SortRows(rows []Row, sorts []Sort) []Row {
	if len(sorts) == 0 {
		return rows
	}
	out := make([]Row, len(rows))
	copy(out, rows)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if lessRow(out[j], out[i], sorts) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func lessRow(a, b Row, sorts []Sort) bool {
	for _, s := range sorts {
		cmp := compareValues(a[s.Property], b[s.Property])
		if cmp == 0 {
			continue
		}
		if strings.EqualFold(s.Direction, "descending") || s.Direction == "desc" {
			return cmp > 0
		}
		return cmp < 0
	}
	return false
}

// GroupRows groups rows by a property value.
func GroupRows(rows []Row, property string) map[string][]Row {
	groups := map[string][]Row{}
	for _, row := range rows {
		key := strings.TrimSpace(row[property])
		if key == "" {
			key = "(empty)"
		}
		groups[key] = append(groups[key], row)
	}
	return groups
}
