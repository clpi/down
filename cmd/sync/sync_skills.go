package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	skillsOutput   string
	skillsProfile  string
	skillsNoFS     bool
	skillsNoKB     bool
	skillsNoMemory bool
	skillsNoVector bool
	skillsNoData   bool
)

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

func detectPlatform(root string) []string {
	indicators := map[string]string{
		"package.json":        "npm",
		"package-lock.json":   "npm",
		"yarn.lock":           "yarn",
		"pnpm-lock.yaml":      "pnpm",
		"go.mod":              "Go",
		"go.sum":              "Go",
		"Cargo.toml":          "Rust/Cargo",
		"Cargo.lock":          "Rust/Cargo",
		"requirements.txt":    "pip",
		"pyproject.toml":      "Python",
		"setup.py":            "Python",
		"Gemfile":             "Ruby/Bundler",
		"mix.exs":             "Elixir/Mix",
		"build.gradle":        "Gradle",
		"pom.xml":             "Maven",
		"composer.json":       "PHP/Composer",
		"Justfile":            "Just",
		"Makefile":            "Make",
		"CMakeLists.txt":      "CMake",
		"stylua.toml":         "StyLua",
		"selene.toml":         "Selene",
		".luarc.json":         "LuaLS",
		"down-scm-1.rockspec": "LuaRocks",
	}
	type depFile struct {
		file    string
		manager string
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

type skillData struct {
	ProjectName   string
	Profile       map[string]interface{}
	Languages     []string
	EntryPoints   []string
	Dependencies  []string
	Structure     []string
	Conventions   []string
	Entities      []map[string]interface{}
	IDFTerms      []string
	ClusterTopics map[int][]string
	DocCount      int
	MemoryEntries []map[string]interface{}
	DataFiles     []map[string]interface{}
	SyncIndex     map[string]interface{}
	ContextMD     string
}

func gatherSkillData(downDir, root string) *skillData {
	sd := &skillData{
		ProjectName: filepath.Base(root),
	}

	if !skillsNoFS {
		sd.Languages = detectLanguages(root)
		sd.EntryPoints = detectEntryPoints(root)
		sd.Dependencies = detectPlatform(root)
		sd.Structure = detectStructure(root, 0)
		sd.Conventions = detectConventions(root)
	}

	if !skillsNoKB {
		sd.Entities = loadKnowledgeEntities(downDir)
	}

	if !skillsNoVector {
		sd.IDFTerms, sd.ClusterTopics, sd.DocCount = loadVectorStats(downDir)
	}

	if !skillsNoMemory {
		sd.MemoryEntries = loadMemoryEntries(downDir)
	}

	if !skillsNoData {
		sd.DataFiles = loadDataFileIndex(downDir)
	}

	sd.SyncIndex = loadSyncIndexData(downDir)
	contextMD, _ := os.ReadFile(filepath.Join(downDir, "context.md"))
	sd.ContextMD = string(contextMD)

	if skillsProfile != "" {
		sd.Profile = loadProfileData(skillsProfile)
	}

	return sd
}

func loadKnowledgeEntities(downDir string) []map[string]interface{} {
	path := filepath.Join(downDir, "knowledge", "entities.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entities []map[string]interface{}
	json.Unmarshal(data, &entities)
	return entities
}

func loadVectorStats(downDir string) (idfTerms []string, clusterTopics map[int][]string, docCount int) {
	path := filepath.Join(downDir, "vector", "model.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, 0
	}
	var model map[string]interface{}
	if json.Unmarshal(data, &model) != nil {
		return nil, nil, 0
	}
	if dc, ok := model["doc_count"]; ok {
		if dcf, ok := dc.(float64); ok {
			docCount = int(dcf)
		}
	}
	if idf, ok := model["idf"].(map[string]interface{}); ok {
		type termScore struct {
			term  string
			score float64
		}
		var terms []termScore
		for t, v := range idf {
			if score, ok := v.(float64); ok && score > 0 {
				terms = append(terms, termScore{term: t, score: score})
			}
		}
		sort.Slice(terms, func(i, j int) bool { return terms[i].score > terms[j].score })
		limit := 30
		if limit > len(terms) {
			limit = len(terms)
		}
		for i := 0; i < limit; i++ {
			idfTerms = append(idfTerms, terms[i].term)
		}
	}

	clusterPath := filepath.Join(downDir, "vector", "clusters.json")
	if cd, err := os.ReadFile(clusterPath); err == nil {
		var clusters map[string]interface{}
		if json.Unmarshal(cd, &clusters) == nil {
			clusterTopics = make(map[int][]string)
			if labels, ok := clusters["labels"].(map[string]interface{}); ok {
				for k, v := range labels {
					var cIdx int
					fmt.Sscanf(k, "%d", &cIdx)
					if arr, ok := v.([]interface{}); ok {
						for _, item := range arr {
							if s, ok := item.(string); ok {
								clusterTopics[cIdx] = append(clusterTopics[cIdx], s)
							}
						}
					}
				}
			}
		}
	}

	return
}

func loadMemoryEntries(downDir string) []map[string]interface{} {
	memDir := filepath.Join(downDir, "memory")
	entries, err := os.ReadDir(memDir)
	if err != nil {
		return nil
	}
	var mems []map[string]interface{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "index.json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(memDir, e.Name()))
		if err != nil {
			continue
		}
		var m map[string]interface{}
		if json.Unmarshal(data, &m) == nil {
			mems = append(mems, m)
		}
	}
	return mems
}

func loadDataFileIndex(downDir string) []map[string]interface{} {
	indexPath := filepath.Join(downDir, "data", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}
	var index map[string]interface{}
	if json.Unmarshal(data, &index) != nil {
		return nil
	}
	var files []map[string]interface{}
	fileList, _ := index["files"].([]interface{})
	for _, f := range fileList {
		fname, _ := f.(string)
		fullPath := filepath.Join(downDir, "data", fname)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		contentStr := string(content)
		title := fname
		source := ""
		if strings.HasPrefix(contentStr, "---\n") {
			end := strings.Index(contentStr[4:], "\n---\n")
			if end > 0 {
				fm := contentStr[4 : 4+end]
				for _, line := range strings.Split(fm, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "title:") {
						title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "title: "))
					}
					if strings.HasPrefix(strings.TrimSpace(line), "source:") {
						source = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "source: "))
					}
				}
			}
		}
		preview := strings.ReplaceAll(contentStr, "\n", " ")
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		files = append(files, map[string]interface{}{
			"file":    fname,
			"title":   title,
			"source":  source,
			"preview": preview,
		})
	}
	return files
}

func loadSyncIndexData(downDir string) map[string]interface{} {
	idx := loadIndex(downDir)
	return map[string]interface{}{
		"version":   idx.Version,
		"fileCount": len(idx.Files),
		"lastSync":  idx.LastSync,
	}
}

func loadProfileData(name string) map[string]interface{} {
	home, _ := os.UserHomeDir()
	profilePath := filepath.Join(home, ".config", "down", "profile.json")
	if name != "" && name != "default" {
		profilePath = filepath.Join(home, ".config", "down", fmt.Sprintf("profile-%s.json", name))
	}
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil
	}
	var profile map[string]interface{}
	if json.Unmarshal(data, &profile) != nil {
		return nil
	}
	return profile
}

func generateSkillsMarkdown(sd *skillData) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", sd.ProjectName))
	b.WriteString(fmt.Sprintf("> Generated: %s by `down sync skills`\n", time.Now().Format("2006-01-02 15:04")))
	if len(sd.Profile) > 0 {
		b.WriteString(fmt.Sprintf("> Profile: %s\n", sd.Profile["name"]))
	}
	b.WriteString("\n")

	b.WriteString("## Project Overview\n\n")
	if len(sd.Languages) > 0 {
		b.WriteString(fmt.Sprintf("**Languages:** %s\n\n", strings.Join(sd.Languages, ", ")))
	}
	if ci, ok := sd.SyncIndex["version"]; ok {
		b.WriteString(fmt.Sprintf("**Version:** sync v%v, %v files tracked\n\n", ci, sd.SyncIndex["fileCount"]))
	}

	if len(sd.Profile) > 0 {
		if ai, ok := sd.Profile["ai_settings"].(map[string]interface{}); ok {
			b.WriteString("### AI Preferences\n\n")
			if prov, ok := ai["provider"]; ok && prov != "" {
				b.WriteString(fmt.Sprintf("- **Provider:** %s\n", prov))
			}
			if model, ok := ai["model"]; ok && model != "" {
				b.WriteString(fmt.Sprintf("- **Model:** %s\n", model))
			}
			if temp, ok := ai["temperature"]; ok {
				b.WriteString(fmt.Sprintf("- **Temperature:** %v\n", temp))
			}
			if tokens, ok := ai["max_tokens"]; ok {
				b.WriteString(fmt.Sprintf("- **Max tokens:** %v\n", tokens))
			}
			if sp, ok := ai["system_prompt"]; ok && sp != "" {
				prompt := sp.(string)
				if len(prompt) > 200 {
					prompt = prompt[:200] + "..."
				}
				b.WriteString(fmt.Sprintf("- **System prompt:** %s\n", prompt))
			}
			b.WriteString("\n")
		}
	}

	if len(sd.EntryPoints) > 0 {
		b.WriteString("## Entry Points\n\n")
		for _, e := range sd.EntryPoints {
			b.WriteString(fmt.Sprintf("- `%s`\n", e))
		}
		b.WriteString("\n")
	}

	if len(sd.Dependencies) > 0 {
		b.WriteString("## Dependencies\n\n")
		for _, d := range sd.Dependencies {
			b.WriteString(fmt.Sprintf("%s\n", d))
		}
		b.WriteString("\n")
	}

	if len(sd.Structure) > 0 {
		b.WriteString("## Project Structure\n\n```\n")
		for _, line := range sd.Structure {
			b.WriteString(fmt.Sprintf("%s\n", line))
		}
		b.WriteString("```\n\n")
	}

	if len(sd.Conventions) > 0 {
		b.WriteString("## Conventions\n\n")
		for _, c := range sd.Conventions {
			b.WriteString(fmt.Sprintf("- %s\n", c))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Key Modules\n\n")

	if len(sd.Entities) > 0 {
		b.WriteString("### Knowledge Graph Entities\n\n")
		tags := []string{}
		concepts := []map[string]interface{}{}
		for _, ent := range sd.Entities {
			kind, _ := ent["kind"].(string)
			if kind == "tag" {
				if name, ok := ent["name"].(string); ok {
					tags = append(tags, name)
				}
			} else if kind == "concept" {
				concepts = append(concepts, ent)
			}
		}
		if len(tags) > 0 {
			sort.Strings(tags)
			if len(tags) > 30 {
				tags = tags[:30]
			}
			b.WriteString(fmt.Sprintf("**Tags (%d):** %s\n\n", len(tags), strings.Join(tags, ", ")))
		}
		if len(concepts) > 0 {
			b.WriteString("**Wiki-linked concepts:**\n\n")
			for _, c := range concepts {
				name, _ := c["name"].(string)
				files, _ := c["file"].(string)
				fileList := strings.Split(files, ",")
				b.WriteString(fmt.Sprintf("- `[[%s]]` — referenced in %d file(s)\n", name, len(fileList)))
			}
			b.WriteString("\n")
		}
	}

	if sd.DocCount > 0 {
		b.WriteString("### Semantic Topics\n\n")
		b.WriteString(fmt.Sprintf("- **Embedded documents:** %d\n", sd.DocCount))
		if len(sd.IDFTerms) > 0 {
			b.WriteString(fmt.Sprintf("- **Top IDF terms:** %s\n", strings.Join(sd.IDFTerms, ", ")))
		}
		b.WriteString("\n")
	}
	if len(sd.ClusterTopics) > 0 {
		b.WriteString("**Topic clusters:**\n\n")
		for cIdx, labels := range sd.ClusterTopics {
			b.WriteString(fmt.Sprintf("- **Cluster %d:** %s\n", cIdx, strings.Join(labels, ", ")))
		}
		b.WriteString("\n")
	}

	if len(sd.MemoryEntries) > 0 {
		b.WriteString("## Memory\n\n")
		b.WriteString("Persisted knowledge entries from workspace memory.\n\n")
		for _, mem := range sd.MemoryEntries {
			key, _ := mem["key"].(string)
			value, _ := mem["value"].(string)
			if value == "" {
				if v, ok := mem["content"]; ok {
					value, _ = v.(string)
				}
			}
			if len(value) > 200 {
				value = value[:200] + "..."
			}
			value = strings.ReplaceAll(value, "\n", " ")
			b.WriteString(fmt.Sprintf("- **%s:** %s\n", key, value))
		}
		b.WriteString("\n")
	}

	if len(sd.DataFiles) > 0 {
		b.WriteString("## Referenced Content\n\n")
		b.WriteString("Content ingested from URLs and external sources.\n\n")
		for _, df := range sd.DataFiles {
			title, _ := df["title"].(string)
			source, _ := df["source"].(string)
			b.WriteString(fmt.Sprintf("- **%s**", title))
			if source != "" {
				b.WriteString(fmt.Sprintf(" — <%s>", source))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Commands\n\n```bash\n")
	b.WriteString("# Sync all workspace data (embeddings, knowledge, context, web)\n")
	b.WriteString("down sync\n\n")
	b.WriteString("# Rebuild vector embeddings with IDF weighting\n")
	b.WriteString("down sync vector\n\n")
	b.WriteString("# Generate this SKILL.md from all data sources\n")
	b.WriteString("down sync skills\n\n")
	b.WriteString("# Profile-based skills generation\n")
	b.WriteString("down sync skills --profile default\n\n")
	b.WriteString("# Full rebuild with forced re-index\n")
	b.WriteString("down sync --force\n")
	b.WriteString("```\n")

	return b.String()
}

var syncSkills = cobra.Command{
	Use:   "skills",
	Short: "Generate SKILL.md from all workspace data sources",
	Long: `Generate a comprehensive SKILL.md file that aggregates data from:

  Filesystem   — languages, entry points, dependencies, structure, conventions
  Knowledge/   — entities, wiki-links, tags extracted from workspace
  Vector/      — IDF-weighted terms, semantic topics, cluster labels
  Memory/      — persisted knowledge entries
  Data/        — ingested webpages and external references
  Context/     — sync history, file change tracking
  Profile      — AI preferences (model, provider, system prompt)

Output is written to SKILL.md in the workspace root (or --output path).`,
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Fprintln(os.Stderr, "No .down/ directory found. Run `down init` first.")
			os.Exit(1)
		}

		sd := gatherSkillData(downDir, root)
		md := generateSkillsMarkdown(sd)

		outPath := skillsOutput
		if outPath == "" {
			outPath = filepath.Join(root, "SKILL.md")
		}
		if err := os.WriteFile(outPath, []byte(md), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing skills: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Skills generated: %s\n", outPath)
		fmt.Printf("  languages: %d\n", len(sd.Languages))
		fmt.Printf("  entities:  %d\n", len(sd.Entities))
		fmt.Printf("  IDF terms: %d\n", len(sd.IDFTerms))
		fmt.Printf("  docs:      %d\n", sd.DocCount)
		fmt.Printf("  memory:    %d entries\n", len(sd.MemoryEntries))
	},
}
