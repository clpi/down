package database

import (
	"os"
	"path/filepath"
	"strings"
)

type Database struct {
	Path      string              `json:"path"`
	Title     string              `json:"title"`
	Type      string              `json:"type"`
	Schema    map[string]FieldDef `json:"schema"`
	Headers   []string            `json:"headers"`
	Rows      []Row               `json:"rows"`
	TableLine int                 `json:"table_line"`
	Inline    bool                `json:"inline"`
}

func ParseFile(path string) (*Database, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	db := Parse(string(data))
	db.Path = path
	return db, nil
}

func Parse(text string) *Database {
	lines := strings.Split(text, "\n")
	db := &Database{Type: "table"}

	fm, _, fmEnd, ok := ParseFrontmatter(text)
	if ok {
		db.Title = asString(fm["title"])
		if t := asString(fm["type"]); t != "" {
			db.Type = t
		}
		schemaRaw := asMap(fm["database"])
		if schemaRaw == nil {
			schemaRaw = asMap(fm["schema"])
		}
		db.Schema = NormalizeSchema(schemaRaw)
		start := fmEnd + 2
		if start < 1 {
			start = 1
		}
		headers, rows, tableStart := ParseMarkdownTable(lines, start)
		db.Headers = headers
		db.Rows = rows
		db.TableLine = tableStart
	} else {
		headers, rows, tableStart := ParseMarkdownTable(lines, 1)
		db.Headers = headers
		db.Rows = rows
		db.TableLine = tableStart
	}

	if len(db.Schema) == 0 && len(db.Headers) > 0 {
		db.Schema = SchemaFromHeaders(db.Headers)
	}
	if db.Title == "" {
		db.Title = filepath.Base(db.Path)
	}
	return db
}

func IsDatabase(text string) bool {
	db := Parse(text)
	if len(db.Rows) == 0 || len(db.Headers) == 0 {
		return false
	}
	fm, _, _, ok := ParseFrontmatter(text)
	if !ok {
		return false
	}
	if asString(fm["type"]) == "database" {
		return true
	}
	if fm["database"] != nil || fm["schema"] != nil {
		return true
	}
	return false
}

func (db *Database) TitleColumn() string {
	for _, key := range []string{"title", "Title", "name", "Name"} {
		if _, ok := db.Schema[key]; ok {
			return key
		}
	}
	if len(db.Headers) > 0 {
		return strings.TrimSpace(db.Headers[0])
	}
	return "title"
}

func (db *Database) ColumnNames() []string {
	if len(db.Schema) > 0 {
		keys := make([]string, 0, len(db.Schema))
		for k := range db.Schema {
			keys = append(keys, k)
		}
		stringsSort(keys)
		return keys
	}
	return db.Headers
}

func stringsSort(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
