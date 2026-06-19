package workspace

import (
	"log"
	"os"

	dfs "github.com/clpi/down/core/fs"
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

func initWorkspace(args []string) {
	w := ""
	if len(args) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		w = wd
	} else {
		w = args[0]
	}
	dfs.Workspace(w)

	log.Println("workspace")
}

var (
	InitWorkspace = cobra.Command{
		Use:     "init <name> <path> [args] ...",
		Example: "init <name> <path> [args] ...",
		Args:    cobra.MinimumNArgs(0),
		Version: lsp.Version,
		Aliases: []string{"i", "wi"},
		Long:    "init",
		Short:   "i",
		PreRun: func(cmd *cobra.Command, args []string) {
		},
		PostRun: func(cmd *cobra.Command, args []string) {
		},
		Run: func(cmd *cobra.Command, args []string) {
			initWorkspace(args)
		},
	}
)
