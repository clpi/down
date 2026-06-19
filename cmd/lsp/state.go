package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/lsp/handler"
	"github.com/clpi/down/lsp/knowledge"
)

// rootFlag is the --root value shared by every `down lsp` subcommand that scans
// a workspace. When empty, resolveRoot walks up from cwd looking for .down/.
var rootFlag string

var markdownExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".mdx":      true,
	".txt":      true,
}

// freshState builds a handler.State around an empty knowledge graph bound to the
// persisted store path (so a later Save() lands in the right place). It never
// loads existing state, so mention counts reflect only what loadWorkspace scans.
func freshState() *handler.State {
	home, _ := os.UserHomeDir()
	storePath := filepath.Join(home, ".down", "knowledge.json")
	return &handler.State{
		Graph:     knowledge.NewFreshGraph(storePath),
		Documents: make(map[string]string),
	}
}

// loadWorkspace walks root for markdown files, populating state.Documents and
// extracting entities into state.Graph. It skips .down/.git/node_modules/
// .obsidian/.trash directories and does not persist the graph. Returns the
// number of documents loaded.
func loadWorkspace(state *handler.State, root string) int {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	count := 0
	_ = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			switch filepath.Base(path) {
			case ".git", "node_modules", ".obsidian", ".trash", ".down":
				return filepath.SkipDir
			}
			return nil
		}
		if !markdownExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		uri := "file://" + path
		state.Documents[uri] = string(data)
		knowledge.ExtractFromDocument(state.Graph, uri, string(data))
		count++
		return nil
	})
	return count
}

// loadFile reads a single markdown file into state.Documents under its file URI.
func loadFile(state *handler.State, path string) (string, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", false
	}
	uri := "file://" + abs
	state.Documents[uri] = string(data)
	knowledge.ExtractFromDocument(state.Graph, uri, string(data))
	return uri, true
}

// resolveRoot returns the workspace to scan: an explicit --root, the nearest
// ancestor containing a .down/ directory, or the current directory.
func resolveRoot() string {
	if rootFlag != "" {
		return rootFlag
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := cwd; dir != "" && dir != string(os.PathSeparator); {
		if info, err := os.Stat(filepath.Join(dir, ".down")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

// shortPath renders a file:// URI as a tidier relative path when possible.
func shortPath(uri string) string {
	p := strings.TrimPrefix(uri, "file://")
	if rel, err := filepath.Rel(cwd(), p); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return p
}

func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return ""
	}
	return d
}
