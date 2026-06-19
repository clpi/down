package export

import (
	"log"

	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var (
	Export = cobra.Command{
		Use:     "export <command>",
		Aliases: []string{"exp", "ex", "pub", "publish"},
		Long:    "export",
		Version: lsp.Version,
		Short:   "e",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("export")
		},
	}
)
