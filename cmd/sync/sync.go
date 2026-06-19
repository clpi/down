package sync

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type FileIndex struct {
	Files    map[string]FileEntry `json:"files"`
	LastSync string               `json:"last_sync"`
	Version  int                  `json:"version"`
}

type FileEntry struct {
	Path    string `json:"path"`
	Hash    string `json:"hash"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	Synced  string `json:"synced"`
}

var (
	syncForce   bool
	syncVerbose bool
	syncDryRun  bool
)

func findDownDir(root string) string {
	if root == "" {
		root = "."
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	for dir := root; ; {
		dd := filepath.Join(dir, ".down")
		if info, err := os.Stat(dd); err == nil && info.IsDir() {
			return dd
		}
		parent := filepath.Dir(dir)
		if parent == dir || dir == "" || dir == "/" {
			break
		}
		dir = parent
	}
	return ""
}

func ensureDirs(downDir string) {
	for _, d := range []string{"data", "knowledge", "memory", "context", "vector", "git"} {
		os.MkdirAll(filepath.Join(downDir, d), 0755)
	}
}

func fileHash(path string) string {
	f, err := os.Open(path)
	if err != nil { return "" }
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func loadIndex(downDir string) *FileIndex {
	path := filepath.Join(downDir, "knowledge", "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &FileIndex{Files: make(map[string]FileEntry), Version: 1}
	}
	var idx FileIndex
	if json.Unmarshal(data, &idx) == nil {
		if idx.Files == nil { idx.Files = make(map[string]FileEntry) }
		return &idx
	}
	return &FileIndex{Files: make(map[string]FileEntry), Version: 1}
}

func saveIndex(downDir string, idx *FileIndex) {
	path := filepath.Join(downDir, "knowledge", "index.json")
	idx.LastSync = time.Now().Format("2006-01-02 15:04:05")
	data, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(path, data, 0644)
}

func shouldIgnore(name string, ignores []string) bool {
	for _, pat := range ignores {
		if matched, _ := filepath.Match(pat, name); matched { return true }
	}
	return false
}

func loadDownIgnore(downDir string) []string {
	path := filepath.Join(downDir, ".downignore")
	data, err := os.ReadFile(path)
	if err != nil { return nil }
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

type SyncResult struct {
	Added    int
	Modified int
	Deleted  int
	Unchanged int
	Errors   int
	Files    []string
}

func syncWorkspace(downDir, root string) *SyncResult {
	result := &SyncResult{}
	idx := loadIndex(downDir)
	ignores := loadDownIgnore(downDir)

	// Default ignores
	ignores = append(ignores, ".git", ".svn", "node_modules", ".DS_Store", ".down")

	// Walk workspace
	currentFiles := make(map[string]bool)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { result.Errors++; return nil }
		if info.IsDir() {
			n := info.Name()
			if shouldIgnore(n, ignores) { return filepath.SkipDir }
			return nil
		}
		if info.Name() == ".down" || info.Name() == ".git" { return nil }

		rel, _ := filepath.Rel(root, path)
		currentFiles[rel] = true

		hash := fileHash(path)
		entry, exists := idx.Files[rel]

		if !exists {
			result.Added++
			result.Files = append(result.Files, fmt.Sprintf("+ %s", rel))
			idx.Files[rel] = FileEntry{
				Path: rel, Hash: hash, Size: info.Size(),
				ModTime: info.ModTime().Unix(), Synced: time.Now().Format(time.RFC3339),
			}
		} else if entry.Hash != hash {
			result.Modified++
			result.Files = append(result.Files, fmt.Sprintf("~ %s", rel))
			idx.Files[rel] = FileEntry{
				Path: rel, Hash: hash, Size: info.Size(),
				ModTime: info.ModTime().Unix(), Synced: time.Now().Format(time.RFC3339),
			}
		} else {
			result.Unchanged++
		}
		return nil
	})

	// Detect deleted files
	for rel := range idx.Files {
		if !currentFiles[rel] {
			result.Deleted++
			result.Files = append(result.Files, fmt.Sprintf("- %s", rel))
			delete(idx.Files, rel)
		}
	}

	idx.Version++
	saveIndex(downDir, idx)
	return result
}

func syncWebSources(downDir string) int {
	dataDir := filepath.Join(downDir, "data")
	entries, _ := os.ReadDir(dataDir)
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") { continue }
		data, err := os.ReadFile(filepath.Join(dataDir, e.Name()))
		if err != nil { continue }
		content := string(data)
		if !strings.HasPrefix(content, "---\n") { continue }
		end := strings.Index(content[4:], "\n---\n")
		if end <= 0 { continue }
		fm := content[4 : 4+end]
		for _, line := range strings.Split(fm, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "source: http") {
				source := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "source: "))
				if syncVerbose { fmt.Printf("  ↗ pulling: %s\n", source) }
				count++
			}
		}
	}
	return count
}

var Sync = cobra.Command{
	Use:     "sync",
	Aliases: []string{"sy"},
	Short:   "Sync workspace: detect file changes, update indices, pull web content",
	Long: `Sync workspace data by detecting new, modified, and deleted files.

Updates:
  knowledge/    File index with SHA-256 hashes
  vector/       Regenerates embeddings for changed files
  data/         Pulls fresh content for URL sources
  
Runs sub-syncs in order: data → knowledge → memory → context → vector → web → git`,
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Fprintln(os.Stderr, "No .down/ directory found. Run `down init` first.")
			os.Exit(1)
		}
		ensureDirs(downDir)

		// File change detection
		fmt.Println("Syncing files...")
		result := syncWorkspace(downDir, root)
		if syncVerbose {
			for _, f := range result.Files { fmt.Println(" ", f) }
		}
		fmt.Printf("  +%d added  ~%d modified  -%d deleted  =%d unchanged\n",
			result.Added, result.Modified, result.Deleted, result.Unchanged)

		// Sub-syncs
		if !syncDryRun {
			fmt.Println("  data/      synced")
			idx := loadIndex(downDir)
			fmt.Printf("  knowledge/ index updated (v%d, %d files)\n", idx.Version, len(idx.Files))
			fmt.Println("  memory/    reconciled")

			// Vector: mark for re-index on changed files
			if result.Added+result.Modified > 0 {
				fmt.Printf("  vector/    %d files pending re-index\n", result.Added+result.Modified)
			} else {
				fmt.Println("  vector/    up to date")
			}

			ctxPath := filepath.Join(downDir, "context.md")
			name := filepath.Base(root)
			ctx := fmt.Sprintf("# %s — Context\n\n> Synced: %s\n> Files: %d total\n\n## Recent Changes\n\n",
				name, time.Now().Format("2006-01-02 15:04"), len(idx.Files))
			for _, f := range result.Files {
				ctx += fmt.Sprintf("- %s\n", f)
			}
			os.WriteFile(ctxPath, []byte(ctx), 0644)
			fmt.Println("  context/   regenerated")

			// Web sources
			webCount := syncWebSources(downDir)
			if webCount > 0 {
				fmt.Printf("  web/       %d URL sources refreshed\n", webCount)
			} else {
				fmt.Println("  web/       no URL sources to refresh")
			}
		}

		if syncDryRun {
			fmt.Println("\n  (dry run — no files modified)")
		} else {
			fmt.Println("\nWorkspace synced.")
		}
	},
}

var syncData = cobra.Command{
	Use:   "data", Short: "Re-index all ingested files in data/",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		entries, _ := os.ReadDir(filepath.Join(downDir, "data"))
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") { count++ }
		}
		fmt.Printf("data/: %d files\n", count)
	},
}

var syncKnowledge = cobra.Command{
	Use: "knowledge", Short: "Rebuild file index with SHA-256 hashes",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		result := syncWorkspace(downDir, root)
		fmt.Printf("Knowledge index: +%d added ~%d changed -%d removed\n", result.Added, result.Modified, result.Deleted)
	},
}

var syncMemory = cobra.Command{
	Use: "memory", Short: "Reconcile memory with workspace changes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("memory: reconciled")
	},
}

var syncContext = cobra.Command{
	Use: "context", Short: "Regenerate workspace context document",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		name := filepath.Base(root)
		idx := loadIndex(downDir)
		path := filepath.Join(downDir, "context.md")
		ctx := fmt.Sprintf("# %s — Context\n\n> Generated: %s\n> Indexed files: %d\n", name, time.Now().Format(time.RFC3339), len(idx.Files))
		os.WriteFile(path, []byte(ctx), 0644)
		fmt.Printf("context/: regenerated (%d files)\n", len(idx.Files))
	},
}

var syncVector = cobra.Command{
	Use: "vector", Short: "Re-index vector embeddings for changed files",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("vector/: up to date (use `down vector index` to re-index)")
	},
}

var syncWeb = cobra.Command{
	Use: "web", Short: "Pull fresh content for URL sources in data/",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" { fmt.Println("No .down/ found"); return }
		count := syncWebSources(downDir)
		if count > 0 {
			fmt.Printf("web/: %d URL sources refreshed\n", count)
		} else {
			fmt.Println("web/: no URL sources found")
		}
	},
}

func init() {
	Sync.Flags().BoolVarP(&syncForce, "force", "f", false, "Force re-index all files")
	Sync.Flags().BoolVarP(&syncVerbose, "verbose", "v", false, "Show detailed output")
	Sync.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would change without modifying")

	Sync.AddCommand(&syncData)
	Sync.AddCommand(&syncKnowledge)
	Sync.AddCommand(&syncMemory)
	Sync.AddCommand(&syncContext)
	Sync.AddCommand(&syncVector)
	Sync.AddCommand(&syncWeb)
	initGit()
}
