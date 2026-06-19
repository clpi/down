package delete

import (
	"log"

	dfs "github.com/clpi/down/core/fs"
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

func RmWorkspace(name string) {
	dfs.Workspace(name)
}

var (
	Delete = cobra.Command{
		Use:     "del <command>",
		Aliases: []string{"del", "d", "rm", "remove"},
		Long:    "delete",
		Version: lsp.Version,
		Short:   "d",
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			default:
				RmWorkspace(args[0])
			case "workspace":
				log.Println("workspace")
				RmWorkspace(args[1])
			}
			log.Println("delete")
		},
	}
)
