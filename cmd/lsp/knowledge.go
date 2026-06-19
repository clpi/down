package lsp

import (
	"fmt"
	"sort"

	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

// lspKnowledge groups the knowledge-graph operations surfaced by the LSP
// (down.knowledge.*) into CLI subcommands.
var lspKnowledge = cobra.Command{
	Use:     "knowledge <command>",
	Aliases: []string{"kg", "graph"},
	Short:   "Query the workspace knowledge graph",
	Long: `Inspect the knowledge graph built from the workspace markdown files:
entities (people, concepts, tags, dates, ...), relations between them, and
documents related by shared entities.

Each subcommand scans the workspace root (nearest .down/ ancestor or cwd;
override with --root) unless noted.`,
}

var kgSummary = cobra.Command{
	Use:   "summary",
	Short: "Print a summary of the knowledge graph",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		state := freshState()
		n := loadWorkspace(state, resolveRoot())
		out := state.Graph.Summary()
		fmt.Printf("%s\nDocuments scanned: %d\n", out, n)
	},
}

var kgSearch = cobra.Command{
	Use:   "search <query>",
	Short: "Search entities by name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		state := freshState()
		loadWorkspace(state, resolveRoot())
		results := state.Graph.Search(args[0])
		if len(results) == 0 {
			fmt.Printf("No entities match %q.\n", args[0])
			return
		}
		for _, ent := range results {
			fmt.Printf("%-28s %-10s %3d mentions\n", ent.Name, ent.Kind, ent.Mentions)
		}
		fmt.Printf("\n%d match(es).\n", len(results))
	},
}

var kgEntities = cobra.Command{
	Use:   "entities [kind]",
	Short: "List entities, optionally filtered by kind",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		state := freshState()
		loadWorkspace(state, resolveRoot())

		var entities []*knowledge.Entity
		if len(args) == 1 {
			entities = state.Graph.EntitiesByKind(knowledge.EntityKind(args[0]))
		} else {
			entities = state.Graph.AllEntities()
		}
		if len(entities) == 0 {
			fmt.Println("No entities found.")
			return
		}
		sort.Slice(entities, func(i, j int) bool {
			if entities[i].Kind != entities[j].Kind {
				return entities[i].Kind < entities[j].Kind
			}
			return entities[i].Name < entities[j].Name
		})
		for _, ent := range entities {
			fmt.Printf("%-28s %-10s %3d mentions\n", ent.Name, ent.Kind, ent.Mentions)
		}
		fmt.Printf("\n%d entit(ies).\n", len(entities))
	},
}

var kgRelations = cobra.Command{
	Use:   "relations <entity>",
	Short: "Show relations involving an entity",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		state := freshState()
		loadWorkspace(state, resolveRoot())
		results := state.Graph.Search(args[0])
		if len(results) == 0 {
			fmt.Printf("No entity matches %q.\n", args[0])
			return
		}
		for _, ent := range results {
			fmt.Printf("\n%s (%s)\n", ent.Name, ent.Kind)
			outgoing := state.Graph.RelationsFrom(ent.ID)
			incoming := state.Graph.RelationsTo(ent.ID)
			if len(outgoing) > 0 {
				fmt.Println("  outgoing:")
				for _, r := range outgoing {
					if t, ok := state.Graph.Entities[r.To]; ok {
						fmt.Printf("    -> %-12s %s (%s)\n", r.Kind, t.Name, t.Kind)
					}
				}
			}
			if len(incoming) > 0 {
				fmt.Println("  incoming:")
				for _, r := range incoming {
					if s, ok := state.Graph.Entities[r.From]; ok {
						fmt.Printf("    <- %-12s from %s (%s)\n", r.Kind, s.Name, s.Kind)
					}
				}
			}
			if len(outgoing) == 0 && len(incoming) == 0 {
				fmt.Println("  no relations")
			}
		}
	},
}

var kgRelated = cobra.Command{
	Use:   "related <file>",
	Short: "Find documents related to a file by shared entities",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		state := freshState()
		loadWorkspace(state, resolveRoot())
		// ensureFile avoids re-extracting an already-indexed file, which would
		// wipe its entity→document mapping (Graph.byDoc).
		uri, ok := ensureFile(state, args[0])
		if !ok {
			fmt.Printf("No such file: %s\n", args[0])
			return
		}
		entities := state.Graph.EntitiesByDocument(uri)
		if len(entities) == 0 {
			fmt.Printf("No entities found in %s.\n", shortPath(uri))
			return
		}
		related := make(map[string]int)
		for _, ent := range entities {
			if ent == nil {
				continue
			}
			for _, src := range ent.Sources {
				if src.URI == uri {
					continue
				}
				related[src.URI]++
			}
		}
		if len(related) == 0 {
			fmt.Println("No related documents.")
			return
		}
		type kv struct {
			uri   string
			count int
		}
		rows := make([]kv, 0, len(related))
		for u, c := range related {
			rows = append(rows, kv{u, c})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].count > rows[j].count })
		for _, r := range rows {
			fmt.Printf("%3d  %s\n", r.count, shortPath(r.uri))
		}
		fmt.Printf("\n%d related document(s).\n", len(related))
	},
}

var kgReindex = cobra.Command{
	Use:   "reindex",
	Short: "Rebuild and persist the knowledge graph",
	Long: `Re-scan the workspace root and overwrite the persisted knowledge graph at
~/.down/knowledge.json with a fresh build. Run this after large structural
changes so the LSP and CLI agree on the index.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		state := freshState()
		n := loadWorkspace(state, resolveRoot())
		if err := state.Graph.Save(); err != nil {
			fmt.Printf("Error saving graph: %v\n", err)
			return
		}
		fmt.Printf("Reindexed %d document(s) into the knowledge graph.\n", n)
	},
}

func init() {
	lspKnowledge.AddCommand(&kgSummary)
	lspKnowledge.AddCommand(&kgSearch)
	lspKnowledge.AddCommand(&kgEntities)
	lspKnowledge.AddCommand(&kgRelations)
	lspKnowledge.AddCommand(&kgRelated)
	lspKnowledge.AddCommand(&kgReindex)
	Lsp.AddCommand(&lspKnowledge)
}
