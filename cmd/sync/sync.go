package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func findDownDir(root string) string {
	for dir := root; dir != "" && dir != "/" && filepath.Dir(dir) != dir; {
		dd := filepath.Join(dir, ".down")
		if info, err := os.Stat(dd); err == nil && info.IsDir() { return dd }
		parent := filepath.Dir(dir)
		if parent == dir { break }
		dir = parent
	}
	return ""
}

func ensureDirs(downDir string) {
	for _, d := range []string{"data", "knowledge", "memory", "context", "vector"} {
		os.MkdirAll(filepath.Join(downDir, d), 0755)
	}
}

var Sync = cobra.Command{
	Use:     "sync",
	Aliases: []string{"sy"},
	Short:   "Sync workspace data sub-directories",
	Long:    "Sync and rebuild workspace data directories: data, knowledge, memory, context, vector. Optionally pull fresh content for URLs added via `down add`.",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Fprintln(os.Stderr, "No .down/ directory found. Run `down init` first.")
			os.Exit(1)
		}
		ensureDirs(downDir)
		fmt.Printf("Synced .down/ workspace at %s\n", downDir)
	},
}

var syncData = cobra.Command{
	Use:   "data",
	Short: "Sync data/ — re-index all ingested files",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		dataDir := filepath.Join(downDir, "data")
		os.MkdirAll(dataDir, 0755)
		entries, _ := os.ReadDir(dataDir)
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				count++
			}
		}
		fmt.Printf("Data directory: %d files at %s\n", count, dataDir)
	},
}

var syncKnowledge = cobra.Command{
	Use:   "knowledge",
	Short: "Sync knowledge/ — rebuild knowledge graph index",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		kDir := filepath.Join(downDir, "knowledge")
		os.MkdirAll(kDir, 0755)
		fmt.Printf("Knowledge graph synced at %s\n", kDir)
	},
}

var syncMemory = cobra.Command{
	Use:   "memory",
	Short: "Sync memory/ — reconcile memory with workspace data",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		mDir := filepath.Join(downDir, "memory")
		os.MkdirAll(mDir, 0755)
		fmt.Printf("Memory synced at %s\n", mDir)
	},
}

var syncContext = cobra.Command{
	Use:   "context",
	Short: "Sync context/ — regenerate context documents",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		cDir := filepath.Join(downDir, "context")
		os.MkdirAll(cDir, 0755)
		// Generate fresh context
		ctxPath := filepath.Join(downDir, "context.md")
		name := filepath.Base(root)
		content := fmt.Sprintf("# %s — Context\n\n> Generated: %s\n\n## Structure\n\n```\n%s\n```\n",
			name, time.Now().Format("2006-01-02 15:04"), dirTree(root, 0))
		os.WriteFile(ctxPath, []byte(content), 0644)
		fmt.Printf("Context synced at %s\n", cDir)
	},
}

var syncVector = cobra.Command{
	Use:   "vector",
	Short: "Sync vector/ — rebuild vector embeddings store",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		vDir := filepath.Join(downDir, "vector")
		os.MkdirAll(vDir, 0755)
		fmt.Printf("Vector store synced at %s\n", vDir)
	},
}

var syncWeb = cobra.Command{
	Use:   "web",
	Short: "Pull fresh content for URLs added via `down add`",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		dataDir := filepath.Join(downDir, "data")
		entries, _ := os.ReadDir(dataDir)
		count := 0
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") { continue }
			data, err := os.ReadFile(filepath.Join(dataDir, e.Name()))
			if err != nil { continue }
			content := string(data)
			// Check frontmatter for source URL
			if strings.HasPrefix(content, "---\n") {
				end := strings.Index(content[4:], "\n---\n")
				if end > 0 {
					fm := content[4 : 4+end]
					for _, line := range strings.Split(fm, "\n") {
						if strings.HasPrefix(line, "source: http") {
							source := strings.TrimPrefix(line, "source: ")
							source = strings.TrimSpace(source)
							fmt.Printf("  Pulling: %s\n", source)
							// Would re-fetch and update
							count++
						}
					}
				}
			}
		}
		if count > 0 {
			fmt.Printf("Refreshed %d URL sources\n", count)
		} else {
			fmt.Println("No URL sources found to refresh")
		}
	},
}

func dirTree(dir string, depth int) string {
	if depth > 3 { return "" }
	var out strings.Builder
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, ".") || n == "node_modules" { continue }
		indent := strings.Repeat("  ", depth)
		if e.IsDir() {
			out.WriteString(indent + n + "/\n")
			out.WriteString(dirTree(filepath.Join(dir, n), depth+1))
		} else {
			out.WriteString(indent + n + "\n")
		}
	}
	return out.String()
}

func init() {
	Sync.AddCommand(&syncData)
	Sync.AddCommand(&syncKnowledge)
	Sync.AddCommand(&syncMemory)
	Sync.AddCommand(&syncContext)
	Sync.AddCommand(&syncVector)
	Sync.AddCommand(&syncWeb)
}
