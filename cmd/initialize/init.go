package initialize

import (
	"log"
	"os"
	"path"

	dfs "github.com/clpi/down/core/fs"
	"github.com/spf13/cobra"
)

func Dir() string {
	d, e := os.Getwd()
	if e != nil {
		log.Println(e)
	}
	// log.Println(path.Base(d))
	b := path.Base(d)
	return b

}
func InitWorkspace(name string) {
	if name == "" {
		log.Println(Dir())
		dfs.Workspace(Dir())

	}
	dfs.Workspace(name)
}

var (
	Init = cobra.Command{
		Use:     "init <command>",
		Aliases: []string{"ini", "initialize", "create"},
		Long:    "init",
		Short:   "i",
		Run: func(cmd *cobra.Command, args []string) {
			l := len(args)
			if l == 0 {
				InitWorkspace("")
				return
			}
			switch args[0] {
			case "workspace":
				log.Println("workspace")
				InitWorkspace(args[1])
			default:
				InitWorkspace(args[0])

			}
			log.Println("init")
		},
	}
)
