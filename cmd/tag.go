package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	Tag = cobra.Command{
		Use:     "tag <command>",
		Aliases: []string{},
		Long:    "tag",
		Short:   "tg",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("tag")
		},
	}
)
