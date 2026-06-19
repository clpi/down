package lsp

import (
	"log"

	ls "github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var (
	Lsp = cobra.Command{
		Use:     "lsp <command>",
		Aliases: []string{"ls", "L"},
		Long:    "lsp",
		Short:   "l",
		Run: func(cmd *cobra.Command, args []string) {
			lsp, err := ls.NewServer()
			if err != nil {
				log.Fatal(err)
			}
			lsp.Server.RunStdio()
		},
		Example: "down lsp",
		Version: ls.Version,
	}
)
