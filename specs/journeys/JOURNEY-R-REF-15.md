# JOURNEY-R-REF-15: Add MultiStrategySimilarity and FindSimilarPairs

**Roadmap Item:** R-REF-15

## Summary

Add two functions to `pkg/alg/similarity`:

1. `MultiStrategySimilarity(a, b string, strategies ...func(string, string) float64) float64` — runs multiple similarity strategies, returns the maximum score.
2. `FindSimilarPairs[T any](items []T, getText func(T) string, threshold float64, sim func(string, string) float64) []Pair[T]` — O(n²) pairwise comparison returning pairs above threshold.

## Design

- `MultiStrategySimilarity` takes variadic strategy functions, returns max score. If no strategies provided, returns 0.
- `FindSimilarPairs` is generic over item type, uses a text extractor and similarity function. Returns `[]Pair[T]` where Pair has Left, Right indices and Score.
- Both are pure functions, no side effects.

## Acceptance Criteria

- [ ] `multi.go` with `MultiStrategySimilarity`
- [ ] `pairs.go` with `FindSimilarPairs[T]` and `Pair[T]`
- [ ] Tests with edge cases
- [ ] `go test ./pkg/alg/similarity/...` passes

## Implementation

- **Created:** `pkg/alg/similarity/multi.go`
- **Created:** `pkg/alg/similarity/pairs.go`
- **Modified:** `pkg/alg/similarity/similarity_test.go`
