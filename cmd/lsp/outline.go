package lsp

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/spf13/cobra"
)

// lspOutline prints the hierarchical heading/task outline of a single markdown
// document — the same tree the LSP exposes via textDocument/documentSymbol,
// modelled on Notion's sidebar outline.
var lspOutline = cobra.Command{
	Use:     "outline <file>",
	Aliases: []string{"ol"},
	Short:   "Show the heading/task outline of a document",
	Long: `Print the hierarchical outline (headings and nested checkbox tasks) of a
single markdown document. Headings nest by level; tasks appear under their
enclosing heading.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		state := freshState()
		uri, ok := loadFile(state, args[0])
		if !ok {
			fmt.Printf("No such file: %s\n", args[0])
			return
		}

		res, err := state.Symbol(nil, &protocol.DocumentSymbolParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentUri(uri)},
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		symbols, ok := res.([]protocol.DocumentSymbol)
		if !ok || len(symbols) == 0 {
			fmt.Println("No outline (no headings or tasks found).")
			return
		}

		printSymbols(symbols, 0)
	},
}

func printSymbols(symbols []protocol.DocumentSymbol, depth int) {
	for _, s := range symbols {
		marker := strings.Repeat("#", depth+1)
		if depth > 5 {
			marker = strings.Repeat("  ", depth-5) + "#"
		}
		if s.Kind == protocol.SymbolKindEvent {
			box := "[ ]"
			if s.Deprecated != nil && *s.Deprecated {
				box = "[x]"
			}
			fmt.Printf("%s%s %s\n", strings.Repeat("  ", depth), box, s.Name)
		} else {
			fmt.Printf("%s%s %s\n", strings.Repeat("  ", depth), marker, s.Name)
		}
		if len(s.Children) > 0 {
			printSymbols(s.Children, depth+1)
		}
	}
}

func init() {
	Lsp.AddCommand(&lspOutline)
}
