package remove

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func findDownDir(root string) string {
	for dir := root; dir != "/" && dir != ""; {
		downDir := filepath.Join(dir, ".down")
		if info, err := os.Stat(downDir); err == nil && info.IsDir() {
			return downDir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

var Remove = cobra.Command{
	Use:     "ignore <pattern> [pattern...]",
	Aliases: []string{"ign"},
	Short:   "Append patterns to .downignore",
	Long:    "Add patterns to the nearest .down/.downignore file to exclude files from compact and add commands.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Fprintln(os.Stderr, "Error: no .down/ directory found. Run `down init` first.")
			os.Exit(1)
		}
		ignorePath := filepath.Join(downDir, ".downignore")
		f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot open %s: %v\n", ignorePath, err)
			os.Exit(1)
		}
		defer f.Close()
		for _, pattern := range args {
			fmt.Fprintf(f, "%s\n", pattern)
			fmt.Printf("Added to .downignore: %s\n", pattern)
		}
	},
}
