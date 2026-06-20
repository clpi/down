package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var statusRoot string

var Status = cobra.Command{
	Use:     "status",
	Aliases: []string{"st", "info", "dashboard"},
	Short:   "Show workspace status dashboard",
	Long:    "Display a complete overview of all workspace data: files, knowledge, memory, vectors, data, and context.",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(statusRoot)
		downDir := filepath.Join(root, ".down")

		fmt.Printf("Workspace: %s\n", filepath.Base(root))
		fmt.Printf("Path:      %s\n", root)
		fmt.Printf("Down dir:  %s\n\n", downDir)

		// Files
		idxPath := filepath.Join(downDir, "knowledge", "index.json")
		if data, err := os.ReadFile(idxPath); err == nil {
			var idx struct {
				Version  int    `json:"version"`
				LastSync string `json:"last_sync"`
				Files    map[string]interface{} `json:"files"`
			}
			if json.Unmarshal(data, &idx) == nil {
				fmt.Printf("═══ Files ═══\n")
				fmt.Printf("  Tracked:    %d\n", len(idx.Files))
				fmt.Printf("  Version:    v%d\n", idx.Version)
				fmt.Printf("  Last sync:  %s\n\n", idx.LastSync)
			}
		}

		// Knowledge graph
		kbPath := filepath.Join(downDir, "knowledge", "entities.json")
		if data, err := os.ReadFile(kbPath); err == nil {
			var entities []map[string]interface{}
			if json.Unmarshal(data, &entities) == nil {
				tags := 0
				concepts := 0
				for _, e := range entities {
					if k, _ := e["kind"].(string); k == "tag" {
						tags++
					} else {
						concepts++
					}
				}
				fmt.Printf("═══ Knowledge ═══\n")
				fmt.Printf("  Entities:   %d\n", len(entities))
				fmt.Printf("  Tags:       %d\n", tags)
				fmt.Printf("  Concepts:   %d\n\n", concepts)
			}
		}

		// Memory
		memDir := filepath.Join(downDir, "memory")
		memIndex := filepath.Join(memDir, "index.json")
		if data, err := os.ReadFile(memIndex); err == nil {
			var mi struct {
				Count   int `json:"count"`
				Expired int `json:"expired"`
			}
			if json.Unmarshal(data, &mi) == nil {
				fmt.Printf("═══ Memory ═══\n")
				fmt.Printf("  Active:     %d\n", mi.Count)
				fmt.Printf("  Expired:    %d\n\n", mi.Expired)
			}
		}

		// Vector / Embeddings
		vDir := filepath.Join(downDir, "vector")
		if entries, err := os.ReadDir(vDir); err == nil {
			fileVecs := 0
			memVecs := 0
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
					continue
				}
				if strings.HasPrefix(e.Name(), "memory_") {
					memVecs++
				} else if strings.HasPrefix(e.Name(), "file_") {
					fileVecs++
				}
			}
			fmt.Printf("═══ Vectors ═══\n")
			fmt.Printf("  File embeddings:   %d\n", fileVecs)
			fmt.Printf("  Memory embeddings: %d\n\n", memVecs)
		}

		// Vector model
		modelPath := filepath.Join(vDir, "model.json")
		if data, err := os.ReadFile(modelPath); err == nil {
			var model map[string]interface{}
			if json.Unmarshal(data, &model) == nil {
				if dim, ok := model["dimension"]; ok {
					fmt.Printf("  Dimension:  %v\n", dim)
				}
				if dc, ok := model["doc_count"]; ok {
					fmt.Printf("  Documents:  %v\n", dc)
				}
				if idf, ok := model["idf"].(map[string]interface{}); ok {
					fmt.Printf("  IDF terms:  %d\n", len(idf))
				}
				fmt.Println()
			}
		}

		// Data / ingested
		dataDir := filepath.Join(downDir, "data")
		dataIndex := filepath.Join(dataDir, "index.json")
		if data, err := os.ReadFile(dataIndex); err == nil {
			var di struct {
				Count int      `json:"count"`
				Files []string `json:"files"`
			}
			if json.Unmarshal(data, &di) == nil {
				fmt.Printf("═══ Data ═══\n")
				fmt.Printf("  Ingested:   %d files\n", di.Count)
				for _, f := range di.Files {
					fmt.Printf("    - %s\n", f)
				}
				fmt.Println()
			}
		}

		// Context
		ctxPath := filepath.Join(downDir, "context.md")
		if info, err := os.Stat(ctxPath); err == nil {
			fmt.Printf("═══ Context ═══\n")
			fmt.Printf("  Size:       %d bytes\n", info.Size())
			fmt.Printf("  Modified:   %s\n\n", info.ModTime().Format("2006-01-02 15:04"))
		}

		// Templates
		tmplDir := filepath.Join(downDir, "templates")
		if entries, err := os.ReadDir(tmplDir); err == nil {
			count := 0
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					count++
				}
			}
			if count > 0 {
				fmt.Printf("═══ Templates ═══\n")
				fmt.Printf("  Available:  %d\n\n", count)
			}
		}

		// Skills
		skillsPath := filepath.Join(root, "SKILL.md")
		if info, err := os.Stat(skillsPath); err == nil {
			fmt.Printf("═══ Skills ═══\n")
			fmt.Printf("  Size:       %d bytes\n", info.Size())
			fmt.Printf("  Modified:   %s\n\n", info.ModTime().Format("2006-01-02 15:04"))
		}

		// Quick commands reference
		fmt.Printf("═══ Commands ═══\n")
		fmt.Printf("  down sync              Full workspace sync\n")
		fmt.Printf("  down sync skills       Generate SKILL.md\n")
		fmt.Printf("  down memory search -s  Semantic memory search\n")
		fmt.Printf("  down template list     List templates\n")
		fmt.Printf("  down vector stats      Embedding quality\n")
		fmt.Printf("  down find              Find files\n")
	},
}

func init() {
	Status.Flags().StringVar(&statusRoot, "root", "", "Workspace root")
}
