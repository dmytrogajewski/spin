// Package vector provides shared mathematical utilities for vector operations.
package vector

import (
	"math"
)

// Float is a constraint for floating-point types supported by vector operations.
type Float interface {
	~float32 | ~float64
}

// CosineSimilarity calculates the cosine similarity between two vectors.
// Returns 0.0 if vectors have different lengths, are empty, or either has zero magnitude.
// Result is in the range [-1.0, 1.0].
func CosineSimilarity[Num Float](a, b []Num) float64 {
	magA := Magnitude(a)
	magB := Magnitude(b)

	if magA == 0 || magB == 0 {
		return 0.0
	}

	return DotProduct(a, b) / (magA * magB)
}

// DotProduct calculates the dot product of two vectors.
// Returns 0.0 if vectors have different lengths or are empty.
func DotProduct[Num Float](a, b []Num) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var result float64

	for i := range a {
		result += float64(a[i]) * float64(b[i])
	}

	return result
}

// Magnitude calculates the Euclidean magnitude (L2 norm) of a vector.
// Returns 0.0 for empty vectors.
func Magnitude[Num Float](vec []Num) float64 {
	if len(vec) == 0 {
		return 0.0
	}

	var sum float64

	for _, val := range vec {
		fv := float64(val)
		sum += fv * fv
	}

	return math.Sqrt(sum)
}
