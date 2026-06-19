package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

var lspMentions = cobra.Command{
	Use:     "mentions [query]",
	Aliases: []string{"mention", "@"},
	Short:   "List @mentions from the workspace knowledge graph",
	Long: `Scan markdown files under the workspace root and list every @person discovered,
with mention counts. An optional query filters people by substring (case-insensitive).

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

		people := state.Graph.EntitiesByKind(knowledge.KindPerson)
		if query != "" {
			filtered := people[:0]
			for _, p := range people {
				if strings.Contains(strings.ToLower(p.Name), query) {
					filtered = append(filtered, p)
				}
			}
			people = filtered
		}

		if len(people) == 0 {
			fmt.Printf("No @mentions found in %s (%d documents scanned).\n", root, n)
			return
		}

		sort.Slice(people, func(i, j int) bool {
			if people[i].Mentions != people[j].Mentions {
				return people[i].Mentions > people[j].Mentions
			}
			return people[i].Name < people[j].Name
		})

		for _, p := range people {
			fmt.Printf("@%-23s %3d mentions\n", p.Name, p.Mentions)
		}
		fmt.Printf("\n%d mention(s) across %d document(s).\n", len(people), n)
	},
}

func init() {
	Lsp.AddCommand(&lspMentions)
}
