package new

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	New = cobra.Command{
		Use:     "new <command>",
		Aliases: []string{"create", "c"},
		Long:    "new",
		Hidden:                true,
		Short:   "n",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("new")
		},
	}
)
