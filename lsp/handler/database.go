package handler

import (
	"path/filepath"
	"strings"

	db "github.com/clpi/down/database"
)

// DatabaseInfo is a lightweight summary for LSP/CLI consumers.
type DatabaseInfo struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Rows  int    `json:"rows"`
	Type  string `json:"type"`
}

// ComputeDatabases scans workspace markdown files for Notion-style databases.
func (s *State) ComputeDatabases() []DatabaseInfo {
	var out []DatabaseInfo
	seen := map[string]bool{}
	for uri := range s.Documents {
		path := strings.TrimPrefix(uri, "file://")
		if seen[path] {
			continue
		}
		seen[path] = true
		text := s.Documents[uri]
		if !db.IsDatabase(text) {
			continue
		}
		d := db.Parse(text)
		d.Path = path
		out = append(out, DatabaseInfo{
			Title: d.Title,
			Path:  path,
			Rows:  len(d.Rows),
			Type:  d.Type,
		})
	}
	return out
}

// WorkspaceDatabaseRoot guesses a workspace root from loaded documents.
func (s *State) WorkspaceDatabaseRoot() string {
	var root string
	for uri := range s.Documents {
		path := strings.TrimPrefix(uri, "file://")
		if root == "" || len(path) < len(root) {
			root = filepath.Dir(path)
		}
	}
	return root
}
