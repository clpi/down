package ai

import (
	"context"
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

// EmbeddingProvider defines the interface for generating embeddings.
type EmbeddingProvider interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
}

// EmbeddingConfig configures the embedding system.
type EmbeddingConfig struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Dimension    int    `json:"dimension"`
	BatchSize    int    `json:"batch_size"`
	MaxTokens    int    `json:"max_tokens"`
	Normalize    bool   `json:"normalize"`
	CacheEnabled bool   `json:"cache_enabled"`
	CachePath    string `json:"cache_path,omitempty"`
	UseNgrams    bool   `json:"use_ngrams"`
	HashSeeds    int    `json:"hash_seeds"`
	SublinearTF  bool   `json:"sublinear_tf"`
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
		UseNgrams:    true,
		HashSeeds:    4,
		SublinearTF:  true,
	}
}

// Stop words for filtering common terms.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "up": true, "about": true,
	"into": true, "through": true, "during": true, "before": true, "after": true,
	"above": true, "below": true, "between": true, "out": true, "off": true,
	"over": true, "under": true, "again": true, "further": true, "then": true,
	"once": true, "here": true, "there": true, "when": true, "where": true,
	"why": true, "how": true, "all": true, "both": true, "each": true,
	"few": true, "more": true, "most": true, "other": true, "some": true,
	"such": true, "no": true, "nor": true, "not": true, "only": true,
	"own": true, "same": true, "so": true, "than": true, "too": true,
	"very": true, "s": true, "t": true, "can": true, "will": true,
	"just": true, "should": true, "now": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "having": true, "do": true,
	"does": true, "did": true, "doing": true, "would": true, "could": true,
	"shall": true, "may": true, "might": true, "must": true, "it": true,
	"its": true, "itself": true, "they": true, "them": true, "their": true,
	"theirs": true, "themselves": true, "what": true, "which": true, "who": true,
	"whom": true, "this": true, "that": true, "these": true, "those": true,
	"am": true, "i": true, "me": true, "my": true, "myself": true,
	"we": true, "our": true, "ours": true, "ourselves": true, "you": true,
	"your": true, "yours": true, "yourself": true, "yourselves": true,
	"he": true, "him": true, "his": true, "himself": true, "she": true,
	"her": true, "hers": true, "herself": true, "as": true, "if": true,
	"because": true, "until": true, "while": true, "also": true,
}

// LocalEmbedding implements a local bag-of-words embedding for offline use
// with n-grams, multi-hash, sublinear TF, and stop-word filtering.
type LocalEmbedding struct {
	config     EmbeddingConfig
	vocab      map[string]int
	idf        map[string]float64
	mu         sync.RWMutex
	docCount   int
	embeddings map[string][]float32
	centroids  map[string][]float32
}

// ClusteringResult holds k-means clustering output.
type ClusteringResult struct {
	Assignments  map[string]int       `json:"assignments"`
	Centroids    [][]float32           `json:"centroids"`
	Labels       map[int][]string      `json:"labels"`
}

// DupResult represents a near-duplicate pair.
type DupResult struct {
	ID1   string  `json:"id1"`
	ID2   string  `json:"id2"`
	Score float64 `json:"score"`
}

// EmbeddingStats holds embedding quality metrics.
type EmbeddingStats struct {
	Count     int     `json:"count"`
	Dim       int     `json:"dim"`
	Sparsity  float64 `json:"sparsity"`
	MeanNorm  float64 `json:"mean_norm"`
	Coverage  float64 `json:"coverage"`
}

// NewLocalEmbedding creates a local embedding provider.
func NewLocalEmbedding(config EmbeddingConfig) *LocalEmbedding {
	return &LocalEmbedding{
		config:     config,
		vocab:      make(map[string]int),
		idf:        make(map[string]float64),
		embeddings: make(map[string][]float32),
		centroids:  make(map[string][]float32),
	}
}

func (le *LocalEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
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
	words := Tokenize(text, le.config.UseNgrams)
	embedding := make([]float32, le.config.Dimension)

	tf := make(map[string]int)
	for _, w := range words {
		tf[w]++
	}

	for word, count := range tf {
		weight := float32(count)
		if le.config.SublinearTF {
			weight = float32(math.Log1p(float64(count)))
		}
		if idfVal, ok := le.idf[word]; ok {
			weight *= float32(idfVal)
		}
		indices := multiHash(word, le.config.Dimension, le.config.HashSeeds)
		for _, idx := range indices {
			embedding[idx] += weight
		}
	}

	if le.config.Normalize {
		normalize(embedding)
	}
	return embedding
}

// StoreEmbedding indexes an embedding by ID for later clustering/dedup.
func (le *LocalEmbedding) StoreEmbedding(id, text string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.embeddings[id] = le.embedText(text)
}

// DeleteEmbedding removes an indexed embedding.
func (le *LocalEmbedding) DeleteEmbedding(id string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	delete(le.embeddings, id)
}

// EmbeddingCount returns the number of stored embeddings.
func (le *LocalEmbedding) EmbeddingCount() int {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return len(le.embeddings)
}

// Centroid computes a document-level centroid from multiple text chunks.
func (le *LocalEmbedding) Centroid(texts []string) []float32 {
	if len(texts) == 0 {
		return nil
	}
	dim := le.config.Dimension
	centroid := make([]float32, dim)
	for _, text := range texts {
		emb := le.embedText(text)
		for i := range emb {
			centroid[i] += emb[i]
		}
	}
	n := float32(len(texts))
	for i := range centroid {
		centroid[i] /= n
	}
	normalize(centroid)
	return centroid
}

// BuildCentroids computes centroids for all stored sources.
func (le *LocalEmbedding) BuildCentroids(sourceChunks map[string][]string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.centroids = make(map[string][]float32)
	for source, chunks := range sourceChunks {
		le.centroids[source] = le.Centroid(chunks)
	}
}

// Cluster runs k-means clustering on stored embeddings.
func (le *LocalEmbedding) Cluster(k, maxIter int) ClusteringResult {
	le.mu.RLock()
	ids := make([]string, 0, len(le.embeddings))
	for id := range le.embeddings {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		le.mu.RUnlock()
		return ClusteringResult{}
	}
	if k > len(ids) {
		k = len(ids)
	}
	dim := len(le.embeddings[ids[0]])
	le.mu.RUnlock()

	centroids := make([][]float32, k)
	used := make(map[int]bool)
	for c := 0; c < k; c++ {
		var seedIdx int
		for {
			seedIdx = int(time.Now().UnixNano()) % len(ids)
			if !used[seedIdx] || len(used) >= len(ids) {
				break
			}
		}
		used[seedIdx] = true
		centroids[c] = make([]float32, dim)
		le.mu.RLock()
		src := le.embeddings[ids[seedIdx]]
		copy(centroids[c], src)
		le.mu.RUnlock()
	}

	for iter := 0; iter < maxIter; iter++ {
		assignments := make(map[string]int)
		le.mu.RLock()
		for _, id := range ids {
			emb := le.embeddings[id]
			bestC, bestSim := 0, float32(-2.0)
			for c := 0; c < k; c++ {
				sim := CosineSimilarity(emb, centroids[c])
				if sim > bestSim {
					bestSim = sim
					bestC = c
				}
			}
			assignments[id] = bestC
		}
		le.mu.RUnlock()

		newCentroids := make([][]float32, k)
		counts := make([]int, k)
		for c := 0; c < k; c++ {
			newCentroids[c] = make([]float32, dim)
		}
		le.mu.RLock()
		for _, id := range ids {
			c := assignments[id]
			counts[c]++
			emb := le.embeddings[id]
			for i := range emb {
				newCentroids[c][i] += emb[i]
			}
		}
		le.mu.RUnlock()
		for c := 0; c < k; c++ {
			if counts[c] > 0 {
				n := float32(counts[c])
				for i := range newCentroids[c] {
					newCentroids[c][i] /= n
				}
				normalize(newCentroids[c])
			}
		}

		moved := 0
		for c := 0; c < k; c++ {
			if CosineSimilarity(centroids[c], newCentroids[c]) < 0.999 {
				moved++
			}
		}
		centroids = newCentroids
		if moved == 0 {
			break
		}
	}

	assignments := make(map[string]int)
	le.mu.RLock()
	for _, id := range ids {
		emb := le.embeddings[id]
		bestC, bestSim := 0, float32(-2.0)
		for c := 0; c < k; c++ {
			sim := CosineSimilarity(emb, centroids[c])
			if sim > bestSim {
				bestSim = sim
				bestC = c
			}
		}
		assignments[id] = bestC
	}
	le.mu.RUnlock()

	labels := make(map[int][]string)
	clusterTexts := make(map[int][]string)
	le.mu.RLock()
	for id, c := range assignments {
		clusterTexts[c] = append(clusterTexts[c], id)
	}
	le.mu.RUnlock()
	for c, texts := range clusterTexts {
		labels[c] = extractClusterLabels(texts, 5)
	}

	return ClusteringResult{
		Assignments: assignments,
		Centroids:   centroids,
		Labels:      labels,
	}
}

// Dedup finds near-duplicate embedding pairs above a threshold.
func (le *LocalEmbedding) Dedup(threshold float64) []DupResult {
	le.mu.RLock()
	defer le.mu.RUnlock()

	ids := make([]string, 0, len(le.embeddings))
	for id := range le.embeddings {
		ids = append(ids, id)
	}

	var dups []DupResult
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			score := float64(CosineSimilarity(le.embeddings[ids[i]], le.embeddings[ids[j]]))
			if score >= threshold {
				dups = append(dups, DupResult{ID1: ids[i], ID2: ids[j], Score: score})
			}
		}
	}
	sort.Slice(dups, func(i, j int) bool { return dups[i].Score > dups[j].Score })
	return dups
}

// Stats computes embedding quality statistics.
func (le *LocalEmbedding) Stats() EmbeddingStats {
	le.mu.RLock()
	defer le.mu.RUnlock()

	count := len(le.embeddings)
	dim := le.config.Dimension
	if count == 0 {
		return EmbeddingStats{Count: 0, Dim: dim, Sparsity: 1, MeanNorm: 0, Coverage: 0}
	}

	var totalNonzero int
	var totalNorm float64
	for _, emb := range le.embeddings {
		nnz := 0
		var norm float64
		for _, v := range emb {
			if v != 0 {
				nnz++
			}
			norm += float64(v) * float64(v)
		}
		totalNonzero += nnz
		totalNorm += math.Sqrt(norm)
	}

	sparsity := 1.0 - float64(totalNonzero)/float64(count*dim)
	meanNorm := totalNorm / float64(count)

	totalTokens := 0
	covered := 0
	for _, w := range le.idf {
		totalTokens++
		if w > 0 {
			covered++
		}
	}
	coverage := 0.0
	if totalTokens > 0 {
		coverage = float64(covered) / float64(totalTokens)
	}

	return EmbeddingStats{
		Count:    count,
		Dim:      dim,
		Sparsity: sparsity,
		MeanNorm: meanNorm,
		Coverage: coverage,
	}
}

// Compare computes cosine similarity between two texts and returns shared terms.
func (le *LocalEmbedding) Compare(a, b string) (score float64, shared []string) {
	vecA := le.embedText(a)
	vecB := le.embedText(b)
	score = float64(CosineSimilarity(vecA, vecB))

	tokensA := make(map[string]bool)
	rawA := Tokenize(a, false)
	for _, t := range rawA {
		if len(t) > 2 && !stopWords[t] {
			tokensA[t] = true
		}
	}
	rawB := Tokenize(b, false)
	for _, t := range rawB {
		if tokensA[t] {
			shared = append(shared, t)
			delete(tokensA, t)
		}
	}
	return
}

// Train updates the vocabulary and IDF weights from a corpus.
func (le *LocalEmbedding) Train(documents []string) {
	le.mu.Lock()
	defer le.mu.Unlock()

	le.docCount = len(documents)
	docFreq := make(map[string]int)

	for _, doc := range documents {
		words := Tokenize(doc, le.config.UseNgrams)
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

	for word, df := range docFreq {
		le.idf[word] = math.Log(float64(le.docCount+1) / float64(df+1))
	}
}

// IDF returns a copy of the current IDF weights.
func (le *LocalEmbedding) IDF() map[string]float64 {
	le.mu.RLock()
	defer le.mu.RUnlock()
	out := make(map[string]float64, len(le.idf))
	for k, v := range le.idf {
		out[k] = v
	}
	return out
}

// FineTuneConfig configures model fine-tuning.
type FineTuneConfig struct {
	TrainingPairs []TrainingPair `json:"training_pairs"`
	HardNegatives []TrainingPair `json:"hard_negatives,omitempty"`
	Epochs        int            `json:"epochs"`
	LearningRate  float64        `json:"learning_rate"`
	BatchSize     int            `json:"batch_size"`
	OutputPath    string         `json:"output_path"`
}

// TrainingPair represents a pair of similar/dissimilar texts.
type TrainingPair struct {
	Anchor   string  `json:"anchor"`
	Positive string  `json:"positive"`
	Label    float64 `json:"label"`
}

// FineTuneResult contains the results of a fine-tuning run.
type FineTuneResult struct {
	ModelPath       string        `json:"model_path"`
	Epochs          int           `json:"epochs"`
	FinalLoss       float64       `json:"final_loss"`
	TrainingSamples int           `json:"training_samples"`
	Duration        time.Duration `json:"duration"`
	Timestamp       time.Time     `json:"timestamp"`
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

// GenerateTrainingPairs automatically creates training pairs from documents.
func GenerateTrainingPairs(documents map[string]string, maxPairs int) []TrainingPair {
	var pairs []TrainingPair

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

	for _, chunk := range chunks {
		if chunk.heading != "" && len(pairs) < maxPairs {
			pairs = append(pairs, TrainingPair{
				Anchor:   chunk.heading,
				Positive: chunk.content,
				Label:    1.0,
			})
		}
	}

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

	var corpus []string
	for _, pair := range config.TrainingPairs {
		corpus = append(corpus, pair.Anchor, pair.Positive)
	}
	ft.embedding.Train(corpus)

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
				sim := CosineSimilarity(anchorEmb, positiveEmb)

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

				anchorWords := Tokenize(pair.Anchor, ft.embedding.config.UseNgrams)
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

// extractClusterLabels finds top TF-IDF terms for a cluster.
func extractClusterLabels(memberIDs []string, topN int) []string {
	tf := make(map[string]int)
	df := make(map[string]int)
	n := 0
	for _, id := range memberIDs {
		n++
		seen := make(map[string]bool)
		for _, token := range extractTokens(id) {
			if len(token) > 2 && !stopWords[token] {
				tf[token]++
				if !seen[token] {
					seen[token] = true
					df[token]++
				}
			}
		}
	}

	type scored struct {
		token string
		score float64
	}
	var items []scored
	for token, freq := range tf {
		idf := math.Log((float64(n) + 1) / (float64(df[token]) + 1))
		items = append(items, scored{token: token, score: float64(freq) * idf})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].score > items[j].score })

	labels := make([]string, 0, topN)
	for i := 0; i < topN && i < len(items); i++ {
		labels = append(labels, items[i].token)
	}
	return labels
}

func extractTokens(text string) []string {
	return Tokenize(text, false)
}

// Utility functions

// Tokenize splits text into tokens with optional n-grams and stop-word filtering.
func Tokenize(text string, ngrams bool) []string {
	text = strings.ToLower(text)
	var unigrams []string
	var current strings.Builder
	for _, r := range text {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			current.WriteRune(r)
		} else {
			if current.Len() > 1 {
				w := current.String()
				if !stopWords[w] {
					unigrams = append(unigrams, w)
				}
			}
			current.Reset()
		}
	}
	if current.Len() > 1 {
		w := current.String()
		if !stopWords[w] {
			unigrams = append(unigrams, w)
		}
	}

	if !ngrams {
		return unigrams
	}

	var tokens []string
	tokens = append(tokens, unigrams...)
	for i := 0; i < len(unigrams)-1; i++ {
		tokens = append(tokens, unigrams[i]+"_"+unigrams[i+1])
	}
	for i := 0; i < len(unigrams)-2; i++ {
		tokens = append(tokens, unigrams[i]+"_"+unigrams[i+1]+"_"+unigrams[i+2])
	}
	return tokens
}

func multiHash(word string, dimension, seeds int) []int {
	indices := make([]int, seeds)
	for s := 0; s < seeds; s++ {
		h := uint32(s)
		for _, ch := range word {
			h = h*31 + uint32(ch)
		}
		indices[s] = int(h%uint32(dimension))
	}
	return indices
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

// CosineSimilarity computes cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float32 {
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
