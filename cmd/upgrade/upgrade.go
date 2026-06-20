package upgrade

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/clpi/down/lsp"
	"github.com/spf13/cobra"
)

var Upgrade = cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the down CLI binary",
	Long:  "Rebuild or reinstall the down CLI from source (requires Go).",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Current version: %s\n", lsp.Version)
		if _, err := exec.LookPath("go"); err != nil {
			fmt.Fprintln(os.Stderr, "Go is not installed. Download a release binary from GitHub releases.")
			os.Exit(1)
		}
		fmt.Println("Run: go install github.com/clpi/down@latest")
		fmt.Println("Or rebuild from ext/down in the plugin repo.")
	},
}
