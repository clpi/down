package log

import (
	"log"

	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var (
	Log = cobra.Command{
		Use:     "log <command>",
		Aliases: []string{"lg", "track", "tr", "trk"},
		Long:    "log",
		Short:   "l",
		Example: "log",
		Version: lsp.Version,
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("log")
		},
	}
)
