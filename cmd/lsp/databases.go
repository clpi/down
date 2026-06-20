package lsp

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var lspDatabases = cobra.Command{
	Use:     "databases",
	Aliases: []string{"database", "db"},
	Short:   "List Notion-style markdown databases in the workspace",
	Long: `Scan markdown files for type: database frontmatter and markdown tables.
Lists each database with row counts and source paths.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := resolveRoot()
		state := freshState()
		n := loadWorkspace(state, root)
		dbs := state.ComputeDatabases()
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(args[0])
		}
		if len(dbs) == 0 {
			fmt.Printf("No databases in %s (%d documents scanned).\n", root, n)
			return
		}
		shown := 0
		for _, d := range dbs {
			rel := shortPath(d.Path)
			if query != "" {
				blob := strings.ToLower(d.Title + " " + rel)
				if !strings.Contains(blob, query) {
					continue
				}
			}
			fmt.Printf("%-28s %3d rows  %s\n", d.Title, d.Rows, rel)
			shown++
		}
		fmt.Printf("\n%d database(s) shown across %d document(s).\n", shown, n)
	},
}

func init() {
	Lsp.AddCommand(&lspDatabases)
}
