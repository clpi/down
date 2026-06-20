package open

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var openRoot string

var Open = cobra.Command{
	Use:     "open [query]",
	Aliases: []string{"o", "edit", "e", "goto"},
	Short:   "Fuzzy-find and open a workspace file",
	Long:    "Search workspace markdown files by name and open with $EDITOR (or print path).",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(openRoot)
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(strings.Join(args, " "))
		}

		files, err := wsutil.WalkMarkdown(root, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open: %v\n", err)
			os.Exit(1)
		}

		type match struct {
			path  string
			score int
		}
		var matches []match
		for _, path := range files {
			rel, _ := filepath.Rel(root, path)
			name := strings.ToLower(filepath.Base(rel))
			relLower := strings.ToLower(rel)
			score := 0
			if query == "" {
				score = 1 // show all
			} else if name == query {
				score = 100
			} else if strings.HasPrefix(name, query) {
				score = 80
			} else if strings.Contains(name, query) {
				score = 50
			} else if strings.Contains(relLower, query) {
				score = 30
			}
			if score > 0 {
				matches = append(matches, match{path: rel, score: score})
			}
		}

		if len(matches) == 0 {
			fmt.Printf("No files matching %q\n", query)
			return
		}

		// Sort by score
		for i := 0; i < len(matches); i++ {
			for j := i + 1; j < len(matches); j++ {
				if matches[j].score > matches[i].score || (matches[j].score == matches[i].score && matches[j].path < matches[i].path) {
					matches[i], matches[j] = matches[j], matches[i]
				}
			}
		}

		// If single best match, open it directly
		if len(matches) == 1 || (len(matches) > 1 && matches[0].score >= 80 && (len(matches) == 1 || matches[0].score > matches[1].score+20)) {
			fullPath := filepath.Join(root, matches[0].path)
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}
			c := exec.Command(editor, fullPath)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Run()
			return
		}

		// Show matches
		limit := 15
		if limit > len(matches) {
			limit = len(matches)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  %s\n", matches[i].path)
		}
		if len(matches) > limit {
			fmt.Printf("  ... and %d more\n", len(matches)-limit)
		}
		fmt.Printf("\n%d match(es). Use `down open <query>` to narrow.\n", len(matches))
	},
}

func init() {
	Open.Flags().StringVar(&openRoot, "root", "", "Workspace root")
}
