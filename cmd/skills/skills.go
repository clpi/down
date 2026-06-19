package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	output     string
	noArch     bool
	noDeps     bool
	noEntries  bool
	noConv     bool
)

type depFile struct {
	file    string
	manager string
}

func detectPlatform(root string) []string {
	indicators := map[string]string{
		"package.json":       "npm",
		"package-lock.json":  "npm",
		"yarn.lock":          "yarn",
		"pnpm-lock.yaml":     "pnpm",
		"go.mod":             "Go",
		"go.sum":             "Go",
		"Cargo.toml":         "Rust/Cargo",
		"Cargo.lock":         "Rust/Cargo",
		"requirements.txt":   "pip",
		"pyproject.toml":     "Python",
		"setup.py":           "Python",
		"Gemfile":            "Ruby/Bundler",
		"mix.exs":            "Elixir/Mix",
		"build.gradle":       "Gradle",
		"pom.xml":            "Maven",
		"composer.json":      "PHP/Composer",
		"Justfile":           "Just",
		"Makefile":           "Make",
		"CMakeLists.txt":     "CMake",
		"stylua.toml":        "StyLua",
		"selene.toml":        "Selene",
		".luarc.json":        "LuaLS",
		"down-scm-1.rockspec": "LuaRocks",
	}

	var deps []depFile
	for file, mgr := range indicators {
		if _, err := os.Stat(filepath.Join(root, file)); err == nil {
			deps = append(deps, depFile{file: file, manager: mgr})
		}
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].file < deps[j].file })

	var lines []string
	for _, d := range deps {
		lines = append(lines, fmt.Sprintf("- `%s` (%s)", d.file, d.manager))
	}
	return lines
}

func detectLanguages(root string) []string {
	extMap := map[string]string{
		".lua": "Lua", ".go": "Go", ".js": "JavaScript", ".ts": "TypeScript",
		".jsx": "React JSX", ".tsx": "React TSX", ".py": "Python", ".rs": "Rust",
		".rb": "Ruby", ".java": "Java", ".c": "C", ".cpp": "C++", ".h": "C/C++ Header",
		".html": "HTML", ".css": "CSS", ".scss": "SCSS", ".md": "Markdown",
		".json": "JSON", ".yaml": "YAML", ".yml": "YAML", ".toml": "TOML",
		".sh": "Shell", ".bash": "Bash", ".zsh": "Zsh", ".vim": "Vimscript",
		".sql": "SQL", ".graphql": "GraphQL",
	}

	seen := make(map[string]bool)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == ".down" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := extMap[ext]; ok {
			seen[lang] = true
		}
		return nil
	})

	var langs []string
	for lang := range seen {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

func detectStructure(root string, depth int) []string {
	if depth > 3 {
		return nil
	}
	var lines []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != ".github" {
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

func detectConventions(root string) []string {
	dirs := map[string]string{
		"lua/":               "Lua source in lua/",
		"src/":               "Source in src/",
		"lib/":               "Library code in lib/",
		"test/":              "Tests in test/",
		"tests/":             "Tests in tests/",
		"spec/":              "Specs in spec/",
		"scripts/":           "Scripts in scripts/",
		"docs/":              "Documentation in docs/",
		"ext/":               "External deps in ext/",
		"queries/":           "Treesitter queries in queries/",
		"plugin/":            "Neovim plugin entry",
		".github/workflows/": "CI/CD via GitHub Actions",
		"book/":              "mdBook documentation",
	}
	var conv []string
	for dir, desc := range dirs {
		if info, err := os.Stat(filepath.Join(root, dir)); err == nil && info.IsDir() {
			conv = append(conv, desc)
		}
	}
	return conv
}

func detectEntryPoints(root string) []string {
	patterns := []string{
		"main.lua", "init.lua", "main.go", "index.js", "index.ts",
		"main.py", "__init__.py", "main.rs", "lib.rs", "main.rb",
	}
	var entries []string
	for _, p := range patterns {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			entries = append(entries, p)
		}
	}
	return entries
}

var Skills = cobra.Command{
	Use:     "skills [directory]",
	Aliases: []string{"sk"},
	Short:   "Generate a project SKILL.md for AI agents",
	Long:    "Analyze a project and generate a SKILL.md file describing its structure, languages, dependencies, entry points, and conventions.",
	Run: func(cmd *cobra.Command, args []string) {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}
		name := filepath.Base(root)

		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n\n", name)
		fmt.Fprintf(&b, "## Project Overview\n\n<!-- Brief description of what this project does -->\n\n")

		langs := detectLanguages(root)
		if len(langs) > 0 {
			fmt.Fprintf(&b, "**Languages:** %s\n\n", strings.Join(langs, ", "))
		}

		if !noEntries {
			entries := detectEntryPoints(root)
			if len(entries) > 0 {
				fmt.Fprintf(&b, "## Entry Points\n\n")
				for _, e := range entries {
					fmt.Fprintf(&b, "- `%s`\n", e)
				}
				fmt.Fprintf(&b, "\n")
			}
		}

		if !noDeps {
			deps := detectPlatform(root)
			if len(deps) > 0 {
				fmt.Fprintf(&b, "## Dependencies\n\n")
				for _, d := range deps {
					fmt.Fprintf(&b, "%s\n", d)
				}
				fmt.Fprintf(&b, "\n")
			}
		}

		if !noArch {
			fmt.Fprintf(&b, "## Project Structure\n\n```\n")
			structure := detectStructure(root, 0)
			for _, line := range structure {
				fmt.Fprintf(&b, "%s\n", line)
			}
			fmt.Fprintf(&b, "```\n\n")
		}

		if !noConv {
			conventions := detectConventions(root)
			if len(conventions) > 0 {
				fmt.Fprintf(&b, "## Conventions\n\n")
				for _, c := range conventions {
					fmt.Fprintf(&b, "- %s\n", c)
				}
				fmt.Fprintf(&b, "\n")
			}
		}

		fmt.Fprintf(&b, "## Key Modules\n\n<!-- Document important modules, their responsibilities, and how they relate -->\n\n")
		fmt.Fprintf(&b, "## Commands\n\n<!-- Common development commands -->\n\n```bash\n# Build\n# Test\n# Lint\n```\n")

		if output != "" {
			if err := os.WriteFile(output, []byte(b.String()), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Wrote %s\n", output)
		} else {
			fmt.Print(b.String())
		}
	},
}

func init() {
	Skills.Flags().StringVarP(&output, "output", "o", "", "Output path (default: stdout)")
	Skills.Flags().BoolVar(&noArch, "no-arch", false, "Skip architecture section")
	Skills.Flags().BoolVar(&noDeps, "no-deps", false, "Skip dependencies section")
	Skills.Flags().BoolVar(&noEntries, "no-entries", false, "Skip entry points section")
	Skills.Flags().BoolVar(&noConv, "no-conventions", false, "Skip conventions section")
}
