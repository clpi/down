package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	Snippet = cobra.Command{
		Use:     "snippet <command>",
		Aliases: []string{"snippet", "snip", "snp", "sn", "sp", "spt", "sppt"},
		Long:    "snippet",
		Short:   "s",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("snippet")
		},
	}
)
