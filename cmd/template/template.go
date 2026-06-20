package template

import (
	"fmt"
	"os"
	"path/filepath"

	tmpl "github.com/clpi/down/core/template"
	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var (
	tmplRoot   string
	tmplType   string
	tmplCat    string
	tmplDesc   string
	tmplVars   string
	tmplTitle  string
	tmplAuthor string
)

func templateDirs(root string) []string {
	return []string{
		filepath.Join(root, ".down", "templates"),
		filepath.Join(root, "templates"),
		filepath.Join(root, "note"),
	}
}

func newEngine(root string) *tmpl.Engine {
	return tmpl.NewEngine(templateDirs(root)...)
}

var Template = cobra.Command{
	Use:     "template",
	Aliases: []string{"tmpl", "tpl", "temp"},
	Short:   "Manage templates: list, show, create, apply, edit, delete",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		listTemplates(root)
	},
}

func listTemplates(root string) {
	engine := newEngine(root)
	templates := engine.Load()
	if len(templates) == 0 {
		fmt.Println("No templates found.")
		fmt.Println("\nCreate one with: down template create <name>")
		fmt.Println("Template dirs:")
		for _, d := range templateDirs(root) {
			fmt.Printf("  %s\n", d)
		}
		return
	}
	fmt.Printf("Templates (%d):\n\n", len(templates))
	for _, t := range templates {
		kind := t.Type
		if t.Category != "" {
			kind += "/" + t.Category
		}
		fmt.Printf("  %-20s  %s\n", t.Name, kind)
		if t.Description != "" {
			fmt.Printf("    %s\n", t.Description)
		}
	}
	fmt.Println()
	fmt.Println("Types:")
	for _, typ := range engine.Types() {
		fmt.Printf("  %s\n", typ)
	}
}

var templateList = cobra.Command{
	Use:   "list",
	Short: "List all available templates",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)
		templates := engine.Load()
		if len(templates) == 0 {
			fmt.Println("No templates found.")
			return
		}
		for _, t := range templates {
			fmt.Printf("%-20s %-10s %s\n", t.Name, t.Type, t.Description)
		}
	},
}

var templateShow = cobra.Command{
	Use:   "show <name>",
	Short: "Show template content with variables expanded",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)
		t := engine.Find(args[0])
		if t == nil {
			b := tmpl.Builtin(args[0])
			if b == nil {
				fmt.Fprintf(os.Stderr, "template %q not found\n", args[0])
				os.Exit(1)
			}
			t = b
		}
		extra := map[string]string{
			"title":  orDefault(tmplTitle, t.Name),
			"author": orDefault(tmplAuthor, ""),
		}
		content := engine.Expand(t, extra)
		fmt.Println(content)
	},
}

var templateApply = cobra.Command{
	Use:   "apply <name> [dest]",
	Short: "Apply template to create a new file",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)
		name := args[0]

		extra := map[string]string{
			"title":  orDefault(tmplTitle, name),
			"author": orDefault(tmplAuthor, ""),
		}

		var dest string
		if len(args) > 1 {
			dest = args[1]
		} else {
			dest = filepath.Join(root, "note", name+".md")
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(root, dest)
		}

		if err := engine.ApplyToFile(name, dest, extra); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(dest)
	},
}

var templateCreate = cobra.Command{
	Use:   "create <name>",
	Short: "Create a new template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)

		content := fmt.Sprintf("# %s\n\n", args[0])
		meta := map[string]string{
			"type":        orDefault(tmplType, "note"),
			"category":    tmplCat,
			"description": tmplDesc,
			"variables":   tmplVars,
		}

		path, err := engine.Create(args[0], content, meta)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created: %s\n", path)
	},
}

var templateDelete = cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)

		if err := engine.Delete(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted: %s\n", args[0])
	},
}

var templateValidate = cobra.Command{
	Use:   "validate <name>",
	Short: "Validate a template",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)

		t := engine.Find(args[0])
		if t == nil {
			fmt.Fprintf(os.Stderr, "template %q not found\n", args[0])
			os.Exit(1)
		}

		issues := engine.Validate(t)
		if len(issues) == 0 {
			fmt.Printf("Template %q is valid.\n", args[0])
			fmt.Printf("  Type: %s\n", t.Type)
			fmt.Printf("  Category: %s\n", t.Category)
			fmt.Printf("  Variables: %v\n", t.Variables)
		} else {
			fmt.Printf("Template %q has %d issue(s):\n", args[0], len(issues))
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}
		}
	},
}

var templateTypes = cobra.Command{
	Use:   "types",
	Short: "List template types/categories",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)

		fmt.Println("Types:")
		for _, t := range engine.Types() {
			fmt.Printf("  %s\n", t)
		}
		fmt.Println("\nCategories:")
		for _, c := range engine.Categories() {
			fmt.Printf("  %s\n", c)
		}
	},
}

var templateInit = cobra.Command{
	Use:   "init",
	Short: "Initialize template directory with defaults",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(tmplRoot)
		engine := newEngine(root)

		defaults := []struct {
			name, tmplType, cat, desc, content string
		}{
			{"daily", "daily", "journal", "Daily journal entry", `# {{date}}

## Morning

## Work Log

- 

## Evening

## Gratitude

- 
`},
			{"meeting", "meeting", "work", "Meeting notes with agenda and action items", `# Meeting: {{title}}

**Date:** {{date}}
**Time:** {{time}}
**Attendees:** 

## Agenda

- 

## Notes

## Action Items

- [ ] 
`},
			{"project", "project", "work", "Project overview with goals and timeline", `# Project: {{title}}

**Status:** in-progress
**Start:** {{date}}
**Target:** 

## Goals

- 

## Timeline

| Phase | Status | Date |
|-------|--------|------|
| Planning | done | |
| Execution | in-progress | |
| Review | pending | |

## Notes
`},
			{"weekly", "weekly", "journal", "Weekly review and planning", `# Week {{iso_year}}-W{{iso_week}}

**{{date}}**

## Highlights

- 

## Monday
## Tuesday
## Wednesday
## Thursday
## Friday

## Next Week
`},
			{"monthly", "monthly", "journal", "Monthly review", `# {{month}} {{year}} Review

## Highlights

- 

## By Week

### Week 1
### Week 2
### Week 3
### Week 4

## Stats
`},
		}

		created := 0
		for _, d := range defaults {
			meta := map[string]string{
				"type":        d.tmplType,
				"category":    d.cat,
				"description": d.desc,
			}
			_, err := engine.Create(d.name, d.content, meta)
			if err == nil {
				created++
				fmt.Printf("  + %s (%s/%s)\n", d.name, d.tmplType, d.cat)
			}
		}
		fmt.Printf("\nCreated %d default templates.\n", created)
	},
}

func orDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func init() {
	Template.PersistentFlags().StringVar(&tmplRoot, "root", "", "Workspace root")
	templateApply.Flags().StringVar(&tmplTitle, "title", "", "Title for template expansion")
	templateApply.Flags().StringVar(&tmplAuthor, "author", "", "Author for template expansion")
	templateShow.Flags().StringVar(&tmplTitle, "title", "", "Title for template expansion")
	templateShow.Flags().StringVar(&tmplAuthor, "author", "", "Author for template expansion")
	templateCreate.Flags().StringVar(&tmplType, "type", "note", "Template type")
	templateCreate.Flags().StringVar(&tmplCat, "category", "", "Template category")
	templateCreate.Flags().StringVar(&tmplDesc, "desc", "", "Template description")
	templateCreate.Flags().StringVar(&tmplVars, "vars", "", "Template variables (comma-separated)")

	Template.AddCommand(&templateList)
	Template.AddCommand(&templateShow)
	Template.AddCommand(&templateApply)
	Template.AddCommand(&templateCreate)
	Template.AddCommand(&templateDelete)
	Template.AddCommand(&templateValidate)
	Template.AddCommand(&templateTypes)
	Template.AddCommand(&templateInit)
}
