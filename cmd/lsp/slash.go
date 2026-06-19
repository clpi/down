package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clpi/down/lsp/handler/completion/entries"
	"github.com/spf13/cobra"
)

var slashCategory string
var slashInsert bool

// lspSlash lists the Notion-style slash commands the LSP exposes in the editor.
// It mirrors Notion's `/` menu: categories, a one-line detail, and (optionally)
// the snippet the editor would insert.
var lspSlash = cobra.Command{
	Use:   "slash [query]",
	Short: "List Notion-style slash commands",
	Long: `List the Notion-style slash commands (/text, /table, /callout, ...) that the
down LSP offers inside the editor. An optional query filters by label, detail, or
description (case-insensitive). Use --category to narrow to a single category and
--insert to preview the snippet each command inserts.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(args[0])
		}

		byCat := make(map[string][]entries.SlashCommand)
		total := 0
		for _, c := range entries.SlashCommands {
			if slashCategory != "" && !strings.EqualFold(c.Category, slashCategory) {
				continue
			}
			if query != "" {
				hay := strings.ToLower(c.Label + " " + c.Detail + " " + c.Description)
				if !strings.Contains(hay, query) {
					continue
				}
			}
			byCat[c.Category] = append(byCat[c.Category], c)
			total++
		}

		if total == 0 {
			fmt.Println("No slash commands match.")
			return
		}

		cats := make([]string, 0, len(byCat))
		for c := range byCat {
			cats = append(cats, c)
		}
		sort.Strings(cats)

		for _, cat := range cats {
			fmt.Printf("\n%s\n", cat)
			for _, c := range byCat[cat] {
				fmt.Printf("  %-16s %s\n", c.Label, c.Detail)
				if slashInsert && c.InsertText != "" {
					fmt.Printf("  %s\n", indent(c.InsertText, "    "))
				}
			}
		}
		fmt.Printf("\n%d command(s).\n", total)
	},
}

func indent(s, pad string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

func init() {
	lspSlash.Flags().StringVarP(&slashCategory, "category", "c", "",
		"Filter by category (Basic, Media, Callout, Advanced, Database, Sync, Date, Inline)")
	lspSlash.Flags().BoolVarP(&slashInsert, "insert", "i", false, "Show the snippet each command inserts")
	Lsp.AddCommand(&lspSlash)
}
