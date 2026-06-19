package find

import (
	"log"

	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var (
	Find = cobra.Command{
		Use:     "fd <command>",
		Aliases: []string{"fd", "search"},
		Long:    "find",
		Version: lsp.Version,
		Short:   "f",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("find")
		},
	}
)
