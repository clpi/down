package lsp

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

// lspBacklinks shows every document that references the given file, mirroring
// Notion's backlinks panel. It scans the workspace, then runs the same
// ComputeBacklinks routine the LSP uses for hover and the backlinks command.
var lspBacklinks = cobra.Command{
	Use:     "backlinks <file>",
	Aliases: []string{"bl"},
	Short:   "Show documents that reference a file",
	Long: `Scan the workspace and list every document that references <file> via a wiki
link ([[...]]), @mention, #tag, or knowledge-graph relation.

The workspace root defaults to the nearest ancestor with a .down/ directory, or
the current directory. Override with --root.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolveRoot()
		state := freshState()
		loadWorkspace(state, root)

		// Ensure the target file itself is loaded even if it sits outside root.
		uri, ok := loadFile(state, args[0])
		if !ok {
			fmt.Printf("No such file: %s\n", args[0])
			return
		}

		result := state.ComputeBacklinks(uri)
		if result.Count == 0 {
			fmt.Printf("No backlinks to %s.\n", shortPath(uri))
			return
		}

		// Stable, readable ordering by source then line.
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
			fmt.Printf("  %s:%d  [%s]  %s\n", shortPath(b.SourceURI), b.Line+1, b.Kind, ctx)
		}
	},
}

func init() {
	Lsp.AddCommand(&lspBacklinks)
}
