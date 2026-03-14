package mathutil

// Journey: specs/journeys/JOURNEY-extract-mathutil.md.

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const floatTolerance = 1e-6

func TestCosineSimilarity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float64
	}{
		{
			name: "identical unit vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{0, 1, 0},
			want: 0.0,
		},
		{
			name: "opposite vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{-1, 0, 0},
			want: -1.0,
		},
		{
			name: "zero vector a",
			a:    []float32{0, 0, 0},
			b:    []float32{1, 2, 3},
			want: 0.0,
		},
		{
			name: "zero vector b",
			a:    []float32{1, 2, 3},
			b:    []float32{0, 0, 0},
			want: 0.0,
		},
		{
			name: "both zero vectors",
			a:    []float32{0, 0, 0},
			b:    []float32{0, 0, 0},
			want: 0.0,
		},
		{
			name: "different lengths",
			a:    []float32{1, 2},
			b:    []float32{1, 2, 3},
			want: 0.0,
		},
		{
			name: "empty vectors",
			a:    []float32{},
			b:    []float32{},
			want: 0.0,
		},
		{
			name: "nil vectors",
			a:    nil,
			b:    nil,
			want: 0.0,
		},
		{
			name: "parallel non-unit vectors",
			a:    []float32{2, 0, 0},
			b:    []float32{5, 0, 0},
			want: 1.0,
		},
		{
			name: "known similarity",
			a:    []float32{1, 2, 3},
			b:    []float32{4, 5, 6},
			// dot = 4+10+18 = 32, |a| = sqrt(14), |b| = sqrt(77)
			// cos = 32 / sqrt(14*77) = 32 / sqrt(1078).
			want: 32.0 / math.Sqrt(1078.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.want, got, floatTolerance,
				"CosineSimilarity(%v, %v)", tt.a, tt.b)
		})
	}
}

func TestCosineSimilarity_Commutative(t *testing.T) {
	t.Parallel()

	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}

	assert.InDelta(t, CosineSimilarity(a, b), CosineSimilarity(b, a), floatTolerance,
		"cosine similarity must be commutative")
}

func TestDotProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float64
	}{
		{
			name: "simple dot product",
			a:    []float32{1, 2, 3},
			b:    []float32{4, 5, 6},
			want: 32.0, // 1*4 + 2*5 + 3*6 = 4+10+18.
		},
		{
			name: "orthogonal",
			a:    []float32{1, 0},
			b:    []float32{0, 1},
			want: 0.0,
		},
		{
			name: "different lengths",
			a:    []float32{1, 2},
			b:    []float32{1},
			want: 0.0,
		},
		{
			name: "empty",
			a:    []float32{},
			b:    []float32{},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DotProduct(tt.a, tt.b)
			assert.InDelta(t, tt.want, got, floatTolerance)
		})
	}
}

func TestMagnitude(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    []float32
		want float64
	}{
		{
			name: "unit vector",
			v:    []float32{1, 0, 0},
			want: 1.0,
		},
		{
			name: "3-4-5 triangle",
			v:    []float32{3, 4},
			want: 5.0,
		},
		{
			name: "zero vector",
			v:    []float32{0, 0, 0},
			want: 0.0,
		},
		{
			name: "empty",
			v:    []float32{},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Magnitude(tt.v)
			assert.InDelta(t, tt.want, got, floatTolerance)
		})
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	const vectorSize = 1536 // Typical embedding size.

	a := make([]float32, vectorSize)
	bVec := make([]float32, vectorSize)

	for i := range vectorSize {
		a[i] = float32(i) * 0.001
		bVec[i] = float32(vectorSize-i) * 0.001
	}

	b.ResetTimer()

	for range b.N {
		_ = CosineSimilarity(a, bVec)
	}
}

func TestCosineSimilarity_SelfSimilarityIsOne(t *testing.T) {
	t.Parallel()

	v := []float32{0.5, -0.3, 0.8, 1.2}
	got := CosineSimilarity(v, v)

	require.InDelta(t, 1.0, got, floatTolerance,
		"self-similarity must be 1.0")
}
