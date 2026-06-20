package link

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp/handler"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

var linkRoot string

func scanState(root string) *handler.State {
	home, _ := os.UserHomeDir()
	state := &handler.State{
		Graph:     knowledge.NewFreshGraph(filepath.Join(home, ".down", "knowledge.json")),
		Documents: make(map[string]string),
	}
	files, _ := wsutil.WalkMarkdown(root, true)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		uri := "file://" + path
		state.Documents[uri] = string(data)
		knowledge.ExtractFromDocument(state.Graph, uri, string(data))
	}
	return state
}

func resolveFile(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if abs, err := filepath.Abs(path); err == nil {
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	candidate := filepath.Join(root, path)
	if abs, err := filepath.Abs(candidate); err == nil {
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	return ""
}

var Link = cobra.Command{
	Use:     "link",
	Aliases: []string{"ln"},
	Short:   "Wiki links and backlinks",
}

var linkBacklinks = cobra.Command{
	Use:   "backlinks <file>",
	Short: "Show documents referencing a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(linkRoot)
		state := scanState(root)
		abs := resolveFile(root, args[0])
		if abs == "" {
			fmt.Printf("No such file: %s\n", args[0])
			return
		}
		uri := "file://" + abs
		if _, ok := state.Documents[uri]; !ok {
			data, err := os.ReadFile(abs)
			if err != nil {
				fmt.Printf("No such file: %s\n", args[0])
				return
			}
			state.Documents[uri] = string(data)
			knowledge.ExtractFromDocument(state.Graph, uri, string(data))
		}
		result := state.ComputeBacklinks(uri)
		if result.Count == 0 {
			fmt.Printf("No backlinks to %s.\n", args[0])
			return
		}
		bl := result.Backlinks
		sort.SliceStable(bl, func(i, j int) bool {
			if bl[i].SourceURI != bl[j].SourceURI {
				return bl[i].SourceURI < bl[j].SourceURI
			}
			return bl[i].Line < bl[j].Line
		})
		fmt.Printf("%s — %d backlink(s)\n", result.Title, result.Count)
		for _, b := range bl {
			ctx := b.Context
			if ctx == "" {
				ctx = b.Kind
			}
			src := strings.TrimPrefix(b.SourceURI, "file://")
			if rel, err := filepath.Rel(root, src); err == nil && !strings.HasPrefix(rel, "..") {
				src = rel
			}
			fmt.Printf("  %s:%d  [%s]  %s\n", src, b.Line+1, b.Kind, ctx)
		}
	},
}

func init() {
	Link.PersistentFlags().StringVar(&linkRoot, "root", "", "Workspace root")
	Link.AddCommand(&linkBacklinks)
	Link.Run = linkBacklinks.Run
}
