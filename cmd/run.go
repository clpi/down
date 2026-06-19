package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	Runc = cobra.Command{
		Use:     "run <command>",
		Aliases: []string{"exec"},
		Long:    "run",
		Short:   "r",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("run")
		},
	}
)
