package database

import (
	"path/filepath"
	"strconv"
	"strings"
)

// WorkspaceContext holds all databases in a workspace for cross-db relations/rollups.
type WorkspaceContext struct {
	Root      string
	All       []*Database
	byTitle   map[string]*Database
	byBase    map[string]*Database
	byRelPath map[string]*Database
}

// LoadWorkspaceContext scans the workspace and indexes databases by title/path.
func LoadWorkspaceContext(root string) (*WorkspaceContext, error) {
	dbs, err := ScanWorkspace(root)
	if err != nil {
		return nil, err
	}
	ctx := &WorkspaceContext{
		Root:      root,
		All:       dbs,
		byTitle:   map[string]*Database{},
		byBase:    map[string]*Database{},
		byRelPath: map[string]*Database{},
	}
	for _, d := range dbs {
		if d.Title != "" {
			ctx.byTitle[strings.ToLower(strings.TrimSpace(d.Title))] = d
		}
		base := strings.TrimSuffix(filepath.Base(d.Path), filepath.Ext(d.Path))
		ctx.byBase[strings.ToLower(base)] = d
		if rel, err := filepath.Rel(root, d.Path); err == nil {
			ctx.byRelPath[strings.ToLower(rel)] = d
		}
	}
	return ctx, nil
}

// FindDatabase resolves a database reference by title, basename, or relative path.
func (ctx *WorkspaceContext) FindDatabase(ref string) *Database {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	lower := strings.ToLower(ref)
	if d, ok := ctx.byTitle[lower]; ok {
		return d
	}
	if d, ok := ctx.byBase[lower]; ok {
		return d
	}
	if d, ok := ctx.byRelPath[lower]; ok {
		return d
	}
	ref = strings.TrimSuffix(ref, ".md")
	if d, ok := ctx.byBase[strings.ToLower(ref)]; ok {
		return d
	}
	for _, d := range ctx.All {
		if strings.EqualFold(d.Title, ref) {
			return d
		}
		rel, _ := filepath.Rel(ctx.Root, d.Path)
		if strings.EqualFold(rel, ref) || strings.Contains(strings.ToLower(rel), lower) {
			return d
		}
	}
	return nil
}

// ResolveComputed fills formula and rollup columns for each row.
func ResolveComputed(db *Database, rows []Row, ctx *WorkspaceContext) []Row {
	if db == nil || len(rows) == 0 {
		return rows
	}
	out := make([]Row, len(rows))
	for i, row := range rows {
		out[i] = cloneRow(row)
	}
	for col, fd := range db.Schema {
		switch fd.Type {
		case "formula":
			for i := range out {
				out[i][col] = computeFormula(out[i], fd, out)
			}
		case "rollup":
			for i := range out {
				out[i][col] = computeRollup(db, out[i], fd, out, ctx)
			}
		}
	}
	return out
}

func cloneRow(row Row) Row {
	out := make(Row, len(row))
	for k, v := range row {
		out[k] = v
	}
	return out
}

func computeFormula(row Row, fd FieldDef, all []Row) string {
	expr := strings.TrimSpace(fd.Formula)
	if expr == "" {
		return ""
	}
	result := expr
	for k, v := range row {
		if strings.HasPrefix(k, "_") {
			continue
		}
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}
	if isMathExpr(result) {
		if val, ok := evalMath(result); ok {
			return formatNumber(val)
		}
	}
	if strings.HasPrefix(result, "length(") && strings.HasSuffix(result, ")") {
		inner := strings.TrimSpace(result[7 : len(result)-1])
		if v, ok := row[inner]; ok {
			return strconv.Itoa(len(v))
		}
	}
	if strings.HasPrefix(result, "if(") && strings.HasSuffix(result, ")") {
		parts := splitTopLevel(result[3 : len(result)-1])
		if len(parts) >= 2 {
			cond := strings.TrimSpace(parts[0])
			if evalCondition(row, cond) {
				return strings.TrimSpace(parts[1])
			}
			if len(parts) >= 3 {
				return strings.TrimSpace(parts[2])
			}
			return ""
		}
	}
	return result
}

func computeRollup(db *Database, row Row, fd FieldDef, all []Row, ctx *WorkspaceContext) string {
	relationCol := strings.TrimSpace(fd.Relation)
	targetCol := strings.TrimSpace(fd.Target)
	aggregate := strings.TrimSpace(fd.Aggregate)
	if aggregate == "" {
		aggregate = "count"
	}
	if relationCol == "" || targetCol == "" {
		return ""
	}

	relationFD, ok := db.Schema[relationCol]
	if !ok {
		relationFD = FieldDef{Type: "relation"}
	}

	var relatedRows []Row
	if relationFD.Type == "relation" && relationFD.Database != "" && ctx != nil {
		targetDB := ctx.FindDatabase(relationFD.Database)
		if targetDB != nil {
			linked := splitLinkedTitles(row[relationCol])
			titleCol := targetDB.TitleColumn()
			for _, r := range targetDB.Rows {
				title := strings.TrimSpace(r[titleCol])
				for _, link := range linked {
					if strings.EqualFold(title, link) {
						relatedRows = append(relatedRows, r)
						break
					}
				}
			}
		}
	} else {
		key := row[relationCol]
		for _, r := range all {
			if r[relationCol] == key {
				relatedRows = append(relatedRows, r)
			}
		}
	}

	return aggregateValues(relatedRows, targetCol, aggregate)
}

func splitLinkedTitles(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func aggregateValues(rows []Row, property, aggregate string) string {
	switch strings.ToLower(aggregate) {
	case "count":
		return strconv.Itoa(len(rows))
	case "count_unique", "unique":
		seen := map[string]bool{}
		for _, r := range rows {
			seen[strings.TrimSpace(r[property])] = true
		}
		delete(seen, "")
		return strconv.Itoa(len(seen))
	case "sum":
		total := 0.0
		for _, r := range rows {
			if n, ok := parseNumber(r[property]); ok {
				total += n
			}
		}
		return formatNumber(total)
	case "average", "avg":
		total, count := 0.0, 0
		for _, r := range rows {
			if n, ok := parseNumber(r[property]); ok {
				total += n
				count++
			}
		}
		if count == 0 {
			return "0"
		}
		return formatNumber(total / float64(count))
	case "min":
		var minVal *float64
		var minStr string
		for _, r := range rows {
			v := strings.TrimSpace(r[property])
			if v == "" {
				continue
			}
			if n, ok := parseNumber(v); ok {
				if minVal == nil || n < *minVal {
					minVal = &n
					minStr = v
				}
			} else if minStr == "" || v < minStr {
				minStr = v
			}
		}
		if minStr != "" {
			return minStr
		}
		return ""
	case "max":
		var maxVal *float64
		var maxStr string
		for _, r := range rows {
			v := strings.TrimSpace(r[property])
			if v == "" {
				continue
			}
			if n, ok := parseNumber(v); ok {
				if maxVal == nil || n > *maxVal {
					maxVal = &n
					maxStr = v
				}
			} else if maxStr == "" || v > maxStr {
				maxStr = v
			}
		}
		if maxStr != "" {
			return maxStr
		}
		return ""
	case "join", "concat", "list":
		var parts []string
		for _, r := range rows {
			v := strings.TrimSpace(r[property])
			if v != "" {
				parts = append(parts, v)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return strconv.Itoa(len(rows))
	}
}

func isMathExpr(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
		case c == '+' || c == '-' || c == '*' || c == '/' || c == '.' || c == ' ' || c == '(' || c == ')':
		default:
			return false
		}
	}
	return strings.ContainsAny(s, "0123456789")
}

func evalMath(expr string) (float64, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, false
	}
	val, err := strconv.ParseFloat(expr, 64)
	if err == nil {
		return val, true
	}
	for _, op := range []string{"+", "-", "*", "/"} {
		if idx := strings.LastIndex(expr, op); idx > 0 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+1:])
			l, lok := parseNumber(left)
			r, rok := parseNumber(right)
			if lok && rok {
				switch op {
				case "+":
					return l + r, true
				case "-":
					return l - r, true
				case "*":
					return l * r, true
				case "/":
					if r != 0 {
						return l / r, true
					}
				}
			}
		}
	}
	return 0, false
}

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(s, 64)
	return n, err == nil
}

func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'f', 2, 64)
}

func evalCondition(row Row, cond string) bool {
	cond = strings.TrimSpace(cond)
	if cond == "true" {
		return true
	}
	if cond == "false" {
		return false
	}
	if strings.HasPrefix(cond, "not ") {
		return !evalCondition(row, strings.TrimSpace(cond[4:]))
	}
	for k, v := range row {
		if cond == k {
			return v != "" && !strings.EqualFold(v, "false") && !strings.EqualFold(v, "no")
		}
		if cond == k+" == true" {
			return strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
		}
		if cond == k+" == false" {
			return strings.EqualFold(v, "false") || strings.EqualFold(v, "no") || v == ""
		}
	}
	return false
}

func splitTopLevel(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}
