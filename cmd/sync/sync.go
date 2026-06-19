package sync

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	Sync = cobra.Command{
		Use:     "sync <command>",
		Aliases: []string{"sy", "remote", "rem", "syn", "sc"},
		Long:    "sync",
		Short:   "sy",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("sync")
		},
	}
)
