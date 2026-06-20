package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var logRoot string
var logLimit int

var Log = cobra.Command{
	Use:     "log",
	Aliases: []string{"lg", "track"},
	Short:   "Show recently modified workspace files",
	Version: lsp.Version,
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(logRoot)
		files, err := wsutil.WalkMarkdown(root, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "log: %v\n", err)
			os.Exit(1)
		}
		type entry struct {
			path string
			mod  time.Time
		}
		var entries []entry
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			entries = append(entries, entry{f, info.ModTime()})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].mod.After(entries[j].mod) })
		limit := logLimit
		if limit <= 0 {
			limit = 20
		}
		if limit > len(entries) {
			limit = len(entries)
		}
		for i := 0; i < limit; i++ {
			rel, _ := filepath.Rel(root, entries[i].path)
			fmt.Printf("%s  %s\n", entries[i].mod.Format("2006-01-02 15:04"), strings.ReplaceAll(rel, string(os.PathSeparator), "/"))
		}
		fmt.Printf("\n%d recent file(s) shown.\n", limit)
	},
}

func init() {
	Log.Flags().StringVar(&logRoot, "root", "", "Workspace root")
	Log.Flags().IntVarP(&logLimit, "limit", "l", 20, "Number of entries")
}
