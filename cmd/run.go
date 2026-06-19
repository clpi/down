package cmd

import (
	"github.com/clpi/down/cmd/lsp"
	"github.com/spf13/cobra"
)

var (
	Runc = cobra.Command{
		Use:     "run <command>",
		Aliases: []string{"exec"},
		Long:    "run",
		Short:   "r",
		Run:     lsp.Lsp.Run,
	}
)
