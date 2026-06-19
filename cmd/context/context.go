package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	ctxOutput  string
	ctxFormat  string
	ctxNoCompact bool
	ctxNoMemory  bool
	ctxPrompt    string
)

type ContextData struct {
	Project   string
	Structure []string
	SkillsMD  string
	CompactXML string
	Memory    []string
	Prompt    string
}

func findDownDir(root string) string {
	for dir := root; dir != "" && dir != "/" && filepath.Dir(dir) != dir; {
		dd := filepath.Join(dir, ".down")
		if info, err := os.Stat(dd); err == nil && info.IsDir() {
			return dd
		}
		parent := filepath.Dir(dir)
		if parent == dir { break }
		dir = parent
	}
	return ""
}

func detectStructure(root string, depth int) []string {
	if depth > 3 { return nil }
	var lines []string
	entries, err := os.ReadDir(root)
	if err != nil { return nil }
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != ".github" && name != ".down" {
			continue
		}
		if name == "node_modules" || name == ".git" {
			continue
		}
		indent := strings.Repeat("  ", depth)
		if e.IsDir() {
			lines = append(lines, indent+name+"/")
			sub := detectStructure(filepath.Join(root, name), depth+1)
			lines = append(lines, sub...)
		} else {
			lines = append(lines, indent+name)
		}
	}
	return lines
}

func detectLanguages(root string) []string {
	extMap := map[string]string{
		".lua": "Lua", ".go": "Go", ".js": "JavaScript", ".ts": "TypeScript",
		".py": "Python", ".rs": "Rust", ".rb": "Ruby", ".java": "Java",
		".html": "HTML", ".css": "CSS", ".scss": "SCSS", ".md": "Markdown",
		".json": "JSON", ".yaml": "YAML", ".yml": "YAML", ".toml": "TOML",
	}
	seen := make(map[string]bool)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if info.IsDir() {
			n := filepath.Base(path)
			if n == ".git" || n == "node_modules" || n == ".down" {
				return filepath.SkipDir
			}
			return nil
		}
		if lang, ok := extMap[strings.ToLower(filepath.Ext(path))]; ok {
			seen[lang] = true
		}
		return nil
	})
	var langs []string
	for l := range seen { langs = append(langs, l) }
	return langs
}

func detectEntryPoints(root string) []string {
	patterns := []string{"main.lua", "init.lua", "main.go", "index.js", "index.ts", "main.py"}
	var entries []string
	for _, p := range patterns {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			entries = append(entries, p)
		}
	}
	return entries
}

func detectPlatform(root string) []string {
	indicators := map[string]string{
		"go.mod": "Go", "package.json": "npm", "Cargo.toml": "Rust",
		"requirements.txt": "pip", "Justfile": "Just", "Makefile": "Make",
	}
	var deps []string
	for file, mgr := range indicators {
		if _, err := os.Stat(filepath.Join(root, file)); err == nil {
			deps = append(deps, fmt.Sprintf("- `%s` (%s)", file, mgr))
		}
	}
	return deps
}

func buildSkillsSection(root string) string {
	var b strings.Builder
	name := filepath.Base(root)

	fmt.Fprintf(&b, "## Project: %s\n\n", name)
	langs := detectLanguages(root)
	if len(langs) > 0 {
		fmt.Fprintf(&b, "**Languages:** %s\n\n", strings.Join(langs, ", "))
	}

	entries := detectEntryPoints(root)
	if len(entries) > 0 {
		fmt.Fprintf(&b, "**Entry points:** %s\n\n", strings.Join(entries, ", "))
	}

	deps := detectPlatform(root)
	if len(deps) > 0 {
		fmt.Fprintf(&b, "**Dependencies:**\n%s\n\n", strings.Join(deps, "\n"))
	}

	fmt.Fprintf(&b, "**Structure:**\n```\n")
	for _, s := range detectStructure(root, 0) {
		fmt.Fprintf(&b, "%s\n", s)
	}
	fmt.Fprintf(&b, "```\n")

	return b.String()
}

var Context = cobra.Command{
	Use:     "context [directory]",
	Aliases: []string{"ctx"},
	Short:   "Generate comprehensive AI project context",
	Long: `Generate a complete project context document for AI agents.

Combines project structure, language analysis, dependency information,
and optionally compact codebase and memory entries into a single
comprehensive context file ready for AI consumption.`,
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		var b strings.Builder
		name := filepath.Base(root)

		fmt.Fprintf(&b, "# %s — AI Context\n\n", name)
		fmt.Fprintf(&b, "> Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

		// Skills section
		fmt.Fprintf(&b, "%s\n", buildSkillsSection(root))

		// Add prompt if specified
		if ctxPrompt != "" {
			fmt.Fprintf(&b, "## Task\n\n%s\n\n", ctxPrompt)
		} else {
			fmt.Fprintf(&b, "## Task\n\n<!-- Describe what you want the AI to do -->\n\n")
		}

		// Output
		outPath := ctxOutput
		if outPath == "" {
			outPath = filepath.Join(root, ".down", "context.md")
		}
		os.MkdirAll(filepath.Dir(outPath), 0755)
		if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Context written to %s\n", outPath)
	},
}

func init() {
	Context.Flags().StringVarP(&ctxOutput, "output", "o", "", "Output path (default: .down/context.md)")
	Context.Flags().StringVarP(&ctxFormat, "format", "f", "markdown", "Output format: markdown")
	Context.Flags().BoolVar(&ctxNoCompact, "no-compact", false, "Omit compacted codebase")
	Context.Flags().BoolVar(&ctxNoMemory, "no-memory", false, "Omit memory entries")
	Context.Flags().StringVarP(&ctxPrompt, "prompt", "p", "", "Task prompt for the AI")
}
