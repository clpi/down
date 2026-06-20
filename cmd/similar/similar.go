package similar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/clpi/down/lsp/ai"
	"github.com/spf13/cobra"
)

var similarRoot string

var Similar = cobra.Command{
	Use:     "similar <file>",
	Aliases: []string{"recommend", "related", "sim"},
	Short:   "Find semantically similar files to the given file",
	Long:    "Use embedding similarity to discover related documents in the workspace.",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(similarRoot)
		target := args[0]
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}

		targetData, err := os.ReadFile(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot read: %s\n", target)
			os.Exit(1)
		}
		targetText := string(targetData)

		// Load IDF model if available
		downDir := filepath.Join(root, ".down")
		emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())
		modelPath := filepath.Join(downDir, "vector", "model.json")
		if data, err := os.ReadFile(modelPath); err == nil {
			var model struct {
				IDF       map[string]float64 `json:"idf"`
				Dimension int                `json:"dimension"`
				DocCount  int                `json:"doc_count"`
			}
			if json.Unmarshal(data, &model) == nil {
				emb.LoadModel(modelPath)
			}
		}

		// Embed target
		ctx := context.Background()
		targetVecs, _ := emb.Embed(ctx, []string{targetText})
		if len(targetVecs) == 0 {
			fmt.Fprintln(os.Stderr, "Could not embed target file")
			os.Exit(1)
		}
		targetVec := targetVecs[0]

		// Walk all markdown files
		files, _ := wsutil.WalkMarkdown(root, true)

		// Train on corpus for IDF
		var corpus []string
		for _, path := range files {
			if data, err := os.ReadFile(path); err == nil {
				corpus = append(corpus, string(data))
			}
		}
		emb.Train(corpus)

		// Re-embed target with IDF
		targetVecs, _ = emb.Embed(ctx, []string{targetText})
		if len(targetVecs) == 0 {
			return
		}
		targetVec = targetVecs[0]

		type scored struct {
			path  string
			score float32
		}
		var results []scored

		for _, path := range files {
			if path == target {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			text := string(data)
			if len(text) > 2000 {
				text = text[:2000]
			}
			vecs, _ := emb.Embed(ctx, []string{text})
			if len(vecs) == 0 {
				continue
			}
			score := ai.CosineSimilarity(targetVec, vecs[0])
			if score > 0.15 {
				rel, _ := filepath.Rel(root, path)
				results = append(results, scored{path: rel, score: score})
			}
		}

		// Sort
		for i := 0; i < len(results); i++ {
			for j := i + 1; j < len(results); j++ {
				if results[j].score > results[i].score {
					results[i], results[j] = results[j], results[i]
				}
			}
		}

		rel, _ := filepath.Rel(root, target)
		fmt.Printf("Similar to %s:\n\n", rel)
		limit := 10
		if limit > len(results) {
			limit = len(results)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  [%.4f] %s\n", results[i].score, results[i].path)
		}
		if len(results) == 0 {
			fmt.Println("  No similar files found.")
		}
	},
}

func init() {
	Similar.Flags().StringVar(&similarRoot, "root", "", "Workspace root")
}
