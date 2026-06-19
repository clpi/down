package workspace

import (
	"log"

	dfs "github.com/clpi/down/core/fs"
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var (
	Workspace = cobra.Command{
		Use:     "workspace <command>",
		Example: "workspace <command>",
		Version: lsp.Version,
		Aliases: []string{"ws", "w"},
		Long:    "workspace",
		Short:   "w",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("workspace")
			if len(args) == 0 {
			} else if args[0] == "clear" {
				dfs.WorkspaceRmAll()
			} else if args[0] == "rm" {
				if len(args) > 1 {
					dfs.WorkspaceRm(args[1])
				}
			} else if args[0] == "init" {
				log.Println("init")
				InitWorkspace.Run(&InitWorkspace, args[1:len(args)-1])
			} else if args[0] == "delete" {
				log.Println("delete")
			}
		},
	}
)
