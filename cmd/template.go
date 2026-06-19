package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

var (
	Template = cobra.Command{
		Use:     "template <command>",
		Aliases: []string{"template", "tmpl", "templ", "tpl", "temp", "tmp"},
		Long:    "template",
		Short:   "t",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("template")
		},
	}
)
