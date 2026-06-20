package database

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
)

// ScanWorkspace finds all databases under root.
func ScanWorkspace(root string) ([]*Database, error) {
	files, err := wsutil.WalkMarkdown(root, true)
	if err != nil {
		return nil, err
	}
	var out []*Database
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(data)
		if !IsDatabase(text) {
			continue
		}
		db := Parse(text)
		db.Path = path
		if db.Title == "" || db.Title == filepath.Base(path) {
			if rel, err := filepath.Rel(root, path); err == nil {
				db.Title = strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
			}
		}
		out = append(out, db)
	}
	return out, nil
}

// FindByName resolves a database by path, name, or partial match.
func FindByName(root, query string) (*Database, error) {
	if query == "" {
		return nil, os.ErrNotExist
	}
	if strings.HasSuffix(strings.ToLower(query), ".md") {
		path := query
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, query)
		}
		return ParseFile(path)
	}
	dbs, err := ScanWorkspace(root)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var matches []*Database
	for _, db := range dbs {
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(db.Path), filepath.Ext(db.Path)))
		rel, _ := filepath.Rel(root, db.Path)
		if base == q || strings.EqualFold(db.Title, query) || strings.Contains(strings.ToLower(rel), q) {
			matches = append(matches, db)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return matches[0], nil
	}
	return nil, os.ErrNotExist
}
