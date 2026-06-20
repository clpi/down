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

	// Create default templates
	createDefaultTemplates(downDir)
}

func createDefaultTemplates(downDir string) {
	tmplDir := filepath.Join(downDir, "templates")
	os.MkdirAll(tmplDir, 0755)

	defaults := map[string]string{
		"daily.md":   "---\ntype: daily\ncategory: journal\ndescription: Daily journal entry\n---\n\n# {{date}}\n\n## Morning\n\n## Work Log\n\n- \n\n## Evening\n\n## Gratitude\n\n- \n",
		"meeting.md": "---\ntype: meeting\ncategory: work\ndescription: Meeting notes with agenda and action items\n---\n\n# Meeting: {{title}}\n\n**Date:** {{date}}\n**Time:** {{time}}\n**Attendees:** \n\n## Agenda\n\n- \n\n## Notes\n\n## Action Items\n\n- [ ] \n",
		"project.md": "---\ntype: project\ncategory: work\ndescription: Project overview with goals and timeline\n---\n\n# Project: {{title}}\n\n**Status:** in-progress\n**Start:** {{date}}\n**Target:** \n\n## Goals\n\n- \n\n## Timeline\n\n| Phase | Status | Date |\n|-------|--------|------|\n| Planning | done | |\n| Execution | in-progress | |\n| Review | pending | |\n\n## Notes\n",
		"weekly.md":  "---\ntype: weekly\ncategory: journal\ndescription: Weekly review and planning\n---\n\n# Week {{iso_week}}\n\n**{{date}}**\n\n## Highlights\n\n- \n\n## Monday\n## Tuesday\n## Wednesday\n## Thursday\n## Friday\n\n## Next Week\n",
		"monthly.md": "---\ntype: monthly\ncategory: journal\ndescription: Monthly review\n---\n\n# {{month}} {{year}} Review\n\n## Highlights\n\n- \n\n## By Week\n\n### Week 1\n### Week 2\n### Week 3\n### Week 4\n\n## Stats\n",
	}

	created := 0
	for name, content := range defaults {
		path := filepath.Join(tmplDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.WriteFile(path, []byte(content), 0644)
			created++
		}
	}
	if created > 0 {
		fmt.Printf("  Created %d default templates in templates/\n", created)
	}
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
