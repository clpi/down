package shell

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	Shell = cobra.Command{
		Use:     "shell <command>",
		Aliases: []string{"shell", "sh", "repl", "re"},
		Long:    "shell",
		Short:   "s",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("shell")
		},
	}
)
