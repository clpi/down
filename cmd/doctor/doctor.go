package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clpi/down/cmd/wsutil"
	"github.com/spf13/cobra"
)

var doctorRoot string

var Doctor = cobra.Command{
	Use:     "doctor",
	Aliases: []string{"health", "check", "lint"},
	Short:   "Check workspace health and report issues",
	Long:    "Scan the workspace for missing directories, broken references, stale data, and configuration issues.",
	Run: func(cmd *cobra.Command, args []string) {
		root := wsutil.ResolveRoot(doctorRoot)
		downDir := filepath.Join(root, ".down")

		if _, err := os.Stat(downDir); os.IsNotExist(err) {
			fmt.Println("ERROR: No .down/ directory found. Run `down init` first.")
			return
		}

		issues := 0
		ok := 0

		check := func(name string, pass bool, detail string) {
			if pass {
				ok++
				fmt.Printf("  \033[32m✓\033[0m %s\n", detail)
			} else {
				issues++
				fmt.Printf("  \033[31m✗\033[0m %s\n", detail)
			}
		}

		fmt.Println("Workspace Health Check")
		fmt.Println("=====================")
		fmt.Println()

		// Required directories
		fmt.Println("Directories:")
		for _, sub := range []string{"data", "knowledge", "memory", "context", "vector", "templates"} {
			dir := filepath.Join(downDir, sub)
			_, err := os.Stat(dir)
			check(sub, err == nil, sub+"/ exists")
		}

		fmt.Println()

		// Configuration
		fmt.Println("Configuration:")
		configPath := filepath.Join(downDir, "down.json")
		if _, err := os.Stat(configPath); err == nil {
			data, _ := os.ReadFile(configPath)
			var cfg map[string]interface{}
			if json.Unmarshal(data, &cfg) == nil {
				name, _ := cfg["name"].(string)
				wiki, _ := cfg["wiki"].(bool)
				check("config", true, fmt.Sprintf("down.json present (name=%q, wiki=%v)", name, wiki))
			} else {
				check("config", false, "down.json is invalid JSON")
			}
		} else {
			check("config", false, "down.json missing")
		}

		ignorePath := filepath.Join(downDir, ".downignore")
		_, err := os.Stat(ignorePath)
		check(".downignore", err == nil, ".downignore present")

		fmt.Println()

		// Knowledge index
		fmt.Println("Knowledge:")
		idxPath := filepath.Join(downDir, "knowledge", "index.json")
		if data, err := os.ReadFile(idxPath); err == nil {
			var idx struct {
				Version  int    `json:"version"`
				LastSync string `json:"last_sync"`
			}
			if json.Unmarshal(data, &idx) == nil {
				check("index", true, fmt.Sprintf("File index v%d (last sync: %s)", idx.Version, idx.LastSync))
			} else {
				check("index", false, "File index is invalid JSON")
			}
		} else {
			check("index", false, "No file index — run `down sync knowledge`")
		}

		entitiesPath := filepath.Join(downDir, "knowledge", "entities.json")
		if data, err := os.ReadFile(entitiesPath); err == nil {
			var entities []map[string]interface{}
			if json.Unmarshal(data, &entities) == nil {
				tags := 0
				for _, e := range entities {
					if k, _ := e["kind"].(string); k == "tag" {
						tags++
					}
				}
				check("entities", true, fmt.Sprintf("Knowledge graph: %d entities (%d tags)", len(entities), tags))
			}
		} else {
			check("entities", false, "No knowledge graph — run `down sync knowledge`")
		}

		fmt.Println()

		// Vector
		fmt.Println("Vectors:")
		modelPath := filepath.Join(downDir, "vector", "model.json")
		if data, err := os.ReadFile(modelPath); err == nil {
			var model map[string]interface{}
			if json.Unmarshal(data, &model) == nil {
				idfSize := 0
				if idf, ok := model["idf"].(map[string]interface{}); ok {
					idfSize = len(idf)
				}
				check("model", true, fmt.Sprintf("IDF model present (dim=%v, docs=%v, terms=%d)",
					model["dimension"], model["doc_count"], idfSize))
			}
		} else {
			check("model", false, "No vector model — run `down sync vector`")
		}

		vDir := filepath.Join(downDir, "vector")
		if entries, err := os.ReadDir(vDir); err == nil {
			fileEmb := 0
			memEmb := 0
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
					continue
				}
				if strings.HasPrefix(e.Name(), "memory_") {
					memEmb++
				} else if strings.HasPrefix(e.Name(), "file_") {
					fileEmb++
				}
			}
			check("embeddings", true, fmt.Sprintf("Embeddings: %d file + %d memory", fileEmb, memEmb))
		}

		fmt.Println()

		// Memory
		fmt.Println("Memory:")
		memIndex := filepath.Join(downDir, "memory", "index.json")
		if data, err := os.ReadFile(memIndex); err == nil {
			var mi struct{ Count, Expired int }
			if json.Unmarshal(data, &mi) == nil {
				check("memory", true, fmt.Sprintf("Memory: %d active, %d expired", mi.Count, mi.Expired))
			}
		} else {
			check("memory", true, "Memory store empty (OK)")
		}

		fmt.Println()

		// Data
		fmt.Println("Data:")
		dataIndex := filepath.Join(downDir, "data", "index.json")
		if data, err := os.ReadFile(dataIndex); err == nil {
			var di struct{ Count int }
			if json.Unmarshal(data, &di) == nil {
				check("data", true, fmt.Sprintf("Data index: %d files", di.Count))
			}
		} else {
			check("data", true, "No ingested data (OK)")
		}

		fmt.Println()

		// Context & Skills
		fmt.Println("Context:")
		ctxPath := filepath.Join(downDir, "context.md")
		if _, err := os.Stat(ctxPath); err == nil {
			check("context.md", true, "Context document present")
		} else {
			check("context.md", false, "No context.md — run `down sync context`")
		}

		skillsPath := filepath.Join(root, "SKILL.md")
		if _, err := os.Stat(skillsPath); err == nil {
			check("SKILL.md", true, "Skills document present")
		} else {
			check("SKILL.md", false, "No SKILL.md — run `down sync skills`")
		}

		fmt.Println()

		// Summary
		fmt.Printf("═══ Results: %d OK, %d issues ═══\n", ok, issues)
		if issues > 0 {
			fmt.Println("\nFix issues with:")
			fmt.Println("  down sync              Full workspace sync")
			fmt.Println("  down sync vector       Rebuild embeddings")
			fmt.Println("  down sync knowledge    Rebuild knowledge graph")
			fmt.Println("  down sync skills       Regenerate SKILL.md")
			fmt.Println("  down template init     Initialize default templates")
		}
	},
}

func init() {
	Doctor.Flags().StringVar(&doctorRoot, "root", "", "Workspace root")
}
