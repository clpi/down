package new

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var (
	newRoot  string
	newDir   string
	newOpen  bool
)

var slugRe = regexp.MustCompile(`[^a-z0-9\-]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return slugRe.ReplaceAllString(s, "")
}

var New = cobra.Command{
	Use:     "new <title>",
	Aliases: []string{"create", "c"},
	Short:   "Create a new note page",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		title := strings.Join(args, " ")
		root := wsutil.ResolveRoot(newRoot)
		dir := newDir
		if dir == "" {
			dir = "."
		}
		name := slugify(title)
		if name == "" {
			name = time.Now().Format("20060102-150405")
		}
		path := filepath.Join(root, dir, name+".md")
		os.MkdirAll(filepath.Dir(path), 0755)
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(os.Stderr, "File already exists: %s\n", path)
			os.Exit(1)
		}
		content := fmt.Sprintf("# %s\n\n> Created %s\n\n", title, time.Now().Format("2006-01-02"))
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "new: %v\n", err)
			os.Exit(1)
		}
		if newOpen {
			fmt.Printf("open %s\n", path)
		} else {
			fmt.Println(path)
		}
	},
}

func init() {
	New.Flags().StringVar(&newRoot, "root", "", "Workspace root")
	New.Flags().StringVarP(&newDir, "dir", "d", "", "Subdirectory for new note")
	New.Flags().BoolVar(&newOpen, "open", false, "Print 'open <path>' for editor integration")
}
