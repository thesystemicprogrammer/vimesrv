package recommendation

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/james-bowman/nlp"
	"github.com/james-bowman/sparse"
	"gonum.org/v1/gonum/mat"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// TFIDFRecommender builds TF-IDF vectors for content and computes similarities
type TFIDFRecommender struct {
	vectorizer  *nlp.CountVectoriser
	transformer *nlp.TfidfTransformer
	matrix      mat.Matrix
	itemIDs     []int64                          // Maps matrix row index to item ID
	idToIndex   map[int64]int                    // Maps item ID to matrix row index
	metadata    map[int64]domain.ContentMetadata // Item metadata for display
	itemType    string                           // "movie" or "series"
}

// NewTFIDFRecommender creates a new TF-IDF recommender
func NewTFIDFRecommender(itemType string) *TFIDFRecommender {
	return &TFIDFRecommender{
		itemType:  itemType,
		idToIndex: make(map[int64]int),
		metadata:  make(map[int64]domain.ContentMetadata),
	}
}

// Build constructs the TF-IDF matrix from content features
func (r *TFIDFRecommender) Build(items []domain.ContentFeatures) error {
	if len(items) == 0 {
		return fmt.Errorf("no items to build model from")
	}

	// Collect documents and build mappings
	documents := make([]string, len(items))
	r.itemIDs = make([]int64, len(items))
	r.idToIndex = make(map[int64]int, len(items))
	r.metadata = make(map[int64]domain.ContentMetadata, len(items))

	for i, item := range items {
		documents[i] = item.FeatureText
		r.itemIDs[i] = item.ID
		r.idToIndex[item.ID] = i
		r.metadata[item.ID] = item.Metadata
	}

	// Create count vectorizer with no stop words (our tokens are already normalized)
	r.vectorizer = nlp.NewCountVectoriser()
	// Use custom tokenizer that handles our weighted token format
	r.vectorizer.Tokeniser = &simpleTokeniser{}

	// Fit and transform documents to get term-frequency matrix
	tfMatrix, err := r.vectorizer.FitTransform(documents...)
	if err != nil {
		return fmt.Errorf("vectorize documents: %w", err)
	}

	// Apply TF-IDF transformation
	r.transformer = nlp.NewTfidfTransformer()
	tfidfMatrix, err := r.transformer.FitTransform(tfMatrix)
	if err != nil {
		return fmt.Errorf("apply tfidf: %w", err)
	}

	r.matrix = tfidfMatrix

	return nil
}

// GetSimilar returns the top N similar items for a given item ID
func (r *TFIDFRecommender) GetSimilar(itemID int64, topN int) ([]domain.SimilarItem, error) {
	if r.matrix == nil {
		return nil, fmt.Errorf("model not built")
	}

	idx, exists := r.idToIndex[itemID]
	if !exists {
		return nil, fmt.Errorf("item %d not found in model", itemID)
	}

	numRows, _ := r.matrix.Dims()

	// Get the target row vector
	targetVec := getRowVector(r.matrix, idx)

	// Compute similarity with all other items
	type scoredItem struct {
		id    int64
		score float64
	}
	similarities := make([]scoredItem, 0, numRows-1)

	for i := 0; i < numRows; i++ {
		if i == idx {
			continue // Skip self
		}

		compareVec := getRowVector(r.matrix, i)
		similarity := cosineSimilarity(targetVec, compareVec)

		if similarity > 0 {
			similarities = append(similarities, scoredItem{
				id:    r.itemIDs[i],
				score: similarity,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].score > similarities[j].score
	})

	// Take top N
	if len(similarities) > topN {
		similarities = similarities[:topN]
	}

	// Convert to result type
	result := make([]domain.SimilarItem, len(similarities))
	for i, s := range similarities {
		result[i] = domain.SimilarItem{
			ID:              s.id,
			SimilarityScore: s.score,
		}
	}

	return result, nil
}

// GetMetadata returns the metadata for an item
func (r *TFIDFRecommender) GetMetadata(itemID int64) (domain.ContentMetadata, bool) {
	meta, ok := r.metadata[itemID]
	return meta, ok
}

// GetItemCount returns the number of items in the model
func (r *TFIDFRecommender) GetItemCount() int {
	return len(r.itemIDs)
}

// GetFeatureCount returns the number of unique features (vocabulary size)
func (r *TFIDFRecommender) GetFeatureCount() int {
	if r.matrix == nil {
		return 0
	}
	_, cols := r.matrix.Dims()
	return cols
}

// getRowVector extracts a row from the matrix as a vector
func getRowVector(m mat.Matrix, row int) []float64 {
	_, cols := m.Dims()
	vec := make([]float64, cols)

	// Try sparse matrix first for efficiency
	if sparse, ok := m.(*sparse.CSR); ok {
		for j := 0; j < cols; j++ {
			vec[j] = sparse.At(row, j)
		}
	} else {
		for j := 0; j < cols; j++ {
			vec[j] = m.At(row, j)
		}
	}

	return vec
}

// cosineSimilarity computes the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// simpleTokeniser implements nlp.Tokeniser interface for whitespace tokenization
type simpleTokeniser struct{}

// ForEachIn iterates over each token within text and invokes function f
func (t *simpleTokeniser) ForEachIn(text string, f func(token string)) {
	for _, token := range t.Tokenise(text) {
		f(token)
	}
}

// Tokenise splits the feature text into tokens using whitespace
func (t *simpleTokeniser) Tokenise(text string) []string {
	text = strings.ToLower(text)
	tokens := strings.Fields(text)

	// Filter out empty tokens
	result := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if tok != "" {
			result = append(result, tok)
		}
	}

	return result
}
