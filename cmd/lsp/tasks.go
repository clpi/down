package lsp

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var tasksOpenOnly bool
var tasksDoneOnly bool

// lspTasks lists markdown checkbox tasks across the workspace, mirroring
// Notion's task database view. Use --open / --done to filter by state.
var lspTasks = cobra.Command{
	Use:     "tasks",
	Aliases: []string{"task"},
	Short:   "List markdown tasks across the workspace",
	Long: `Scan markdown files under the workspace root and list every checkbox task
(- [ ] / - [x]) grouped by source document, with line numbers and state.
Use --open or --done to filter by completion state.

The workspace root defaults to the nearest ancestor with a .down/ directory, or
the current directory. Override with --root.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		root := resolveRoot()
		state := freshState()
		n := loadWorkspace(state, root)
		result := state.ComputeTasks()

		// Group by source document, preserving first-seen order.
		order := make([]string, 0)
		byDoc := make(map[string][]taskLine)
		open, shown := 0, 0
		for _, t := range result.Tasks {
			if t.Completed {
				if tasksOpenOnly {
					continue
				}
			} else {
				open++
				if tasksDoneOnly {
					continue
				}
			}
			if _, ok := byDoc[t.URI]; !ok {
				order = append(order, t.URI)
			}
			byDoc[t.URI] = append(byDoc[t.URI], taskLine{
				text:      t.Text,
				line:      t.Line + 1,
				completed: t.Completed,
			})
			shown++
		}

		if shown == 0 {
			which := "tasks"
			switch {
			case tasksOpenOnly:
				which = "open tasks"
			case tasksDoneOnly:
				which = "completed tasks"
			}
			fmt.Printf("No %s in %s (%d documents scanned).\n", which, root, n)
			return
		}

		for _, uri := range order {
			title := docTitle(uri, byDoc[uri])
			fmt.Printf("\n%s\n", title)
			for _, tl := range byDoc[uri] {
				box := "[ ]"
				if tl.completed {
					box = "[x]"
				}
				fmt.Printf("  %s %s  (%s:%d)\n", box, tl.text, shortPath(uri), tl.line)
			}
		}
		fmt.Printf("\n%d task(s) shown (%d open, %d total) across %d document(s).\n", shown, open, result.Count, n)
	},
}

type taskLine struct {
	text      string
	line      int
	completed bool
}

// docTitle picks a readable heading for a document group.
func docTitle(uri string, _ []taskLine) string {
	base := shortPath(uri)
	base = strings.TrimSuffix(base, ".md")
	base = strings.TrimSuffix(base, ".markdown")
	return base
}

func init() {
	lspTasks.Flags().BoolVar(&tasksOpenOnly, "open", false, "Show only incomplete tasks")
	lspTasks.Flags().BoolVar(&tasksDoneOnly, "done", false, "Show only completed tasks")
	Lsp.AddCommand(&lspTasks)
}
