package database

import "strings"

// Row is one database record keyed by column name.
type Row map[string]string

// ParseMarkdownTable extracts rows from a markdown pipe table starting at startLine (1-based).
func ParseMarkdownTable(lines []string, startLine int) (headers []string, rows []Row, tableStart int) {
	if startLine < 1 {
		startLine = 1
	}
	inTable := false
	for i := startLine - 1; i < len(lines); i++ {
		line := lines[i]
		if !inTable {
			cols := splitTableRow(line)
			if len(cols) > 0 {
				headers = cols
				inTable = true
				tableStart = i + 1
			}
			continue
		}
		if isAlignmentRow(line) {
			continue
		}
		cols := splitTableRow(line)
		if len(cols) == 0 {
			break
		}
		row := Row{}
		for j, header := range headers {
			key := strings.TrimSpace(header)
			val := ""
			if j < len(cols) {
				val = strings.TrimSpace(cols[j])
			}
			row[key] = val
		}
		row["_line"] = itoa(i + 1)
		rows = append(rows, row)
	}
	return headers, rows, tableStart
}

func splitTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return nil
	}
	parts := strings.Split(line, "|")
	var cols []string
	for _, p := range parts {
		cols = append(cols, strings.TrimSpace(p))
	}
	if len(cols) > 0 && cols[0] == "" {
		cols = cols[1:]
	}
	if len(cols) > 0 && cols[len(cols)-1] == "" {
		cols = cols[:len(cols)-1]
	}
	hasContent := false
	for _, c := range cols {
		if c != "" {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return nil
	}
	return cols
}

func isAlignmentRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") || !strings.Contains(trimmed, "-") {
		return false
	}
	for _, part := range strings.Split(trimmed, "|") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		for _, ch := range p {
			if ch != '-' && ch != ':' && ch != ' ' {
				return false
			}
		}
	}
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
