package find

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp"
	"github.com/clpi/down/lsp/ai"
	"github.com/spf13/cobra"
)

var (
	findRoot     string
	findFiles    bool
	findLimit    int
	findSemantic bool
)

var Find = cobra.Command{
	Use:     "find <query>",
	Aliases: []string{"fd", "search", "f"},
	Short:   "Search workspace notes and files",
	Long: `Search markdown files in the workspace by filename and content.
Use --semantic for embedding-based similarity search.`,
	Version: lsp.Version,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.ToLower(strings.Join(args, " "))
		root := wsutil.ResolveRoot(findRoot)

		if findSemantic {
			semanticFind(root, strings.Join(args, " "))
			return
		}

		files, err := wsutil.WalkMarkdown(root, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "find: %v\n", err)
			os.Exit(1)
		}
		count := 0
		for _, path := range files {
			rel, _ := filepath.Rel(root, path)
			name := filepath.Base(path)
			if findFiles && !strings.Contains(strings.ToLower(name), query) && !strings.Contains(strings.ToLower(rel), query) {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			text := string(data)
			lower := strings.ToLower(text)
			if findFiles && !strings.Contains(strings.ToLower(name), query) && !strings.Contains(strings.ToLower(rel), query) && !strings.Contains(lower, query) {
				continue
			}
			if !findFiles && !strings.Contains(lower, query) && !strings.Contains(strings.ToLower(name), query) {
				continue
			}
			fmt.Printf("%s\n", rel)
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), query) {
					fmt.Printf("  %d: %s\n", i+1, strings.TrimSpace(line))
				}
			}
			count++
			if findLimit > 0 && count >= findLimit {
				break
			}
		}
		if count == 0 {
			fmt.Printf("No matches for %q in %s\n", query, root)
		} else {
			fmt.Printf("\n%d match(es).\n", count)
		}
	},
}

func semanticFind(root, query string) {
	// Load vector model for IDF
	downDir := filepath.Join(root, ".down")
	emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())

	modelPath := filepath.Join(downDir, "vector", "model.json")
	if data, err := os.ReadFile(modelPath); err == nil {
		var model struct {
			IDF       map[string]float64 `json:"idf"`
			Dimension int                `json:"dimension"`
		}
		if json.Unmarshal(data, &model) == nil {
			emb.LoadModel(modelPath)
		}
	}

	// Collect all markdown files
	files, _ := wsutil.WalkMarkdown(root, true)

	// Train on corpus
	var corpus []string
	for _, path := range files {
		if data, err := os.ReadFile(path); err == nil {
			corpus = append(corpus, string(data))
		}
	}
	emb.Train(corpus)

	// Embed query
	queryVecs, _ := emb.Embed(nil, []string{query})
	if len(queryVecs) == 0 {
		fmt.Println("Could not embed query")
		return
	}
	queryVec := queryVecs[0]

	type scoredFile struct {
		path  string
		score float32
	}
	var results []scoredFile

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(data)
		if len(text) > 2000 {
			text = text[:2000]
		}
		vecs, _ := emb.Embed(nil, []string{text})
		if len(vecs) == 0 {
			continue
		}
		score := ai.CosineSimilarity(queryVec, vecs[0])
		if score > 0.1 {
			rel, _ := filepath.Rel(root, path)
			results = append(results, scoredFile{path: rel, score: score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) == 0 {
		fmt.Printf("No semantic matches for %q\n", query)
		return
	}

	limit := findLimit
	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}

	fmt.Printf("Semantic results for %q:\n\n", query)
	for i := 0; i < limit; i++ {
		r := results[i]
		fmt.Printf("  [%.4f] %s\n", r.score, r.path)
	}
	fmt.Printf("\n%d match(es).\n", len(results))
}

func init() {
	Find.Flags().StringVar(&findRoot, "root", "", "Workspace root")
	Find.Flags().BoolVar(&findFiles, "files-only", false, "Match filenames only")
	Find.Flags().IntVarP(&findLimit, "limit", "l", 0, "Max files to show (0 = unlimited)")
	Find.Flags().BoolVarP(&findSemantic, "semantic", "s", false, "Use semantic (embedding) search")
}
