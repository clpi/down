package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

// lspTags lists every #tag discovered in the workspace, with mention counts.
// This is the CLI counterpart of the LSP's # tag completion (Notion-style tags).
var lspTags = cobra.Command{
	Use:   "tags [query]",
	Short: "List tags from the workspace knowledge graph",
	Long: `Scan markdown files under the workspace root and list every #tag discovered,
with mention counts. An optional query filters tags by substring (case-insensitive).

The workspace root defaults to the nearest ancestor containing a .down/ directory,
or the current directory. Override with --root.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(args[0])
		}

		root := resolveRoot()
		state := freshState()
		n := loadWorkspace(state, root)

		tags := state.Graph.EntitiesByKind(knowledge.KindTag)
		if query != "" {
			filtered := tags[:0]
			for _, t := range tags {
				if strings.Contains(strings.ToLower(t.Name), query) {
					filtered = append(filtered, t)
				}
			}
			tags = filtered
		}

		if len(tags) == 0 {
			fmt.Printf("No tags found in %s (%d documents scanned).\n", root, n)
			return
		}

		sort.Slice(tags, func(i, j int) bool {
			if tags[i].Mentions != tags[j].Mentions {
				return tags[i].Mentions > tags[j].Mentions
			}
			return tags[i].Name < tags[j].Name
		})

		for _, t := range tags {
			fmt.Printf("#%-24s %3d mentions\n", t.Name, t.Mentions)
		}
		fmt.Printf("\n%d tag(s) across %d document(s).\n", len(tags), n)
	},
}

func init() {
	Lsp.AddCommand(&lspTags)
}
