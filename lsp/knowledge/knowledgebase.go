package knowledge

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// KnowledgeBase is a higher-level abstraction over the knowledge graph
// that supports document collections, semantic search, and query operations.
type KnowledgeBase struct {
	mu          sync.RWMutex
	Graph       *Graph
	Collections map[string]*Collection `json:"collections"`
	Embeddings  *EmbeddingStore        `json:"embeddings,omitempty"`
	StorePath   string                 `json:"-"`
	Stats       KBStats                `json:"stats"`
}

// KBStats tracks knowledge base statistics.
type KBStats struct {
	TotalDocuments  int       `json:"total_documents"`
	TotalEntities   int       `json:"total_entities"`
	TotalRelations  int       `json:"total_relations"`
	TotalChunks     int       `json:"total_chunks"`
	LastIndexed     time.Time `json:"last_indexed"`
	IndexDuration   float64   `json:"index_duration_ms"`
}

// Collection groups related documents together.
type Collection struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Documents   []string  `json:"documents"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DocumentChunk represents a section of a document for embedding.
type DocumentChunk struct {
	ID        string    `json:"id"`
	DocURI    string    `json:"doc_uri"`
	Content   string    `json:"content"`
	Heading   string    `json:"heading,omitempty"`
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Embedding []float32 `json:"embedding,omitempty"`
}

// EmbeddingStore manages document chunk embeddings for semantic search.
type EmbeddingStore struct {
	mu        sync.RWMutex
	Chunks    []DocumentChunk `json:"chunks"`
	Dimension int             `json:"dimension"`
	Model     string          `json:"model"`
}

// SearchResult represents a semantic search result.
type SearchResult struct {
	Chunk      DocumentChunk `json:"chunk"`
	Score      float64       `json:"score"`
	EntityHits []*Entity     `json:"entity_hits,omitempty"`
}

// QueryOptions configures a knowledge base query.
type QueryOptions struct {
	MaxResults   int      `json:"max_results"`
	MinScore     float64  `json:"min_score"`
	Collections  []string `json:"collections,omitempty"`
	EntityKinds  []EntityKind `json:"entity_kinds,omitempty"`
	DateAfter    string   `json:"date_after,omitempty"`
	DateBefore   string   `json:"date_before,omitempty"`
}

// DefaultQueryOptions returns sensible defaults for queries.
func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		MaxResults: 10,
		MinScore:   0.1,
	}
}

// NewKnowledgeBase creates a knowledge base backed by a graph.
func NewKnowledgeBase(graph *Graph, storePath string) *KnowledgeBase {
	kb := &KnowledgeBase{
		Graph:       graph,
		Collections: make(map[string]*Collection),
		Embeddings:  NewEmbeddingStore(384, "default"),
		StorePath:   storePath,
	}
	kb.load()
	return kb
}

// NewEmbeddingStore creates a new embedding store.
func NewEmbeddingStore(dimension int, model string) *EmbeddingStore {
	return &EmbeddingStore{
		Chunks:    make([]DocumentChunk, 0),
		Dimension: dimension,
		Model:     model,
	}
}

// CreateCollection creates a new document collection.
func (kb *KnowledgeBase) CreateCollection(name, description string) *Collection {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	col := &Collection{
		ID:          id,
		Name:        name,
		Description: description,
		Documents:   make([]string, 0),
		Tags:        make([]string, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	kb.Collections[id] = col
	return col
}

// AddToCollection adds a document URI to a collection.
func (kb *KnowledgeBase) AddToCollection(collectionID string, docURI string) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	col, ok := kb.Collections[collectionID]
	if !ok {
		return fmt.Errorf("collection %q not found", collectionID)
	}

	for _, d := range col.Documents {
		if d == docURI {
			return nil // already in collection
		}
	}
	col.Documents = append(col.Documents, docURI)
	col.UpdatedAt = time.Now()
	return nil
}

// RemoveFromCollection removes a document from a collection.
func (kb *KnowledgeBase) RemoveFromCollection(collectionID string, docURI string) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	col, ok := kb.Collections[collectionID]
	if !ok {
		return fmt.Errorf("collection %q not found", collectionID)
	}

	filtered := col.Documents[:0]
	for _, d := range col.Documents {
		if d != docURI {
			filtered = append(filtered, d)
		}
	}
	col.Documents = filtered
	col.UpdatedAt = time.Now()
	return nil
}

// IndexDocument chunks a document and prepares it for semantic search.
func (kb *KnowledgeBase) IndexDocument(uri string, text string) {
	chunks := chunkDocument(uri, text)
	kb.Embeddings.mu.Lock()
	defer kb.Embeddings.mu.Unlock()

	// Remove old chunks for this document
	filtered := kb.Embeddings.Chunks[:0]
	for _, c := range kb.Embeddings.Chunks {
		if c.DocURI != uri {
			filtered = append(filtered, c)
		}
	}
	kb.Embeddings.Chunks = append(filtered, chunks...)

	kb.mu.Lock()
	kb.Stats.TotalChunks = len(kb.Embeddings.Chunks)
	kb.Stats.LastIndexed = time.Now()
	kb.mu.Unlock()
}

// chunkDocument splits a document into meaningful chunks based on headings.
func chunkDocument(uri string, text string) []DocumentChunk {
	lines := strings.Split(text, "\n")
	var chunks []DocumentChunk
	var currentChunk strings.Builder
	var currentHeading string
	startLine := 0
	chunkIdx := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// New heading starts a new chunk
		if strings.HasPrefix(trimmed, "#") && currentChunk.Len() > 0 {
			content := strings.TrimSpace(currentChunk.String())
			if len(content) > 20 {
				chunks = append(chunks, DocumentChunk{
					ID:        fmt.Sprintf("%s#%d", uri, chunkIdx),
					DocURI:    uri,
					Content:   content,
					Heading:   currentHeading,
					StartLine: startLine,
					EndLine:   i - 1,
				})
				chunkIdx++
			}
			currentChunk.Reset()
			startLine = i
			currentHeading = trimmed
		}

		if strings.HasPrefix(trimmed, "#") && currentChunk.Len() == 0 {
			currentHeading = trimmed
			startLine = i
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")

		// Also chunk on large paragraphs (>500 chars)
		if currentChunk.Len() > 500 && trimmed == "" {
			content := strings.TrimSpace(currentChunk.String())
			if len(content) > 20 {
				chunks = append(chunks, DocumentChunk{
					ID:        fmt.Sprintf("%s#%d", uri, chunkIdx),
					DocURI:    uri,
					Content:   content,
					Heading:   currentHeading,
					StartLine: startLine,
					EndLine:   i,
				})
				chunkIdx++
			}
			currentChunk.Reset()
			startLine = i + 1
		}
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		if len(content) > 20 {
			chunks = append(chunks, DocumentChunk{
				ID:        fmt.Sprintf("%s#%d", uri, chunkIdx),
				DocURI:    uri,
				Content:   content,
				Heading:   currentHeading,
				StartLine: startLine,
				EndLine:   len(lines) - 1,
			})
		}
	}

	return chunks
}

// Search performs a combined keyword and entity search across the knowledge base.
func (kb *KnowledgeBase) Search(query string, opts QueryOptions) []SearchResult {
	if opts.MaxResults == 0 {
		opts.MaxResults = 10
	}

	var results []SearchResult
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	kb.Embeddings.mu.RLock()
	chunks := kb.Embeddings.Chunks
	kb.Embeddings.mu.RUnlock()

	for _, chunk := range chunks {
		// Filter by collection if specified
		if len(opts.Collections) > 0 {
			inCollection := false
			kb.mu.RLock()
			for _, colID := range opts.Collections {
				if col, ok := kb.Collections[colID]; ok {
					for _, d := range col.Documents {
						if d == chunk.DocURI {
							inCollection = true
							break
						}
					}
				}
				if inCollection {
					break
				}
			}
			kb.mu.RUnlock()
			if !inCollection {
				continue
			}
		}

		// Calculate keyword relevance score
		score := keywordScore(chunk.Content, queryWords)

		if score > opts.MinScore {
			result := SearchResult{
				Chunk: chunk,
				Score: score,
			}

			// Add related entities
			if kb.Graph != nil {
				graphResults := kb.Graph.Search(query)
				for _, ent := range graphResults {
					for _, src := range ent.Sources {
						if src.URI == chunk.DocURI {
							result.EntityHits = append(result.EntityHits, ent)
							break
						}
					}
				}
			}

			results = append(results, result)
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}

	return results
}

// keywordScore calculates a simple TF-based relevance score.
func keywordScore(content string, queryWords []string) float64 {
	contentLower := strings.ToLower(content)
	contentWords := strings.Fields(contentLower)
	if len(contentWords) == 0 {
		return 0
	}

	hits := 0
	for _, qw := range queryWords {
		for _, cw := range contentWords {
			if strings.Contains(cw, qw) {
				hits++
			}
		}
	}

	// Normalize by content length (TF-like)
	score := float64(hits) / math.Sqrt(float64(len(contentWords)))
	return score
}

// GetRelatedDocuments finds documents related to a given document based on shared entities.
func (kb *KnowledgeBase) GetRelatedDocuments(docURI string, maxResults int) []string {
	if kb.Graph == nil {
		return nil
	}

	entities := kb.Graph.EntitiesByDocument(docURI)
	docScores := make(map[string]float64)

	for _, ent := range entities {
		for _, src := range ent.Sources {
			if src.URI != docURI {
				docScores[src.URI] += 1.0 / float64(ent.Mentions)
			}
		}
	}

	type docScore struct {
		URI   string
		Score float64
	}
	var scored []docScore
	for uri, score := range docScores {
		scored = append(scored, docScore{URI: uri, Score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	if maxResults > 0 && len(scored) > maxResults {
		scored = scored[:maxResults]
	}

	result := make([]string, len(scored))
	for i, s := range scored {
		result[i] = s.URI
	}
	return result
}

// UpdateStats refreshes the knowledge base statistics.
func (kb *KnowledgeBase) UpdateStats() {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	if kb.Graph != nil {
		kb.Stats.TotalEntities = len(kb.Graph.Entities)
		kb.Stats.TotalRelations = len(kb.Graph.Relations)

		docs := make(map[string]bool)
		for _, ent := range kb.Graph.Entities {
			for _, src := range ent.Sources {
				docs[src.URI] = true
			}
		}
		kb.Stats.TotalDocuments = len(docs)
	}
	kb.Stats.TotalChunks = len(kb.Embeddings.Chunks)
}

// Save persists the knowledge base to disk.
func (kb *KnowledgeBase) Save() error {
	if kb.StorePath == "" {
		return nil
	}
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	dir := filepath.Dir(kb.StorePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(kb, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(kb.StorePath, data, 0644)
}

func (kb *KnowledgeBase) load() {
	if kb.StorePath == "" {
		return
	}
	data, err := os.ReadFile(kb.StorePath)
	if err != nil {
		return
	}
	var loaded KnowledgeBase
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}
	if loaded.Collections != nil {
		kb.Collections = loaded.Collections
	}
	if loaded.Embeddings != nil && len(loaded.Embeddings.Chunks) > 0 {
		kb.Embeddings = loaded.Embeddings
	}
	kb.Stats = loaded.Stats
}

// Summary returns a human-readable summary of the knowledge base.
func (kb *KnowledgeBase) Summary() string {
	kb.UpdateStats()
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("## Knowledge Base\n\n")
	sb.WriteString(fmt.Sprintf("- Documents: %d\n", kb.Stats.TotalDocuments))
	sb.WriteString(fmt.Sprintf("- Entities: %d\n", kb.Stats.TotalEntities))
	sb.WriteString(fmt.Sprintf("- Relations: %d\n", kb.Stats.TotalRelations))
	sb.WriteString(fmt.Sprintf("- Chunks: %d\n", kb.Stats.TotalChunks))
	sb.WriteString(fmt.Sprintf("- Collections: %d\n", len(kb.Collections)))
	if !kb.Stats.LastIndexed.IsZero() {
		sb.WriteString(fmt.Sprintf("- Last indexed: %s\n", kb.Stats.LastIndexed.Format(time.RFC3339)))
	}

	if len(kb.Collections) > 0 {
		sb.WriteString("\n### Collections\n\n")
		for _, col := range kb.Collections {
			sb.WriteString(fmt.Sprintf("- **%s** (%d docs): %s\n", col.Name, len(col.Documents), col.Description))
		}
	}

	return sb.String()
}
