package find

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var (
	findRoot  string
	findFiles bool
	findLimit int
)

var Find = cobra.Command{
	Use:     "find <query>",
	Aliases: []string{"fd", "search", "f"},
	Short:   "Search workspace notes and files",
	Long:    "Search markdown files in the workspace by filename and content.",
	Version: lsp.Version,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.ToLower(strings.Join(args, " "))
		root := wsutil.ResolveRoot(findRoot)
		files, err := wsutil.WalkMarkdown(root, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "find: %v\n", err)
			os.Exit(1)
		}
		count := 0
		for _, path := range files {
			rel, _ := filepath.Rel(root, path)
			name := filepath.Base(path)
			if findFiles && !strings.Contains(strings.ToLower(name), query) && !strings.Contains(strings.ToLower(rel), query) {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			text := string(data)
			lower := strings.ToLower(text)
			if findFiles && !strings.Contains(strings.ToLower(name), query) && !strings.Contains(strings.ToLower(rel), query) && !strings.Contains(lower, query) {
				continue
			}
			if !findFiles && !strings.Contains(lower, query) && !strings.Contains(strings.ToLower(name), query) {
				continue
			}
			fmt.Printf("%s\n", rel)
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), query) {
					fmt.Printf("  %d: %s\n", i+1, strings.TrimSpace(line))
				}
			}
			count++
			if findLimit > 0 && count >= findLimit {
				break
			}
		}
		if count == 0 {
			fmt.Printf("No matches for %q in %s\n", query, root)
		} else {
			fmt.Printf("\n%d match(es).\n", count)
		}
	},
}

func init() {
	Find.Flags().StringVar(&findRoot, "root", "", "Workspace root")
	Find.Flags().BoolVar(&findFiles, "files-only", false, "Match filenames only")
	Find.Flags().IntVarP(&findLimit, "limit", "l", 0, "Max files to show (0 = unlimited)")
}
