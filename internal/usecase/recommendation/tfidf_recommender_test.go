package recommendation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

func TestNewTFIDFRecommender(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	assert.NotNil(t, recommender)
	assert.Equal(t, "movie", recommender.itemType)
	assert.NotNil(t, recommender.idToIndex)
	assert.NotNil(t, recommender.metadata)
}

func TestTFIDFRecommender_Build_EmptyItems(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	err := recommender.Build([]domain.ContentFeatures{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no items to build model from")
}

func TestTFIDFRecommender_Build_Success(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	items := []domain.ContentFeatures{
		{
			ID:          1,
			Type:        "movie",
			FeatureText: "action adventure hero explosion",
			Metadata:    domain.ContentMetadata{Title: "Action Movie 1"},
		},
		{
			ID:          2,
			Type:        "movie",
			FeatureText: "action adventure hero sword",
			Metadata:    domain.ContentMetadata{Title: "Action Movie 2"},
		},
		{
			ID:          3,
			Type:        "movie",
			FeatureText: "romance love drama comedy",
			Metadata:    domain.ContentMetadata{Title: "Romance Movie"},
		},
	}

	err := recommender.Build(items)

	require.NoError(t, err)
	assert.Equal(t, 3, recommender.GetItemCount())
	assert.Greater(t, recommender.GetFeatureCount(), 0)
	assert.NotNil(t, recommender.matrix)
}

func TestTFIDFRecommender_GetSimilar_ModelNotBuilt(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	_, err := recommender.GetSimilar(1, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model not built")
}

func TestTFIDFRecommender_GetSimilar_ItemNotFound(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action adventure"},
		{ID: 2, FeatureText: "action hero"},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	_, err = recommender.GetSimilar(999, 10)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "item 999 not found in model")
}

func TestTFIDFRecommender_GetSimilar_ReturnsSimilarItems(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	// Create items where 1 and 2 share terms (action, adventure, hero)
	// Item 3 shares "action" but is otherwise different
	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action adventure hero explosion"},
		{ID: 2, FeatureText: "action adventure hero sword"},
		{ID: 3, FeatureText: "action romance comedy drama"},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	similar, err := recommender.GetSimilar(1, 10)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(similar), 1, "Should have at least one similar item")

	// Item 2 should be most similar to item 1 (shares action, adventure, hero)
	assert.Equal(t, int64(2), similar[0].ID, "Action movie 2 should be most similar to action movie 1")
	assert.Greater(t, similar[0].SimilarityScore, 0.0)
}

func TestTFIDFRecommender_GetSimilar_RespectsTopN(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action adventure hero"},
		{ID: 2, FeatureText: "action adventure villain"},
		{ID: 3, FeatureText: "action hero sword"},
		{ID: 4, FeatureText: "action villain explosion"},
		{ID: 5, FeatureText: "romance love drama"},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	similar, err := recommender.GetSimilar(1, 2)

	require.NoError(t, err)
	assert.LessOrEqual(t, len(similar), 2)
}

func TestTFIDFRecommender_GetSimilar_SortedByScore(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action adventure hero explosion"},
		{ID: 2, FeatureText: "action adventure hero sword"},     // Most similar to 1
		{ID: 3, FeatureText: "action comedy"},                   // Somewhat similar
		{ID: 4, FeatureText: "romance love drama wedding kiss"}, // Least similar
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	similar, err := recommender.GetSimilar(1, 10)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(similar), 2)

	// Verify sorted in descending order
	for i := 1; i < len(similar); i++ {
		assert.GreaterOrEqual(t, similar[i-1].SimilarityScore, similar[i].SimilarityScore,
			"Results should be sorted by similarity score descending")
	}
}

func TestTFIDFRecommender_GetSimilar_ExcludesSelf(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action adventure hero"},
		{ID: 2, FeatureText: "action adventure villain"},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	similar, err := recommender.GetSimilar(1, 10)

	require.NoError(t, err)
	for _, s := range similar {
		assert.NotEqual(t, int64(1), s.ID, "Should not include self in similar items")
	}
}

func TestTFIDFRecommender_GetMetadata(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	items := []domain.ContentFeatures{
		{
			ID:          1,
			FeatureText: "action adventure",
			Metadata: domain.ContentMetadata{
				Title:       "Test Movie",
				Year:        "2024",
				VoteAverage: 8.5,
			},
		},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	metadata, ok := recommender.GetMetadata(1)

	assert.True(t, ok)
	assert.Equal(t, "Test Movie", metadata.Title)
	assert.Equal(t, "2024", metadata.Year)
	assert.Equal(t, 8.5, metadata.VoteAverage)
}

func TestTFIDFRecommender_GetMetadata_NotFound(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action adventure"},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	_, ok := recommender.GetMetadata(999)

	assert.False(t, ok)
}

func TestTFIDFRecommender_GetItemCount(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	assert.Equal(t, 0, recommender.GetItemCount())

	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action"},
		{ID: 2, FeatureText: "adventure"},
		{ID: 3, FeatureText: "comedy"},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	assert.Equal(t, 3, recommender.GetItemCount())
}

func TestTFIDFRecommender_GetFeatureCount(t *testing.T) {
	recommender := NewTFIDFRecommender("movie")

	assert.Equal(t, 0, recommender.GetFeatureCount())

	items := []domain.ContentFeatures{
		{ID: 1, FeatureText: "action adventure hero"},
		{ID: 2, FeatureText: "action comedy villain"},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	// Should have unique terms: action, adventure, hero, comedy, villain = 5
	assert.Equal(t, 5, recommender.GetFeatureCount())
}

func TestTFIDFRecommender_ManyTermsFewDocuments(t *testing.T) {
	// This test verifies the fix for the panic where vocabulary size > document count
	// The original bug was iterating over rows (terms) instead of columns (documents)
	recommender := NewTFIDFRecommender("movie")

	// Create 3 documents with many unique terms
	// vocabulary > document count to trigger the original bug
	// Item 1 and 2 share several terms, item 3 is different
	items := []domain.ContentFeatures{
		{
			ID:          1,
			FeatureText: "action adventure hero explosion car chase thriller suspense",
		},
		{
			ID:          2,
			FeatureText: "action adventure hero sword fight battle warrior combat",
		},
		{
			ID:          3,
			FeatureText: "romance love comedy wedding dance music happy sweet",
		},
	}
	err := recommender.Build(items)
	require.NoError(t, err)

	// Vocabulary (features) should be larger than document count
	assert.Equal(t, 3, recommender.GetItemCount())
	assert.Greater(t, recommender.GetFeatureCount(), recommender.GetItemCount(),
		"This test requires more features than documents to verify the fix")

	// This should not panic (the original bug would panic here)
	similar, err := recommender.GetSimilar(1, 10)

	require.NoError(t, err)
	// Items 1 and 2 share "action", "adventure", "hero" so they should be similar
	require.GreaterOrEqual(t, len(similar), 1, "Should have at least one similar item")
	assert.Equal(t, int64(2), similar[0].ID, "Item 2 should be most similar to item 1")
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 2, 3},
			b:        []float64{1, 2, 3},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0},
			b:        []float64{-1, 0},
			expected: -1.0,
		},
		{
			name:     "different lengths",
			a:        []float64{1, 2},
			b:        []float64{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "zero vector a",
			a:        []float64{0, 0, 0},
			b:        []float64{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "zero vector b",
			a:        []float64{1, 2, 3},
			b:        []float64{0, 0, 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestSimpleTokeniser_Tokenise(t *testing.T) {
	tokeniser := &simpleTokeniser{}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple tokens",
			input:    "action adventure hero",
			expected: []string{"action", "adventure", "hero"},
		},
		{
			name:     "mixed case",
			input:    "Action ADVENTURE Hero",
			expected: []string{"action", "adventure", "hero"},
		},
		{
			name:     "extra whitespace",
			input:    "  action   adventure  hero  ",
			expected: []string{"action", "adventure", "hero"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single token",
			input:    "action",
			expected: []string{"action"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokeniser.Tokenise(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleTokeniser_ForEachIn(t *testing.T) {
	tokeniser := &simpleTokeniser{}

	var tokens []string
	tokeniser.ForEachIn("action adventure hero", func(token string) {
		tokens = append(tokens, token)
	})

	assert.Equal(t, []string{"action", "adventure", "hero"}, tokens)
}
