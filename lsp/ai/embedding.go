package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// EmbeddingProvider defines the interface for generating embeddings.
type EmbeddingProvider interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
}

// EmbeddingConfig configures the embedding system.
type EmbeddingConfig struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Dimension    int     `json:"dimension"`
	BatchSize    int     `json:"batch_size"`
	MaxTokens    int     `json:"max_tokens"`
	Normalize    bool    `json:"normalize"`
	CacheEnabled bool    `json:"cache_enabled"`
	CachePath    string  `json:"cache_path,omitempty"`
}

// DefaultEmbeddingConfig returns sensible defaults.
func DefaultEmbeddingConfig() EmbeddingConfig {
	return EmbeddingConfig{
		Provider:     "local",
		Model:        "bag-of-words",
		Dimension:    384,
		BatchSize:    32,
		MaxTokens:    512,
		Normalize:    true,
		CacheEnabled: true,
	}
}

// LocalEmbedding implements a simple bag-of-words embedding for offline use.
// This provides basic semantic similarity without external API calls.
type LocalEmbedding struct {
	config    EmbeddingConfig
	vocab     map[string]int
	idf       map[string]float64
	mu        sync.RWMutex
	docCount  int
}

// NewLocalEmbedding creates a local embedding provider.
func NewLocalEmbedding(config EmbeddingConfig) *LocalEmbedding {
	return &LocalEmbedding{
		config: config,
		vocab:  make(map[string]int),
		idf:    make(map[string]float64),
	}
}

func (le *LocalEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		results[i] = le.embedText(text)
	}
	return results, nil
}

func (le *LocalEmbedding) Dimension() int {
	return le.config.Dimension
}

func (le *LocalEmbedding) ModelName() string {
	return "local-bow-" + fmt.Sprintf("%d", le.config.Dimension)
}

func (le *LocalEmbedding) embedText(text string) []float32 {
	words := tokenize(text)
	embedding := make([]float32, le.config.Dimension)

	for _, word := range words {
		// Simple hash-based embedding
		h := hashWord(word, le.config.Dimension)
		weight := float32(1.0)
		if idfVal, ok := le.idf[word]; ok {
			weight = float32(idfVal)
		}
		embedding[h] += weight
	}

	if le.config.Normalize {
		normalize(embedding)
	}
	return embedding
}

// Train updates the vocabulary and IDF weights from a corpus.
func (le *LocalEmbedding) Train(documents []string) {
	le.mu.Lock()
	defer le.mu.Unlock()

	le.docCount = len(documents)
	docFreq := make(map[string]int)

	for _, doc := range documents {
		words := tokenize(doc)
		seen := make(map[string]bool)
		for _, w := range words {
			if _, ok := le.vocab[w]; !ok {
				le.vocab[w] = len(le.vocab)
			}
			if !seen[w] {
				docFreq[w]++
				seen[w] = true
			}
		}
	}

	// Calculate IDF
	for word, df := range docFreq {
		le.idf[word] = math.Log(float64(le.docCount+1) / float64(df+1))
	}
}

// FineTuneConfig configures model fine-tuning.
type FineTuneConfig struct {
	// Training pairs: (anchor, positive) - texts that should be similar
	TrainingPairs []TrainingPair `json:"training_pairs"`
	// Hard negatives: texts that should be dissimilar
	HardNegatives []TrainingPair `json:"hard_negatives,omitempty"`
	// Number of training epochs
	Epochs int `json:"epochs"`
	// Learning rate
	LearningRate float64 `json:"learning_rate"`
	// Batch size for training
	BatchSize int `json:"batch_size"`
	// Output model path
	OutputPath string `json:"output_path"`
}

// TrainingPair represents a pair of similar/dissimilar texts.
type TrainingPair struct {
	Anchor   string `json:"anchor"`
	Positive string `json:"positive"`
	Label    float64 `json:"label"` // 1.0 for similar, 0.0 for dissimilar
}

// FineTuneResult contains the results of a fine-tuning run.
type FineTuneResult struct {
	ModelPath    string    `json:"model_path"`
	Epochs       int       `json:"epochs"`
	FinalLoss    float64   `json:"final_loss"`
	TrainingSamples int   `json:"training_samples"`
	Duration     time.Duration `json:"duration"`
	Timestamp    time.Time `json:"timestamp"`
}

// FineTuner manages embedding model fine-tuning.
type FineTuner struct {
	config    FineTuneConfig
	embedding *LocalEmbedding
	mu        sync.Mutex
	results   []FineTuneResult
	storePath string
}

// NewFineTuner creates a fine-tuner for the local embedding model.
func NewFineTuner(embedding *LocalEmbedding, storePath string) *FineTuner {
	return &FineTuner{
		embedding: embedding,
		results:   make([]FineTuneResult, 0),
		storePath: storePath,
	}
}

// GenerateTrainingPairs automatically creates training pairs from the knowledge graph.
// Pairs are generated from:
// - Documents that share entities (positive pairs)
// - Headings and their content (positive pairs)
// - Unrelated documents (negative pairs)
func GenerateTrainingPairs(documents map[string]string, maxPairs int) []TrainingPair {
	var pairs []TrainingPair

	// Generate positive pairs from documents that share headings/topics
	type docChunk struct {
		uri     string
		heading string
		content string
	}
	var chunks []docChunk

	for uri, text := range documents {
		lines := strings.Split(text, "\n")
		var currentHeading string
		var currentContent strings.Builder

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				if currentContent.Len() > 50 && currentHeading != "" {
					chunks = append(chunks, docChunk{
						uri:     uri,
						heading: currentHeading,
						content: currentContent.String(),
					})
				}
				currentHeading = trimmed
				currentContent.Reset()
			} else {
				currentContent.WriteString(line + "\n")
			}
		}
		if currentContent.Len() > 50 {
			chunks = append(chunks, docChunk{
				uri:     uri,
				heading: currentHeading,
				content: currentContent.String(),
			})
		}
	}

	// Create positive pairs: heading → content
	for _, chunk := range chunks {
		if chunk.heading != "" && len(pairs) < maxPairs {
			pairs = append(pairs, TrainingPair{
				Anchor:   chunk.heading,
				Positive: chunk.content,
				Label:    1.0,
			})
		}
	}

	// Create negative pairs: unrelated chunks
	for i := 0; i < len(chunks) && len(pairs) < maxPairs*2; i++ {
		j := (i + len(chunks)/2) % len(chunks)
		if chunks[i].uri != chunks[j].uri {
			pairs = append(pairs, TrainingPair{
				Anchor:   chunks[i].content,
				Positive: chunks[j].content,
				Label:    0.0,
			})
		}
	}

	if len(pairs) > maxPairs {
		pairs = pairs[:maxPairs]
	}
	return pairs
}

// FineTune runs the fine-tuning process using contrastive learning.
func (ft *FineTuner) FineTune(config FineTuneConfig) (*FineTuneResult, error) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	start := time.Now()

	if config.Epochs == 0 {
		config.Epochs = 3
	}
	if config.LearningRate == 0 {
		config.LearningRate = 0.01
	}
	if config.BatchSize == 0 {
		config.BatchSize = 16
	}

	// Extract training corpus for vocabulary update
	var corpus []string
	for _, pair := range config.TrainingPairs {
		corpus = append(corpus, pair.Anchor, pair.Positive)
	}
	ft.embedding.Train(corpus)

	// Contrastive fine-tuning loop
	totalLoss := 0.0
	for epoch := 0; epoch < config.Epochs; epoch++ {
		epochLoss := 0.0
		for i := 0; i < len(config.TrainingPairs); i += config.BatchSize {
			end := i + config.BatchSize
			if end > len(config.TrainingPairs) {
				end = len(config.TrainingPairs)
			}
			batch := config.TrainingPairs[i:end]

			for _, pair := range batch {
				anchorEmb := ft.embedding.embedText(pair.Anchor)
				positiveEmb := ft.embedding.embedText(pair.Positive)

				// Compute cosine similarity
				sim := cosineSimilarity(anchorEmb, positiveEmb)

				// Contrastive loss
				var loss float64
				if pair.Label > 0.5 {
					loss = 1.0 - float64(sim)
				} else {
					margin := 0.5
					if float64(sim) > margin {
						loss = float64(sim) - margin
					}
				}
				epochLoss += loss

				// Update IDF weights based on gradient
				anchorWords := tokenize(pair.Anchor)
				for _, w := range anchorWords {
					if pair.Label > 0.5 {
						ft.embedding.idf[w] += config.LearningRate * (1.0 - float64(sim))
					} else {
						ft.embedding.idf[w] -= config.LearningRate * float64(sim) * 0.1
					}
				}
			}
		}
		totalLoss = epochLoss / float64(len(config.TrainingPairs))
	}

	result := &FineTuneResult{
		ModelPath:       config.OutputPath,
		Epochs:          config.Epochs,
		FinalLoss:       totalLoss,
		TrainingSamples: len(config.TrainingPairs),
		Duration:        time.Since(start),
		Timestamp:       time.Now(),
	}

	ft.results = append(ft.results, *result)

	// Save the fine-tuned model
	if config.OutputPath != "" {
		ft.saveModel(config.OutputPath)
	}

	return result, nil
}

// Results returns all fine-tuning results.
func (ft *FineTuner) Results() []FineTuneResult {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return ft.results
}

func (ft *FineTuner) saveModel(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	model := struct {
		Vocab    map[string]int     `json:"vocab"`
		IDF      map[string]float64 `json:"idf"`
		Dimension int               `json:"dimension"`
		DocCount  int               `json:"doc_count"`
	}{
		Vocab:     ft.embedding.vocab,
		IDF:       ft.embedding.idf,
		Dimension: ft.embedding.config.Dimension,
		DocCount:  ft.embedding.docCount,
	}

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadModel loads a fine-tuned model from disk.
func (le *LocalEmbedding) LoadModel(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var model struct {
		Vocab    map[string]int     `json:"vocab"`
		IDF      map[string]float64 `json:"idf"`
		Dimension int               `json:"dimension"`
		DocCount  int               `json:"doc_count"`
	}
	if err := json.Unmarshal(data, &model); err != nil {
		return err
	}

	le.mu.Lock()
	defer le.mu.Unlock()
	le.vocab = model.Vocab
	le.idf = model.IDF
	le.config.Dimension = model.Dimension
	le.docCount = model.DocCount
	return nil
}

// Utility functions

func tokenize(text string) []string {
	text = strings.ToLower(text)
	// Simple whitespace + punctuation tokenizer
	var words []string
	var current strings.Builder
	for _, r := range text {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			current.WriteRune(r)
		} else {
			if current.Len() > 1 {
				words = append(words, current.String())
			}
			current.Reset()
		}
	}
	if current.Len() > 1 {
		words = append(words, current.String())
	}
	return words
}

func hashWord(word string, dimension int) int {
	h := uint32(0)
	for _, ch := range word {
		h = h*31 + uint32(ch)
	}
	return int(h % uint32(dimension))
}

func normalize(v []float32) {
	var norm float64
	for _, val := range v {
		norm += float64(val) * float64(val)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range v {
			v[i] = float32(float64(v[i]) / norm)
		}
	}
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}
