package cycle

import (
	"testing"
)

const (
	testSimilarityInput = "hello world test"
)

func TestCalculateSimilarity_Identical(t *testing.T) {
	t.Parallel()

	a := testSimilarityInput
	b := testSimilarityInput

	similarity := calculateSimilarity(a, b)
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for identical strings, got %f", similarity)
	}
}

func TestCalculateSimilarity_CompletelyDifferent(t *testing.T) {
	t.Parallel()

	a := "hello world"
	b := "goodbye universe"

	similarity := calculateSimilarity(a, b)
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for completely different strings, got %f", similarity)
	}
}

func TestCalculateSimilarity_PartialOverlap(t *testing.T) {
	t.Parallel()

	a := "the quick brown fox jumps"
	b := "the lazy brown dog sleeps"

	// Words: the, quick, brown, fox, jumps
	// vs: the, lazy, brown, dog, sleeps
	// Intersection: the, brown (2)
	// Union: the, quick, brown, fox, jumps, lazy, dog, sleeps (8)
	// Similarity: 2/8 = 0.25.

	expected := 2.0 / 8.0
	similarity := calculateSimilarity(a, b)

	if abs(similarity-expected) > 0.001 {
		t.Errorf("Expected similarity %f, got %f", expected, similarity)
	}
}

func TestCalculateSimilarity_CaseInsensitive(t *testing.T) {
	t.Parallel()

	a := "Hello World Test"
	b := testSimilarityInput

	similarity := calculateSimilarity(a, b)
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for case variations, got %f", similarity)
	}
}

func TestCalculateSimilarity_Punctuation(t *testing.T) {
	t.Parallel()

	a := "Hello, world! How are you?"
	b := "Hello world How are you"

	similarity := calculateSimilarity(a, b)
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 ignoring punctuation, got %f", similarity)
	}
}

func TestCalculateSimilarity_EmptyStrings(t *testing.T) {
	t.Parallel()

	// Both empty.
	similarity := calculateSimilarity("", "")
	if similarity != 1.0 {
		t.Errorf("Expected similarity 1.0 for both empty strings, got %f", similarity)
	}

	// One empty, one not.
	similarity = calculateSimilarity("", "hello world")
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for one empty string, got %f", similarity)
	}

	similarity = calculateSimilarity("hello world", "")
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for one empty string, got %f", similarity)
	}
}

func TestCalculateSimilarity_ShortWords(t *testing.T) {
	t.Parallel()

	a := "a is test"
	b := "this is a test"

	// After filtering short words (<3 chars): test
	// vs: this, test
	// Intersection: test (1)
	// Union: test, this (2)
	// Similarity: 1/2 = 0.5.

	expected := 1.0 / 2.0
	similarity := calculateSimilarity(a, b)

	if abs(similarity-expected) > 0.001 {
		t.Errorf("Expected similarity %f, got %f", expected, similarity)
	}
}

func TestCalculateSimilarity_SemanticSimilarity(t *testing.T) {
	t.Parallel()

	a := "the cat sat on the mat"
	b := "feline rested on rug"

	// After processing: cat, sat, mat
	// vs: feline, rested, rug
	// No common words, so similarity should be 0.

	similarity := calculateSimilarity(a, b)
	if similarity != 0.0 {
		t.Errorf("Expected similarity 0.0 for semantically similar but lexically different text, got %f", similarity)
	}
}

func TestExtractWords(t *testing.T) {
	t.Parallel()

	text := "Hello, world! This is a test... with multiple sentences."
	words := extractWords(text)

	expected := []string{"hello", "world", "this", "test", "with", "multiple", "sentences"}
	if len(words) != len(expected) {
		t.Errorf("Expected %d words, got %d", len(expected), len(words))
	}

	// Check that short words are filtered out.
	for _, word := range words {
		if len(word) < 3 {
			t.Errorf("Short word '%s' should have been filtered out", word)
		}
	}
}

func TestExtractWords_Empty(t *testing.T) {
	t.Parallel()

	words := extractWords("")
	if len(words) != 0 {
		t.Errorf("Expected 0 words for empty string, got %d", len(words))
	}
}

func TestExtractWords_Punctuation(t *testing.T) {
	t.Parallel()

	text := "test,word!with@punctuation#and$symbols%"
	words := extractWords(text)

	// Should extract at least "test", "word", "with", "punctuation", "and"
	// Note: "symbols" might be filtered out as too short.
	expectedMin := 3 // At least these words should be extracted.
	if len(words) < expectedMin {
		t.Errorf("Expected at least %d words, got %d", expectedMin, len(words))
	}
}

// Helper function for floating point comparison.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}

	return x
}
