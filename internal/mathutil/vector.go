// Package mathutil provides shared mathematical utilities for vector operations.
package mathutil

import "math"

// CosineSimilarity calculates the cosine similarity between two float32 vectors.
// Returns 0.0 if vectors have different lengths, are empty, or either has zero magnitude.
// Result is in the range [-1.0, 1.0].
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dotProduct += ai * bi
		normA += ai * ai
		normB += bi * bi
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// DotProduct calculates the dot product of two float32 vectors.
// Returns 0.0 if vectors have different lengths or are empty.
func DotProduct(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var result float64

	for i := range a {
		result += float64(a[i]) * float64(b[i])
	}

	return result
}

// Magnitude calculates the Euclidean magnitude (L2 norm) of a float32 vector.
// Returns 0.0 for empty vectors.
func Magnitude(v []float32) float64 {
	if len(v) == 0 {
		return 0.0
	}

	var sum float64

	for _, val := range v {
		fv := float64(val)
		sum += fv * fv
	}

	return math.Sqrt(sum)
}
