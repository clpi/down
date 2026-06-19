package list

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	List = cobra.Command{
		Use:     "list <command>",
		Aliases: []string{"ls"},
		Long:    "list",
		Short:   "l",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("list")
		},
	}
)
