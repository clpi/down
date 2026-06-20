package snippet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var snippetRoot string

var Snippet = cobra.Command{
	Use:     "snippet",
	Aliases: []string{"snip", "sn"},
	Short:   "List text snippets",
}

var snippetList = cobra.Command{
	Use:   "list [query]",
	Short: "List snippets in .down/snippets/",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(snippetRoot)
		dir := filepath.Join(root, ".down", "snippets")
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Println("No snippets directory (.down/snippets/).")
			return
		}
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(args[0])
		}
		count := 0
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if query != "" && !strings.Contains(strings.ToLower(name), query) {
				continue
			}
			fmt.Println(name)
			count++
		}
		fmt.Printf("\n%d snippet(s).\n", count)
	},
}

var snippetShow = cobra.Command{
	Use:   "show <name>",
	Short: "Show snippet contents",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(snippetRoot)
		path := filepath.Join(root, ".down", "snippets", args[0])
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "snippet not found: %s\n", args[0])
			os.Exit(1)
		}
		fmt.Print(string(data))
	},
}

func init() {
	Snippet.PersistentFlags().StringVar(&snippetRoot, "root", "", "Workspace root")
	Snippet.AddCommand(&snippetList)
	Snippet.AddCommand(&snippetShow)
	Snippet.Run = snippetList.Run
}
