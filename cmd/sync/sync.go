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

	"github.com/clpi/down/cmd/add"
	"github.com/clpi/down/lsp/ai"
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

type kbEntity struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	File      string   `json:"file"`
	Relations []string `json:"relations"`
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
	if err != nil {
		return ""
	}
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
		if idx.Files == nil {
			idx.Files = make(map[string]FileEntry)
		}
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
		if matched, _ := filepath.Match(pat, name); matched {
			return true
		}
	}
	return false
}

func loadDownIgnore(downDir string) []string {
	path := filepath.Join(downDir, ".downignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
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
	Added       int
	Modified    int
	Deleted     int
	Unchanged   int
	Errors      int
	AddedFiles   []string
	ModifiedFiles []string
	DeletedFiles  []string
	Files       []string
}

func syncWorkspace(downDir, root string) *SyncResult {
	result := &SyncResult{}
	idx := loadIndex(downDir)
	ignores := loadDownIgnore(downDir)
	ignores = append(ignores, ".git", ".svn", "node_modules", ".DS_Store", ".down")

	currentFiles := make(map[string]bool)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors++
			return nil
		}
		if info.IsDir() {
			n := info.Name()
			if shouldIgnore(n, ignores) {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == ".down" || info.Name() == ".git" {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		currentFiles[rel] = true

		hash := fileHash(path)
		entry, exists := idx.Files[rel]

		if !exists {
			result.Added++
			result.Files = append(result.Files, fmt.Sprintf("+ %s", rel))
			result.AddedFiles = append(result.AddedFiles, rel)
			idx.Files[rel] = FileEntry{
				Path:    rel,
				Hash:    hash,
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
				Synced:  time.Now().Format(time.RFC3339),
			}
		} else if entry.Hash != hash {
			result.Modified++
			result.Files = append(result.Files, fmt.Sprintf("~ %s", rel))
			result.ModifiedFiles = append(result.ModifiedFiles, rel)
			idx.Files[rel] = FileEntry{
				Path:    rel,
				Hash:    hash,
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
				Synced:  time.Now().Format(time.RFC3339),
			}
		} else {
			result.Unchanged++
		}
		return nil
	})

	for rel := range idx.Files {
		if !currentFiles[rel] {
			result.Deleted++
			result.Files = append(result.Files, fmt.Sprintf("- %s", rel))
			result.DeletedFiles = append(result.DeletedFiles, rel)
			delete(idx.Files, rel)
		}
	}

	idx.Version++
	saveIndex(downDir, idx)
	return result
}

func loadMarkdownFiles(root string) []string {
	var files []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			n := info.Name()
			if n == ".git" || n == "node_modules" || n == ".down" || strings.HasPrefix(n, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".md") {
			rel, _ := filepath.Rel(root, path)
			files = append(files, rel)
		}
		return nil
	})
	return files
}

func buildEmbeddings(downDir, root string, changedFiles []string) int {
	emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())
	vDir := filepath.Join(downDir, "vector")
	os.MkdirAll(vDir, 0755)

	allMD := loadMarkdownFiles(root)
	var allTexts []string
	for _, rel := range allMD {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		allTexts = append(allTexts, string(data))
	}

	emb.Train(allTexts)

	count := 0
	toIndex := changedFiles
	if len(changedFiles) == 0 || syncForce {
		toIndex = allMD
	}
	for _, rel := range toIndex {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		text := string(data)
		vec, _ := emb.Embed(nil, []string{text})
		if vec == nil || len(vec) == 0 {
			continue
		}

		entry := vectorEntry{
			ID:      "file_" + strings.ReplaceAll(rel, "/", "_"),
			Vector:  vec[0],
			Text:    truncateText(text, 1000),
			Source:  rel,
			Created: time.Now().Format(time.RFC3339),
		}
		saveVectorEntry(vDir, entry)
		count++
	}

	modelPath := filepath.Join(vDir, "model.json")
	idf := emb.IDF()
	model := map[string]interface{}{
		"idf":        idf,
		"dimension":  emb.Dimension(),
		"doc_count":  len(allMD),
		"updated":    time.Now().Format(time.RFC3339),
	}
	modelData, _ := json.MarshalIndent(model, "", "  ")
	os.WriteFile(modelPath, modelData, 0644)

	if syncVerbose {
		fmt.Printf("  vector/    %d files embedded, IDF: %d terms\n", count, len(idf))
	}
	return count
}

type vectorEntry struct {
	ID      string    `json:"id"`
	Vector  []float32 `json:"vector"`
	Text    string    `json:"text"`
	Source  string    `json:"source"`
	Created string    `json:"created"`
}

func saveVectorEntry(dir string, v vectorEntry) error {
	id := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ' ' || r == ':' {
			return '_'
		}
		return r
	}, v.ID)
	data, _ := json.MarshalIndent(v, "", "  ")
	return os.WriteFile(filepath.Join(dir, id+".json"), data, 0644)
}

func syncWebSources(downDir string) int {
	dataDir := filepath.Join(downDir, "data")
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fullPath := filepath.Join(dataDir, e.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(data)
		if !strings.HasPrefix(content, "---\n") {
			continue
		}
		end := strings.Index(content[4:], "\n---\n")
		if end <= 0 {
			continue
		}
		fm := content[4 : 4+end]
		for _, line := range strings.Split(fm, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "source: http") {
				source := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "source: "))
				if syncVerbose {
					fmt.Printf("  web/       re-fetching: %s\n", source)
				}
				newContent, err := add.FetchURL(source)
				if err != nil {
					if syncVerbose {
						fmt.Printf("  web/       error fetching %s: %v\n", source, err)
					}
					continue
				}
				os.WriteFile(fullPath, []byte(newContent), 0644)
				count++
			}
		}
	}
	return count
}

func buildKnowledgeBase(downDir, root string) int {
	allMD := loadMarkdownFiles(root)
	count := 0

	var entities []kbEntity
	tagIndex := make(map[string]bool)
	linkIndex := make(map[string][]string)

	for _, rel := range allMD {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		content := string(data)

		if strings.HasPrefix(content, "---\n") {
			end := strings.Index(content[4:], "\n---\n")
			if end > 0 {
				fm := content[4 : 4+end]
				for _, line := range strings.Split(fm, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "tags:") {
						tags := strings.TrimPrefix(strings.TrimSpace(line), "tags:")
						for _, t := range strings.Split(tags, ",") {
							t = strings.TrimSpace(t)
							if t != "" {
								tagIndex[t] = true
							}
						}
					}
				}
			}
		}

		for _, line := range strings.Split(content, "\n") {
			for _, tag := range extractTags(line) {
				tagIndex[tag] = true
			}
			for _, link := range extractWikiLinks(line) {
				linkIndex[link] = append(linkIndex[link], rel)
			}
		}
	}

	for tag := range tagIndex {
		entities = append(entities, kbEntity{
			ID:        "tag/" + tag,
			Name:      tag,
			Kind:      "tag",
			File:      "",
			Relations: nil,
		})
		count++
	}

	for link, files := range linkIndex {
		entities = append(entities, kbEntity{
			ID:        "wiki/" + link,
			Name:      link,
			Kind:      "concept",
			File:      strings.Join(files, ","),
			Relations: files,
		})
		count++
	}

	// Link memory tags and keys into the knowledge graph
	entities = linkMemoryToKnowledge(entities, downDir)

	kbPath := filepath.Join(downDir, "knowledge", "entities.json")
	kbData, _ := json.MarshalIndent(entities, "", "  ")
	os.WriteFile(kbPath, kbData, 0644)

	return count
}

func extractTags(line string) []string {
	var tags []string
	for _, word := range strings.Fields(line) {
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			tag := strings.TrimPrefix(word, "#")
			tag = strings.TrimRight(tag, ".,;:!?)")
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

func extractWikiLinks(line string) []string {
	var links []string
	start := 0
	for {
		i := strings.Index(line[start:], "[[")
		if i < 0 {
			break
		}
		j := strings.Index(line[start+i+2:], "]]")
		if j < 0 {
			break
		}
		link := line[start+i+2 : start+i+2+j]
		if link != "" {
			links = append(links, link)
		}
		start = start + i + j + 3
	}
	return links
}

func buildContext(downDir, root string, result *SyncResult) string {
	idx := loadIndex(downDir)
	name := filepath.Base(root)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s — Workspace Context\n\n", name))
	b.WriteString(fmt.Sprintf("> Synced: %s\n", time.Now().Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("> Files: %d tracked\n\n", len(idx.Files)))

	b.WriteString("## File Statistics\n\n")
	b.WriteString(fmt.Sprintf("- Added: %d\n", result.Added))
	b.WriteString(fmt.Sprintf("- Modified: %d\n", result.Modified))
	b.WriteString(fmt.Sprintf("- Deleted: %d\n", result.Deleted))
	b.WriteString(fmt.Sprintf("- Unchanged: %d\n\n", result.Unchanged))

	if len(result.AddedFiles)+len(result.ModifiedFiles) > 0 {
		b.WriteString("## Recent Changes\n\n")
		for _, f := range result.AddedFiles {
			b.WriteString(fmt.Sprintf("- + %s\n", f))
		}
		for _, f := range result.ModifiedFiles {
			b.WriteString(fmt.Sprintf("- ~ %s\n", f))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Directory Map\n\n")
	dirs := make(map[string]bool)
	for rel := range idx.Files {
		dir := filepath.Dir(rel)
		if dir != "." {
			dirs[dir] = true
		}
	}
	for d := range dirs {
		b.WriteString(fmt.Sprintf("- `%s/`\n", d))
	}
	b.WriteString("\n")

	embsFile := filepath.Join(downDir, "vector", "model.json")
	if data, err := os.ReadFile(embsFile); err == nil {
		var model map[string]interface{}
		if json.Unmarshal(data, &model) == nil {
			b.WriteString("## Vector Index\n\n")
			if dim, ok := model["dimension"]; ok {
				b.WriteString(fmt.Sprintf("- Dimension: %v\n", dim))
			}
			if dc, ok := model["doc_count"]; ok {
				b.WriteString(fmt.Sprintf("- Documents: %v\n", dc))
			}
			if idf, ok := model["idf"].(map[string]interface{}); ok {
				b.WriteString(fmt.Sprintf("- IDF terms: %d\n\n", len(idf)))
			}
		}
	}

	return b.String()
}

func syncDataFiles(downDir, root string) int {
	dataDir := filepath.Join(downDir, "data")
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return 0
	}
	count := 0
	var compacted []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		count++
		compacted = append(compacted, e.Name())
	}

	compactPath := filepath.Join(downDir, "data", "index.json")
	index := map[string]interface{}{
		"files":    compacted,
		"count":    count,
		"updated":  time.Now().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(index, "", "  ")
	os.WriteFile(compactPath, data, 0644)

	if syncVerbose {
		fmt.Printf("  data/      %d files indexed\n", count)
	}
	return count
}

func syncMemoryStore(downDir string) int {
	memDir := filepath.Join(downDir, "memory")
	os.MkdirAll(memDir, 0755)

	entries, err := os.ReadDir(memDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			data, err := os.ReadFile(filepath.Join(memDir, e.Name()))
			if err != nil {
				continue
			}
			var mem map[string]interface{}
			if json.Unmarshal(data, &mem) != nil {
				continue
			}
			if ts, ok := mem["expires"]; ok {
				if expire, ok := ts.(string); ok {
					if t, err := time.Parse(time.RFC3339, expire); err == nil && time.Now().After(t) {
						os.Remove(filepath.Join(memDir, e.Name()))
						count++
						continue
					}
				}
			}
		}
	}

	indexPath := filepath.Join(memDir, "index.json")
	entries, _ = os.ReadDir(memDir)
	active := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() != "index.json" {
			active = append(active, e.Name())
		}
	}
	idx := map[string]interface{}{
		"entries":  active,
		"count":    len(active),
		"expired":  count,
		"reconciled": time.Now().Format(time.RFC3339),
	}
	idxData, _ := json.MarshalIndent(idx, "", "  ")
	os.WriteFile(indexPath, idxData, 0644)

	if syncVerbose {
		fmt.Printf("  memory/    %d active, %d expired\n", len(active), count)
	}
	return len(active)
}

// embedMemoryEntries embeds active memory entries using the same engine as file embeddings.
// Writes memory vectors to vector/memory_<key>.json so they're searchable alongside file vectors.
func embedMemoryEntries(downDir string, emb *ai.LocalEmbedding) int {
	memDir := filepath.Join(downDir, "memory")
	vDir := filepath.Join(downDir, "vector")
	os.MkdirAll(vDir, 0755)

	entries, err := os.ReadDir(memDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "index.json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(memDir, e.Name()))
		if err != nil {
			continue
		}
		var mem map[string]interface{}
		if json.Unmarshal(data, &mem) != nil {
			continue
		}
		key, _ := mem["key"].(string)
		value, _ := mem["value"].(string)
		if value == "" {
			if v, ok := mem["content"]; ok {
				value, _ = v.(string)
			}
		}
		if key == "" || value == "" {
			continue
		}

		vecs, _ := emb.Embed(nil, []string{value})
		if len(vecs) == 0 {
			continue
		}
		entry := vectorEntry{
			ID:      "memory_" + sanitizeKey(key),
			Vector:  vecs[0],
			Text:    truncateText(value, 500),
			Source:  "memory/" + key,
			Created: time.Now().Format(time.RFC3339),
		}
		saveVectorEntry(vDir, entry)
		count++
	}
	return count
}

// linkMemoryToKnowledge adds memory tags and keys to the knowledge graph entities.
func linkMemoryToKnowledge(entities []kbEntity, downDir string) []kbEntity {
	memDir := filepath.Join(downDir, "memory")
	entries, err := os.ReadDir(memDir)
	if err != nil {
		return entities
	}

	tagSet := make(map[string]bool)
	memKeys := make(map[string][]string) // tag -> memory keys

	for _, ent := range entities {
		if ent.Kind == "tag" {
			tagSet[ent.Name] = true
		}
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "index.json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(memDir, e.Name()))
		if err != nil {
			continue
		}
		var mem map[string]interface{}
		if json.Unmarshal(data, &mem) != nil {
			continue
		}
		key, _ := mem["key"].(string)
		if key == "" {
			continue
		}

		// Add memory key as a concept entity
		entities = append(entities, kbEntity{
			ID:        "memory/" + key,
			Name:      key,
			Kind:      "concept",
			File:      "memory",
			Relations: nil,
		})

		// Extract tags from memory entry
		if tags, ok := mem["tags"].([]interface{}); ok {
			for _, t := range tags {
				if tag, ok := t.(string); ok && tag != "" {
					memKeys[tag] = append(memKeys[tag], key)
					if !tagSet[tag] {
						tagSet[tag] = true
						entities = append(entities, kbEntity{
							ID:   "tag/" + tag,
							Name: tag,
							Kind: "tag",
							File: "",
						})
					}
				}
			}
		}
	}

	// Add back-links for memory tags
	for tag, keys := range memKeys {
		entities = append(entities, kbEntity{
			ID:        "memtag/" + tag,
			Name:      tag,
			Kind:      "concept",
			File:      strings.Join(keys, ","),
			Relations: keys,
		})
	}

	return entities
}

// autoTagData uses nearest-neighbor embedding search to suggest tags for data/ content.
// Writes suggested tags back to the data file frontmatter.
func autoTagData(downDir string, emb *ai.LocalEmbedding) int {
	dataDir := filepath.Join(downDir, "data")
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return 0
	}

	// Collect existing tags from knowledge entities
	entitiesPath := filepath.Join(downDir, "knowledge", "entities.json")
	existingTags := make(map[string]bool)
	if data, err := os.ReadFile(entitiesPath); err == nil {
		var entities []kbEntity
		if json.Unmarshal(data, &entities) == nil {
			for _, ent := range entities {
				if ent.Kind == "tag" {
					existingTags[ent.Name] = true
				}
			}
		}
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fullPath := filepath.Join(dataDir, e.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		contentStr := string(content)

		// Skip if already has tags
		if strings.Contains(contentStr, "\ntags:") {
			continue
		}

		// Find body (skip frontmatter)
		body := contentStr
		if strings.HasPrefix(body, "---\n") {
			if end := strings.Index(body[4:], "\n---\n"); end > 0 {
				body = body[4+end+5:]
			}
		}

		// Find nearest knowledge entity by embedding similarity
		vecs, _ := emb.Embed(nil, []string{body})
		if len(vecs) == 0 {
			continue
		}

		var bestTag string
		var bestScore float32
		for tag := range existingTags {
			tagVecs, _ := emb.Embed(nil, []string{tag})
			if len(tagVecs) > 0 {
				score := ai.CosineSimilarity(vecs[0], tagVecs[0])
				if score > 0.3 && score > bestScore {
					bestScore = score
					bestTag = tag
				}
			}
		}

		if bestTag == "" {
			continue
		}

		// Insert tags: into frontmatter
		var tagged string
		if strings.HasPrefix(contentStr, "---\n") {
			end := strings.Index(contentStr[4:], "\n---\n")
			if end > 0 {
				fm := contentStr[4 : 4+end]
				rest := contentStr[4+end+5:]
				tagged = "---\n" + fm + "\ntags: " + bestTag + "\n---\n\n" + rest
				os.WriteFile(fullPath, []byte(tagged), 0644)
				count++
			}
		}
	}
	return count
}

func sanitizeKey(key string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ' ' || r == ':' || r == '.' || r == '@' || r == '#' {
			return '_'
		}
		return r
	}, key)
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

var Sync = cobra.Command{
	Use:     "sync",
	Aliases: []string{"sy"},
	Short:   "Sync workspace: detect changes, update embeddings, knowledge, context, web",
	Long: `Sync workspace data by detecting new, modified, and deleted files.

Updates:
  data/         Indexes all ingested data files
  knowledge/    File index with SHA-256 hashes + entity extraction
  memory/       Reconciles memory entries, prunes expired
  context/      Regenerates workspace context document
  vector/       Builds IDF vocabulary and re-embeds changed files
  web/          Re-fetches content for URL-based sources

Runs sub-syncs in order: data → knowledge → memory → context → vector → web`,
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Fprintln(os.Stderr, "No .down/ directory found. Run `down init` first.")
			os.Exit(1)
		}
		ensureDirs(downDir)

		fmt.Println("Syncing workspace...")

		result := syncWorkspace(downDir, root)
		if syncVerbose {
			for _, f := range result.Files {
				fmt.Println(" ", f)
			}
		}
		fmt.Printf("  +%d added  ~%d modified  -%d deleted  =%d unchanged\n",
			result.Added, result.Modified, result.Deleted, result.Unchanged)

		if syncDryRun {
			fmt.Println("\n  (dry run — no files modified)")
			return
		}

		allChanged := append(result.AddedFiles, result.ModifiedFiles...)

		dc := syncDataFiles(downDir, root)
		fmt.Printf("  data/      %d files\n", dc)

		kc := buildKnowledgeBase(downDir, root)
		idx := loadIndex(downDir)
		fmt.Printf("  knowledge/ index v%d, %d files, %d entities\n", idx.Version, len(idx.Files), kc)

		mc := syncMemoryStore(downDir)
		fmt.Printf("  memory/    %d entries\n", mc)

		ctx := buildContext(downDir, root, result)
		os.WriteFile(filepath.Join(downDir, "context.md"), []byte(ctx), 0644)
		fmt.Println("  context/   regenerated")

		var vc int
		if syncForce || len(allChanged) > 0 {
			vc = buildEmbeddings(downDir, root, allChanged)
			fmt.Printf("  vector/    %d embeddings updated\n", vc)
		} else {
			vc = buildEmbeddings(downDir, root, nil)
			fmt.Printf("  vector/    %d embeddings checked\n", vc)
		}

		// Embed memory entries for semantic search
		emb := ai.NewLocalEmbedding(ai.DefaultEmbeddingConfig())
		allMD := loadMarkdownFiles(root)
		var allTexts []string
		for _, rel := range allMD {
			if data, err := os.ReadFile(filepath.Join(root, rel)); err == nil {
				allTexts = append(allTexts, string(data))
			}
		}
		emb.Train(allTexts)
		if mc2 := embedMemoryEntries(downDir, emb); mc2 > 0 {
			fmt.Printf("  memory/    %d entries embedded for semantic search\n", mc2)
		}

		// Auto-tag data/ content using embedding similarity
		if ac := autoTagData(downDir, emb); ac > 0 {
			fmt.Printf("  data/      %d files auto-tagged\n", ac)
		}

		wc := syncWebSources(downDir)
		if wc > 0 {
			fmt.Printf("  web/       %d URL sources refreshed\n", wc)
		} else {
			fmt.Println("  web/       up to date")
		}

		fmt.Println("\nWorkspace synced.")
	},
}

var syncData = cobra.Command{
	Use:   "data",
	Short: "Re-index all ingested files in data/",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Println("No .down/ found")
			return
		}
		c := syncDataFiles(downDir, root)
		fmt.Printf("data/: %d files indexed\n", c)
	},
}

var syncKnowledge = cobra.Command{
	Use:   "knowledge",
	Short: "Rebuild file index with SHA-256 hashes and entity extraction",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Println("No .down/ found")
			return
		}
		result := syncWorkspace(downDir, root)
		kc := buildKnowledgeBase(downDir, root)
		fmt.Printf("Knowledge: +%d added ~%d changed -%d removed, %d entities\n",
			result.Added, result.Modified, result.Deleted, kc)
	},
}

var syncMemory = cobra.Command{
	Use:   "memory",
	Short: "Reconcile memory entries and prune expired",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Println("No .down/ found")
			return
		}
		c := syncMemoryStore(downDir)
		fmt.Printf("memory: %d active entries\n", c)
	},
}

var syncContext = cobra.Command{
	Use:   "context",
	Short: "Regenerate workspace context document",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Println("No .down/ found")
			return
		}
		result := syncWorkspace(downDir, root)
		ctx := buildContext(downDir, root, result)
		os.WriteFile(filepath.Join(downDir, "context.md"), []byte(ctx), 0644)
		fmt.Printf("context/: regenerated (%d files)\n", len(loadIndex(downDir).Files))
	},
}

var syncVector = cobra.Command{
	Use:   "vector",
	Short: "Rebuild vector embeddings for all markdown files in workspace",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Println("No .down/ found")
			return
		}
		result := syncWorkspace(downDir, root)
		allChanged := append(result.AddedFiles, result.ModifiedFiles...)
		if syncForce {
			allChanged = nil
		}
		vc := buildEmbeddings(downDir, root, allChanged)
		fmt.Printf("vector/: %d embeddings updated\n", vc)
	},
}

var syncWeb = cobra.Command{
	Use:   "web",
	Short: "Re-fetch content for URL-based sources in data/",
	Run: func(cmd *cobra.Command, args []string) {
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Println("No .down/ found")
			return
		}
		wc := syncWebSources(downDir)
		if wc > 0 {
			fmt.Printf("web/: %d URL sources refreshed\n", wc)
		} else {
			fmt.Println("web/: no URL sources to refresh")
		}
	},
}

var syncAdd = cobra.Command{
	Use:   "add <url>",
	Short: "Fetch URL and store as markdown in .down/data/",
	Long: `Fetch a webpage and convert to markdown, storing in .down/data/.

This command:
  1. Fetches the URL content
  2. Extracts Open Graph metadata (title, description)
  3. Converts HTML to clean markdown format
  4. Adds frontmatter with source URL and fetch date
  5. Saves to .down/data/ with a sanitized filename

Useful for archiving online documentation, articles, or API references.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		root, _ := os.Getwd()
		downDir := findDownDir(root)
		if downDir == "" {
			fmt.Fprintln(os.Stderr, "No .down/ directory found. Run `down init` first.")
			os.Exit(1)
		}
		dataDir := filepath.Join(downDir, "data")
		os.MkdirAll(dataDir, 0755)

		content, err := add.FetchURL(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching URL: %v\n", err)
			os.Exit(1)
		}

		domain := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
		filename := strings.ReplaceAll(domain, ".", "_") + ".md"
		if strings.Contains(filename, "/") {
			filename = strings.ReplaceAll(filename, "/", "_")
		}

		outPath := filepath.Join(dataDir, filename)
		if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added: %s\n  -> %s\n", url, outPath)
	},
}

func init() {
	Sync.Flags().BoolVarP(&syncForce, "force", "f", false, "Force full rebuild of all indices and embeddings")
	Sync.Flags().BoolVarP(&syncVerbose, "verbose", "v", false, "Show detailed output")
	Sync.Flags().BoolVar(&syncDryRun, "dry-run", false, "Show what would change without modifying")

	syncSkills.Flags().StringVarP(&skillsOutput, "output", "o", "", "Output path (default: SKILL.md in workspace root)")
	syncSkills.Flags().StringVarP(&skillsProfile, "profile", "p", "", "Profile name for AI settings (default: none)")
	syncSkills.Flags().BoolVar(&skillsNoFS, "no-fs", false, "Skip filesystem structure detection")
	syncSkills.Flags().BoolVar(&skillsNoKB, "no-kb", false, "Skip knowledge graph entities")
	syncSkills.Flags().BoolVar(&skillsNoMemory, "no-memory", false, "Skip memory entries")
	syncSkills.Flags().BoolVar(&skillsNoVector, "no-vector", false, "Skip vector/semantic topics")
	syncSkills.Flags().BoolVar(&skillsNoData, "no-data", false, "Skip ingested data references")

	Sync.AddCommand(&syncAdd)
	Sync.AddCommand(&syncData)
	Sync.AddCommand(&syncKnowledge)
	Sync.AddCommand(&syncMemory)
	Sync.AddCommand(&syncContext)
	Sync.AddCommand(&syncVector)
	Sync.AddCommand(&syncWeb)
	Sync.AddCommand(&syncSkills)
	initGit()
}
