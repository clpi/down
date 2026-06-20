package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var templateRoot string

var Template = cobra.Command{
	Use:     "template",
	Aliases: []string{"tmpl", "tpl", "temp"},
	Short:   "List and apply note templates",
}

var templateList = cobra.Command{
	Use:   "list",
	Short: "List available templates",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(templateRoot)
		dirs := []string{
			filepath.Join(root, "note"),
			filepath.Join(root, ".down", "templates"),
			filepath.Join(root, "templates"),
		}
		count := 0
		for _, dir := range dirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				fmt.Println(filepath.Join(dir, e.Name()))
				count++
			}
		}
		if count == 0 {
			fmt.Println("No templates found.")
		}
	},
}

var templateApply = cobra.Command{
	Use:   "apply <template> [dest]",
	Short: "Apply a template to a new file",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(templateRoot)
		tmplPath := args[0]
		if !filepath.IsAbs(tmplPath) {
			for _, dir := range []string{filepath.Join(root, "note"), filepath.Join(root, "templates")} {
				candidate := filepath.Join(dir, tmplPath)
				if _, err := os.Stat(candidate); err == nil {
					tmplPath = candidate
					break
				}
			}
		}
		data, err := os.ReadFile(tmplPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template not found: %s\n", args[0])
			os.Exit(1)
		}
		content := string(data)
		content = strings.ReplaceAll(content, "{{date}}", time.Now().Format("2006-01-02"))
		content = strings.ReplaceAll(content, "{{time}}", time.Now().Format("15:04:05"))
		dest := ""
		if len(args) > 1 {
			dest = args[1]
		} else {
			dest = filepath.Join(root, "note", time.Now().Format("2006/01/02")+".md")
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(root, dest)
		}
		os.MkdirAll(filepath.Dir(dest), 0755)
		os.WriteFile(dest, []byte(content), 0644)
		fmt.Println(dest)
	},
}

func init() {
	Template.PersistentFlags().StringVar(&templateRoot, "root", "", "Workspace root")
	Template.AddCommand(&templateList)
	Template.AddCommand(&templateApply)
	Template.Run = templateList.Run
}
