package list

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var listRoot string

var List = cobra.Command{
	Use:     "list [subdir]",
	Aliases: []string{"ls"},
	Short:   "List markdown files in the workspace",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(listRoot)
		if len(args) > 0 {
			root = filepath.Join(root, args[0])
		}
		files, err := wsutil.WalkMarkdown(root, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "list: %v\n", err)
			os.Exit(1)
		}
		sort.Strings(files)
		wsRoot := wsutil.ResolveRoot(listRoot)
		for _, f := range files {
			rel, _ := filepath.Rel(wsRoot, f)
			fmt.Println(strings.ReplaceAll(rel, string(os.PathSeparator), "/"))
		}
		fmt.Printf("\n%d file(s).\n", len(files))
	},
}

func init() {
	List.Flags().StringVar(&listRoot, "root", "", "Workspace root")
}
