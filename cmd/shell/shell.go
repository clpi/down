package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var Shell = cobra.Command{
	Use:     "shell",
	Aliases: []string{"sh", "repl", "re"},
	Short:   "Interactive down command shell",
	Long:    "Run down subcommands interactively. Type 'exit' or Ctrl-D to quit.",
	Run: func(cmd *cobra.Command, args []string) {
		exe, err := os.Executable()
		if err != nil {
			exe = "down"
		}
		fmt.Println("down shell — type a subcommand (e.g. `note today`, `lsp tags`). exit to quit.")
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("down> ")
			if !scanner.Scan() {
				fmt.Println()
				return
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if line == "exit" || line == "quit" {
				return
			}
			parts := strings.Fields(line)
			c := exec.Command(exe, parts...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin
			if err := c.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		}
	},
}
