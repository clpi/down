package vector

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clpi/down/lsp/ai"
	"github.com/clpi/down/lsp/knowledge"
	"github.com/spf13/cobra"
)

var (
	vecDim    int
	vecQuery  string
	vecLimit  int
	vecSource string
	vecAll    bool
	vecKBPath string
)

type VectorEntry struct {
	ID      string    `json:"id"`
	Vector  []float64 `json:"vector"`
	Text    string    `json:"text"`
	Source  string    `json:"source"`
	Created string    `json:"created"`
}

func dataDir() string {
	d := os.Getenv("XDG_DATA_HOME")
	if d == "" {
		home, _ := os.UserHomeDir()
		d = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(d, "down", "vector")
}

func workspaceVectorDir() string {
	cwd, _ := os.Getwd()
	for dir := cwd; dir != "" && dir != "/" && filepath.Dir(dir) != dir; {
		dd := filepath.Join(dir, ".down", "vector")
		if info, err := os.Stat(dd); err == nil && info.IsDir() { return dd }
		parent := filepath.Dir(dir)
		if parent == dir { break }
		dir = parent
	}
	vDir := filepath.Join(cwd, ".down", "vector")
	os.MkdirAll(vDir, 0755)
	return vDir
}

func hashWord(word string, dim int) int {
	h := 0
	for _, c := range word {
		h = (h*31 + int(c)) % 1000000
	}
	return h % dim
}

func embedLocal(text string, dim int) []float64 {
	tokens := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_')
	})
	if len(tokens) == 0 {
		v := make([]float64, dim)
		return v
	}
	vector := make([]float64, dim)
	for _, token := range tokens {
		if len(token) < 2 { continue }
		idx := hashWord(token, dim)
		vector[idx] += 1.0
	}
	// Normalize
	var norm float64
	for _, v := range vector { norm += v * v }
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vector { vector[i] /= norm }
	}
	return vector
}

func cosine(a, b []float64) float64 {
	minLen := len(a)
	if len(b) < minLen { minLen = len(b) }
	var dot, na, nb float64
	for i := 0; i < minLen; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	na, nb = math.Sqrt(na), math.Sqrt(nb)
	if na == 0 || nb == 0 { return 0 }
	return dot / (na * nb)
}

func loadEntries(dir string) []VectorEntry {
	entries, _ := os.ReadDir(dir)
	var vecs []VectorEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" { continue }
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		var v VectorEntry
		if json.Unmarshal(data, &v) == nil {
			vecs = append(vecs, v)
		}
	}
	return vecs
}

func saveEntry(dir string, v VectorEntry) error {
	data, _ := json.MarshalIndent(v, "", "  ")
	return os.WriteFile(filepath.Join(dir, v.ID+".json"), data, 0644)
}

var Vector = cobra.Command{
	Use:     "vector <command>",
	Aliases: []string{"vec", "v"},
	Short:   "Manage vector embeddings in .down/vector/",
	Long:    "Generate, store, search, and manage vector embeddings for semantic similarity in the workspace.",
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()
		entries := loadEntries(vDir)
		fmt.Printf("Vector store: %d embeddings at %s\n", len(entries), vDir)
	},
}

var vectorIndex = cobra.Command{
	Use:   "index <source> [text]",
	Short: "Generate and store an embedding",
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()
		source := ""
		var text string
		if len(args) >= 2 {
			source = args[0]
			text = args[1]
		} else if len(args) == 1 {
			source = args[0]
			// Read file content
			data, err := os.ReadFile(source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Cannot read: %s\n", source)
				return
			}
			text = string(data)
		} else {
			fmt.Println("Usage: down vector index <source> [text]")
			return
		}
		dim := vecDim
		if dim == 0 { dim = 384 }
		vec := embedLocal(text, dim)
		entry := VectorEntry{
			ID: fmt.Sprintf("vec_%d", len(loadEntries(vDir))+1),
			Vector: vec, Text: text, Source: source,
			Created: fmt.Sprintf("%d", os.Getenv("EPOCHSECONDS")),
		}
		if err := saveEntry(vDir, entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		fmt.Printf("Indexed: %s (%d-dimensional vector)\n", source, dim)
	},
}

var vectorSearch = cobra.Command{
	Use:   "search <query>",
	Short: "Search for similar embeddings",
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()
		entries := loadEntries(vDir)
		if len(entries) == 0 {
			fmt.Println("No embeddings found")
			return
		}
		query := ""
		if len(args) > 0 { query = strings.Join(args, " ") }
		if query == "" {
			fmt.Println("Usage: down vector search <query>")
			return
		}
		dim := 384
		if len(entries) > 0 && len(entries[0].Vector) > 0 {
			dim = len(entries[0].Vector)
		}
		qVec := embedLocal(query, dim)

		type result struct {
			entry VectorEntry
			score float64
		}
		var results []result
		for _, e := range entries {
			score := cosine(qVec, e.Vector)
			if score > 0.1 {
				results = append(results, result{e, score})
			}
		}
		sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
		limit := vecLimit
		if limit <= 0 { limit = 10 }
		if limit > len(results) { limit = len(results) }

		if len(results) == 0 {
			fmt.Println("No similar results found")
			return
		}
		fmt.Printf("Results for \"%s\" (%d total, showing top %d):\n\n", query, len(results), limit)
		for i := 0; i < limit; i++ {
			r := results[i]
			preview := r.entry.Text
			if len(preview) > 200 { preview = preview[:200] + "..." }
			fmt.Printf("## %s (%.2f)\n%s\n\n", r.entry.Source, r.score, preview)
		}
	},
}

var vectorList = cobra.Command{
	Use:   "list",
	Short: "List all embeddings",
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()
		entries := loadEntries(vDir)
		if len(entries) == 0 {
			fmt.Println("No embeddings")
			return
		}
		for _, e := range entries {
			fmt.Printf("- %s (%d dims) — %s\n", e.Source, len(e.Vector), e.Created)
		}
	},
}

var vectorDelete = cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an embedding",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 { fmt.Println("Usage: down vector delete <id>"); return }
		vDir := workspaceVectorDir()
		path := filepath.Join(vDir, args[0]+".json")
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(os.Stderr, "Not found: %s\n", args[0])
			return
		}
		fmt.Printf("Deleted: %s\n", args[0])
	},
}

// indexKnowledgeGraph embeddings for entities in the knowledge graph
var vectorIndexKB = cobra.Command{
	Use:   "index-kb",
	Short: "Generate embeddings for knowledge graph entities",
	Long: `Index all entities from the knowledge graph as vector embeddings.

This enables semantic search over people, concepts, tags, and other
extracted entities in the workspace.`,
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()

		// Find knowledge graph path
		kbPath := vecKBPath
		if kbPath == "" {
			home, _ := os.UserHomeDir()
			kbPath = filepath.Join(home, ".down", "knowledge.json")
		}

		data, err := os.ReadFile(kbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No knowledge graph found at %s\n", kbPath)
			os.Exit(1)
		}

		g := knowledge.NewGraph(kbPath)
		if err := json.Unmarshal(data, g); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing knowledge graph: %v\n", err)
			os.Exit(1)
		}

		dim := vecDim
		if dim == 0 { dim = 384 }

		count := 0
		for _, ent := range g.Entities {
			vec := embedLocal(ent.Name, dim)
			entry := VectorEntry{
				ID: fmt.Sprintf("entity_%s", ent.ID),
				Vector: vec,
				Text: ent.Name,
				Source: vecKBPath,
				Created: time.Now().Format("2006-01-02 15:04"),
			}
			if err := saveEntry(vDir, entry); err == nil {
				count++
			}
		}
		fmt.Printf("Indexed %d knowledge graph entities as embeddings\n", count)
	},
}

// indexAllFiles creates embeddings for all markdown files in the workspace
var vectorIndexAll = cobra.Command{
	Use:   "index-all [directory]",
	Short: "Generate embeddings for all markdown files in workspace",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		if len(args) > 0 {
			root = args[0]
		}
		vDir := workspaceVectorDir()

		dim := vecDim
		if dim == 0 { dim = 384 }

		count := 0
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(path), ".md") {
				return nil
			}
			if strings.HasPrefix(filepath.Base(path), ".") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			text := string(data)
			vec := embedLocal(text, dim)
			rel, _ := filepath.Rel(root, path)
			entry := VectorEntry{
				ID: fmt.Sprintf("file_%s", rel),
				Vector: vec,
				Text: text,
				Source: rel,
				Created: time.Now().Format(time.RFC3339),
			}
			saveEntry(vDir, entry)
			count++
			return nil
		})
		fmt.Printf("Indexed %d markdown files as embeddings\n", count)
	},
}

// updateVectorStore updates embeddings for changed/added files
func UpdateVectorStore(downDir, root string, added, modified []string) int {
	vDir := filepath.Join(downDir, "vector")
	os.MkdirAll(vDir, 0755)

	dim := vecDim
	if dim == 0 { dim = 384 }

	count := 0
	for _, path := range append(added, modified...) {
		vec := embedLocalFile(filepath.Join(root, path), dim)
		if vec == nil {
			continue
		}
		entry := VectorEntry{
			ID: fmt.Sprintf("file_%s", strings.ReplaceAll(path, "/", "_")),
			Vector: vec,
			Text: truncateText(loadTextFromFile(filepath.Join(root, path)), 500),
			Source: path,
			Created: time.Now().Format(time.RFC3339),
		}
		saveEntry(vDir, entry)
		count++
	}
	return count
}

func embedLocalFile(path string, dim int) []float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return embedLocal(string(data), dim)
}

func loadTextFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func init() {
	vectorIndex.Flags().IntVarP(&vecDim, "dim", "d", 384, "Vector dimension")
	vectorCluster.Flags().IntVarP(&clusterK, "k", "k", 5, "Number of clusters")
	vectorDedup.Flags().Float64VarP(&dedupThreshold, "threshold", "t", 0.92, "Similarity threshold for duplicates")
	Vector.AddCommand(&vectorIndex)
	Vector.AddCommand(&vectorSearch)
	Vector.AddCommand(&vectorList)
	Vector.AddCommand(&vectorDelete)
	Vector.AddCommand(&vectorIndexKB)
	Vector.AddCommand(&vectorIndexAll)
	Vector.AddCommand(&vectorCluster)
	Vector.AddCommand(&vectorDedup)
	Vector.AddCommand(&vectorStats)
	Vector.AddCommand(&vectorCompare)
	vectorIndexKB.Flags().StringVar(&vecKBPath, "knowledge", "", "Path to knowledge.json")
	vectorIndexAll.Flags().IntVarP(&vecDim, "dim", "d", 384, "Vector dimension")
}

var clusterK int
var dedupThreshold float64

var vectorCluster = cobra.Command{
	Use:   "cluster",
	Short: "Run k-means clustering on embeddings",
	Long:  "Group embeddings into k clusters and display topic labels for each.",
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()
		entries := loadEntries(vDir)
		if len(entries) == 0 {
			fmt.Println("No embeddings to cluster")
			return
		}

		emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())
		for _, e := range entries {
			emb.StoreEmbedding(e.Source, e.Text)
		}

		k := clusterK
		if k == 0 {
			k = 5
		}
		result := emb.Cluster(k, 20)

		clusterMembers := make(map[int][]string)
		for id, c := range result.Assignments {
			clusterMembers[c] = append(clusterMembers[c], id)
		}

		fmt.Printf("K-means clustering (k=%d):\n\n", k)
		for c := 0; c < k; c++ {
			label := strings.Join(result.Labels[c], ", ")
			fmt.Printf("=== Cluster %d: %s ===\n", c, label)
			for _, id := range clusterMembers[c] {
				fmt.Printf("  - %s\n", id)
			}
			fmt.Println()
		}
	},
}

var vectorDedup = cobra.Command{
	Use:   "dedup",
	Short: "Find near-duplicate embeddings",
	Long:  "Detect embedding pairs with similarity above the threshold.",
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()
		entries := loadEntries(vDir)
		if len(entries) == 0 {
			fmt.Println("No embeddings")
			return
		}

		emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())
		for _, e := range entries {
			emb.StoreEmbedding(e.Source, e.Text)
		}

		threshold := dedupThreshold
		if threshold == 0 {
			threshold = 0.92
		}
		dups := emb.Dedup(threshold)

		if len(dups) == 0 {
			fmt.Printf("No near-duplicates found (threshold=%.2f)\n", threshold)
			return
		}
		fmt.Printf("Near-duplicates (threshold=%.2f):\n", threshold)
		for _, d := range dups {
			fmt.Printf("  %.4f  %s  <->  %s\n", d.Score, d.ID1, d.ID2)
		}
	},
}

var vectorStats = cobra.Command{
	Use:   "stats",
	Short: "Show embedding quality statistics",
	Run: func(cmd *cobra.Command, args []string) {
		vDir := workspaceVectorDir()
		entries := loadEntries(vDir)

		emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())
		for _, e := range entries {
			emb.StoreEmbedding(e.Source, e.Text)
		}

		s := emb.Stats()
		fmt.Printf("Embedding Statistics:\n")
		fmt.Printf("  Count:      %d\n", s.Count)
		fmt.Printf("  Dimension:  %d\n", s.Dim)
		fmt.Printf("  Sparsity:   %.4f\n", s.Sparsity)
		fmt.Printf("  Mean norm:  %.4f\n", s.MeanNorm)
		fmt.Printf("  Coverage:   %.4f\n", s.Coverage)
	},
}

var vectorCompare = cobra.Command{
	Use:   "compare <text-a> <text-b>",
	Short: "Compare two texts for semantic similarity",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())
		score, shared := emb.Compare(args[0], args[1])
		fmt.Printf("Similarity: %.4f\n", score)
		fmt.Printf("Shared terms (%d): %s\n", len(shared), strings.Join(shared, ", "))
	},
}
