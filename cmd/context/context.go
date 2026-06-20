package context

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clpi/down/cmd/compact"
	"github.com/clpi/down/cmd/memory"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

var (
	ctxOutput     string
	ctxFormat     string
	ctxNoCompact  bool
	ctxNoMemory   bool
	ctxNoVectors  bool
	ctxPrompt     string
	ctxVecQuery   string
	ctxVecLimit   int
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

		if !ctxNoCompact {
			if xml, err := compact.PackXMLString(root); err != nil {
				fmt.Fprintf(os.Stderr, "compact: %v\n", err)
			} else if xml != "" {
				fmt.Fprintf(&b, "## Codebase (compact)\n\n```xml\n%s\n```\n\n", xml)
			}
		}

		if !ctxNoMemory {
			entries, err := memory.ListEntries()
			if err != nil {
				fmt.Fprintf(os.Stderr, "memory: %v\n", err)
			} else if len(entries) > 0 {
				fmt.Fprintf(&b, "## Memory\n\n")
				for _, e := range entries {
					preview := strings.ReplaceAll(e.Value, "\n", " ")
					if len(preview) > 200 {
						preview = preview[:200] + "..."
					}
					fmt.Fprintf(&b, "- **%s**: %s\n", e.Key, preview)
				}
				fmt.Fprintf(&b, "\n")
			}
		}

		// Add prompt if specified
		if ctxPrompt != "" {
			fmt.Fprintf(&b, "## Task\n\n%s\n\n", ctxPrompt)
		} else {
			fmt.Fprintf(&b, "## Task\n\n<!-- Describe what you want the AI to do -->\n\n")
		}

		// Add vector similarity search results if query provided
		if !ctxNoVectors && ctxVecQuery != "" {
			fmt.Fprintf(&b, "## Vector Search Results (%s)\n\n", ctxVecQuery)
			// Try to load vector embeddings
			vDir := findVectorDir(root)
			if vDir != "" {
				entries, _ := os.ReadDir(vDir)
				var vecs []vectorEntry
				for _, e := range entries {
					if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
						continue
					}
					data, _ := os.ReadFile(filepath.Join(vDir, e.Name()))
					var ve vectorEntry
					if json.Unmarshal(data, &ve) == nil && len(ve.Vector) > 0 {
						vecs = append(vecs, ve)
					}
				}
				if len(vecs) > 0 {
					dim := len(vecs[0].Vector)
					qvec := embedQuery(ctxVecQuery, dim)
					var results []similarityResult
					for _, ve := range vecs {
						score := cosineSim(qvec, ve.Vector)
						if score > 0.1 {
							results = append(results, similarityResult{entry: ve, score: score})
						}
					}
					sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
					limit := ctxVecLimit
					if limit <= 0 || limit > len(results) {
						limit = len(results)
						if limit > 10 {
							limit = 10
						}
					}
					for i := 0; i < limit && i < len(results); i++ {
						r := results[i]
						preview := r.entry.Text
						if len(preview) > 200 {
							preview = preview[:200] + "..."
						}
						fmt.Fprintf(&b, "- (%.2f) **%s**: %s\n", r.score, r.entry.Source, preview)
					}
				}
			}
			fmt.Fprintf(&b, "\n")
		}

		// Add knowledge graph summary
		fmt.Fprintf(&b, "## Knowledge Graph\n\n")
		kbPath := filepath.Join(root, ".down", "knowledge.json")
		if _, err := os.Stat(kbPath); err == nil {
			data, _ := os.ReadFile(kbPath)
			var g knowledge.Graph
			if json.Unmarshal(data, &g) == nil {
				fmt.Fprintf(&b, "%s\n\n", g.Summary())
			}
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
	Context.Flags().BoolVar(&ctxNoVectors, "no-vectors", false, "Omit vector search results")
	Context.Flags().StringVarP(&ctxVecQuery, "vector-query", "q", "", "Query for vector similarity search")
	Context.Flags().IntVarP(&ctxVecLimit, "vector-limit", "l", 10, "Max vector search results")
	Context.Flags().StringVarP(&ctxPrompt, "prompt", "p", "", "Task prompt for the AI")
}

// Vector search helper types and functions

type vectorEntry struct {
	ID     string    `json:"id"`
	Vector []float64 `json:"vector"`
	Text   string    `json:"text"`
	Source string    `json:"source"`
}

type similarityResult struct {
	entry vectorEntry
	score float64
}

func findVectorDir(root string) string {
	for dir := root; dir != "" && dir != "/" && filepath.Dir(dir) != dir; {
		vd := filepath.Join(dir, ".down", "vector")
		if info, err := os.Stat(vd); err == nil && info.IsDir() {
			return vd
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(root, ".down", "vector")
}

// Simple hash-based embedding (shared with vector package)
func hashWord(word string) int {
	h := 0
	for _, c := range word {
		h = (h*31 + int(c)) % 1000000
	}
	return h
}

func embedQuery(text string, dim int) []float64 {
	tokens := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_')
	})
	vec := make([]float64, dim)
	for _, token := range tokens {
		if len(token) < 2 {
			continue
		}
		idx := hashWord(token) % dim
		vec[idx] += 1.0
	}
	// Normalize
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}

func cosineSim(a, b []float64) float64 {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < minLen; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	na, nb = math.Sqrt(na), math.Sqrt(nb)
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (na * nb)
}

// EmbeddingSimilarity finds similar embeddings and returns formatted results
// This can be called from knowledge or context packages
func EmbeddingSimilarity(query string, limit int) []vectorEntry {
	root, _ := os.Getwd()
	vDir := findVectorDir(root)
	entries, _ := os.ReadDir(vDir)

	var vecs []vectorEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(vDir, e.Name()))
		var ve vectorEntry
		if json.Unmarshal(data, &ve) == nil && len(ve.Vector) > 0 {
			vecs = append(vecs, ve)
		}
	}

	if len(vecs) == 0 {
		return nil
	}

	dim := len(vecs[0].Vector)
	qvec := embedQuery(query, dim)
	type result struct {
		entry vectorEntry
		score float64
	}
	var results []result
	for _, ve := range vecs {
		score := cosineSim(qvec, ve.Vector)
		if score > 0.1 {
			results = append(results, result{entry: ve, score: score})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })

	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}
	var out []vectorEntry
	for i := 0; i < limit; i++ {
		out = append(out, results[i].entry)
	}
	return out
}
