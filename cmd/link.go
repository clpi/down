package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	Link = cobra.Command{
		Use:     "link <command>",
		Aliases: []string{"ln"},
		Long:    "link",
		Short:   "l",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("link")
		},
	}
)
