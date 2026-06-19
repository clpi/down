package note

import (
	"log"

	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
	_ "path/filepath"
)

var (
	Note = cobra.Command{
		Use:     "note <command>",
		Aliases: []string{"note", "journal", "nt"},
		Long:    "note",
		Short:   "n",
		Example: "note today",
		Version: lsp.Version,
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("note")
		},
	}
)

func Notes() {
}
