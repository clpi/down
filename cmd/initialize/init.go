package initialize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	initWiki  bool
	initName  string
	initTitle string
)

type DownConfig struct {
	Name    string            `json:"name"`
	Wiki    bool              `json:"wiki,omitempty"`
	Created string            `json:"created"`
	Meta    map[string]string `json:"meta,omitempty"`
}

func initWorkspace(root, name string, wiki bool) {
	downDir := filepath.Join(root, ".down")
	os.MkdirAll(downDir, 0755)

	for _, sub := range []string{"data", "knowledge", "memory", "context", "vector", "templates", "git"} {
		os.MkdirAll(filepath.Join(downDir, sub), 0755)
	}

	// Create .downignore
	ignorePath := filepath.Join(downDir, ".downignore")
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		os.WriteFile(ignorePath, []byte("# Files ignored by down compact/add\n.git/\n.svn/\nnode_modules/\n"), 0644)
	}

	// Create index.md (in root for wiki mode, .down/ for codebase mode)
	indexPath := filepath.Join(downDir, "index.md")
	if wiki {
		indexPath = filepath.Join(root, "index.md")
	}
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		desc := "codebase"
		content := "# Index\n\nWelcome to your down workspace.\n\n"
		if wiki {
			desc = "wiki"
			content = "# " + name + "\n\n> A knowledge wiki workspace.\n\n## Getting Started\n\n- Create notes with `down note today`\n- Use `down template apply daily` to start journaling\n- Use `down sync` to index everything\n- Use `down sync skills` for AI context\n"
		}
		os.WriteFile(indexPath, []byte(content), 0644)
		fmt.Printf("  Created index.md (%s mode)\n", desc)
	}

	// Create down.json config
	configPath := filepath.Join(downDir, "down.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DownConfig{
			Name:    name,
			Wiki:    wiki,
			Created: time.Now().Format("2006-01-02 15:04"),
		}
		data, _ := json.MarshalIndent(cfg, "", "  ")
		os.WriteFile(configPath, data, 0644)
		if wiki {
			fmt.Printf("  Created down.json (wiki=true)\n")
		} else {
			fmt.Printf("  Created down.json\n")
		}
	}

	fmt.Printf("Workspace initialized at %s\n", downDir)
}

var Init = cobra.Command{
	Use:     "init [directory]",
	Aliases: []string{"ini", "initialize", "create"},
	Short:   "Initialize a .down/ workspace",
	Long:    "Create a .down/ workspace directory with index.md, down.json, .downignore, and subdirectories for data, knowledge, memory, context, and vector.",
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 { root = args[0] }
		name := initName
		if name == "" { name = filepath.Base(root) }
		if initTitle == "" { initTitle = name }

		fmt.Printf("Initializing \"%s\" workspace...\n", name)
		initWorkspace(root, name, initWiki)
	},
}

func init() {
	Init.Flags().BoolVarP(&initWiki, "wiki", "w", false, "Initialize as a wiki/knowledge workspace (markdown-first)")
	Init.Flags().StringVarP(&initName, "name", "n", "", "Workspace name")
	Init.Flags().StringVarP(&initTitle, "title", "t", "", "Index page title")
}
