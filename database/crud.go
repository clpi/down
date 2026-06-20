package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var defaultSchema = map[string]FieldDef{
	"title":       {Type: "title"},
	"status":      {Type: "select", Options: []string{"Backlog", "Todo", "In Progress", "Done"}},
	"priority":    {Type: "select", Options: []string{"Low", "Medium", "High", "Urgent"}},
	"due_date":    {Type: "date"},
	"tags":        {Type: "multi_select"},
	"description": {Type: "text"},
}

// CreateDatabase writes a new database markdown file.
func CreateDatabase(root, name, schemaKind string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	safe := strings.ReplaceAll(strings.TrimSpace(name), " ", "_")
	if safe == "" {
		safe = "Database"
	}
	path := filepath.Join(root, safe+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("database already exists: %s", path)
	}

	schema := defaultSchema
	switch strings.ToLower(schemaKind) {
	case "tasks", "task":
		schema = defaultSchema
	case "projects", "project":
		schema = map[string]FieldDef{
			"title":    {Type: "title"},
			"status":   {Type: "status", Options: []string{"Planning", "Active", "On Hold", "Done"}},
			"owner":    {Type: "person"},
			"due_date": {Type: "date"},
			"tags":     {Type: "multi_select"},
		}
	case "blank", "":
		schema = map[string]FieldDef{
			"title": {Type: "title"},
			"notes": {Type: "text"},
		}
	}

	cols := make([]string, 0, len(schema))
	for k := range schema {
		cols = append(cols, k)
	}
	stringsSort(cols)

	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: %s\ntype: database\ndatabase:\n", name)
	for _, col := range cols {
		fd := schema[col]
		fmt.Fprintf(&b, "  %s:\n    type: %s\n", col, fd.Type)
		if len(fd.Options) > 0 {
			b.WriteString("    options:\n")
			for _, opt := range fd.Options {
				fmt.Fprintf(&b, "      - %s\n", opt)
			}
		}
	}
	b.WriteString("---\n\n")
	b.WriteString("| " + strings.Join(cols, " | ") + " |\n")
	seps := make([]string, len(cols))
	for i := range seps {
		seps[i] = "---"
	}
	b.WriteString("| " + strings.Join(seps, " | ") + " |\n")

	first := make([]string, len(cols))
	for i, col := range cols {
		first[i] = DefaultValue(schema[col])
		if col == "title" {
			first[i] = name
		}
		if col == "status" {
			first[i] = "Todo"
		}
		if col == "priority" {
			first[i] = "Medium"
		}
		if col == "due_date" {
			first[i] = time.Now().Format("2006-01-02")
		}
	}
	fmt.Fprintf(&b, "| %s |\n", strings.Join(first, " | "))

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// AddRow appends a row to a database file.
func AddRow(path string, values map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	db := Parse(text)
	cols := db.Headers
	if len(cols) == 0 {
		cols = db.ColumnNames()
	}
	if len(cols) == 0 {
		return fmt.Errorf("no table columns found")
	}

	cells := make([]string, len(cols))
	for i, col := range cols {
		key := strings.TrimSpace(col)
		if v, ok := values[key]; ok {
			cells[i] = v
			continue
		}
		if fd, ok := db.Schema[key]; ok {
			cells[i] = DefaultValue(fd)
		}
	}
	rowLine := "| " + strings.Join(cells, " | ") + " |"

	lines := strings.Split(text, "\n")
	insertAt := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		if splitTableRow(lines[i]) != nil {
			insertAt = i + 1
			break
		}
	}
	out := append(lines[:insertAt], append([]string{rowLine}, lines[insertAt:]...)...)
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

// UpdateCell updates a single cell in a database table row.
func UpdateCell(path string, rowLine int, column string, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	if rowLine < 1 || rowLine > len(lines) {
		return fmt.Errorf("row line out of range: %d", rowLine)
	}
	db := Parse(string(data))
	cols := db.Headers
	if len(cols) == 0 {
		cols = db.ColumnNames()
	}
	colIdx := -1
	column = strings.TrimSpace(column)
	for i, c := range cols {
		if strings.EqualFold(strings.TrimSpace(c), column) {
			colIdx = i
			break
		}
	}
	if colIdx < 0 {
		return fmt.Errorf("column not found: %s", column)
	}
	cells := splitTableRow(lines[rowLine-1])
	if cells == nil {
		return fmt.Errorf("not a table row at line %d", rowLine)
	}
	for len(cells) < len(cols) {
		cells = append(cells, "")
	}
	cells[colIdx] = value
	lines[rowLine-1] = "| " + strings.Join(cells, " | ") + " |"
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

