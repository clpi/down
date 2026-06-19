package vector

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	vecDim    int
	vecQuery  string
	vecLimit  int
	vecSource string
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

func init() {
	vectorIndex.Flags().IntVarP(&vecDim, "dim", "d", 384, "Vector dimension")
	Vector.AddCommand(&vectorIndex)
	Vector.AddCommand(&vectorSearch)
	Vector.AddCommand(&vectorList)
	Vector.AddCommand(&vectorDelete)
}
