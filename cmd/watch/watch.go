package watch

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var (
	watchRoot    string
	watchDebounce time.Duration
	watchOnce    bool
)

var Watch = cobra.Command{
	Use:     "watch [directory]",
	Aliases: []string{"w", "auto"},
	Short:   "Watch workspace files and auto-sync on changes",
	Long:    "Monitor markdown files in the workspace and run `down sync` automatically when changes are detected.",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(watchRoot)
		if len(args) > 0 {
			root = args[0]
		}

		downDir := ""
		for d := root; d != "" && d != "/"; {
			dd := filepath.Join(d, ".down")
			if info, err := os.Stat(dd); err == nil && info.IsDir() {
				downDir = dd
				break
			}
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
		if downDir == "" {
			fmt.Fprintln(os.Stderr, "No .down/ directory found. Run `down init` first.")
			os.Exit(1)
		}

		fmt.Printf("Watching: %s\n", root)
		fmt.Printf("Debounce: %v\n", watchDebounce)
		fmt.Println("Press Ctrl+C to stop.")
		fmt.Println()

		if watchOnce {
			fmt.Println("Running initial sync...")
			exec.Command("down", "sync").Run()
			return
		}

		// Track modification times of all .md files
		lastMod := make(map[string]time.Time)
		var pending bool
		var lastSync time.Time

		// Initial scan
		scanFiles(root, lastMod)

		fmt.Println("Waiting for changes...")
		fmt.Println()

		ticker := time.NewTicker(watchDebounce)
		defer ticker.Stop()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)

		for {
			select {
			case <-sigCh:
				fmt.Println("\nStopped.")
				return
			case <-ticker.C:
				changed := scanFiles(root, lastMod)
				if changed {
					pending = true
					lastSync = time.Now()
				}

				if pending && time.Since(lastSync) >= watchDebounce {
					pending = false
					fmt.Printf("[%s] Syncing...\n", time.Now().Format("15:04:05"))
					exec.Command("down", "sync").Run()
					scanFiles(root, lastMod)
				}
			}
		}
	},
}

func scanFiles(root string, lastMod map[string]time.Time) bool {
	changed := false
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && (info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == ".down") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		prev, exists := lastMod[path]
		if !exists || info.ModTime().After(prev) {
			changed = true
		}
		lastMod[path] = info.ModTime()
		return nil
	})
	return changed
}

func init() {
	Watch.Flags().StringVar(&watchRoot, "root", "", "Workspace root")
	Watch.Flags().DurationVarP(&watchDebounce, "debounce", "d", 3*time.Second, "Debounce duration before sync")
	Watch.Flags().BoolVarP(&watchOnce, "once", "1", false, "Sync once and exit")
}
