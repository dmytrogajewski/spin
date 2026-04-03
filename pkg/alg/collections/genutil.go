// Package collections provides generic, zero-dependency collection helpers
// that consolidate duplicated slice/map operations across the codebase.
package collections

// TailN returns the last n elements of a slice.
// Returns nil if the input is nil/empty or n <= 0.
// Returns a copy of the full slice if n >= len(input).
func TailN[Elem any](input []Elem, n int) []Elem {
	if n <= 0 || len(input) == 0 {
		return nil
	}

	if n >= len(input) {
		result := make([]Elem, len(input))
		copy(result, input)

		return result
	}

	result := make([]Elem, n)
	copy(result, input[len(input)-n:])

	return result
}

// ToSet converts a slice to a membership map.
// Returns nil if items is nil.
// Returns an empty map if items is empty.
func ToSet[Key comparable](items []Key) map[Key]bool {
	if items == nil {
		return nil
	}

	result := make(map[Key]bool, len(items))

	for _, item := range items {
		result[item] = true
	}

	return result
}

// AllSame returns true if all elements yield the same value via the key extractor.
// Returns true for empty or single-element slices (vacuous truth).
func AllSame[Elem any, K comparable](items []Elem, key func(Elem) K) bool {
	if len(items) <= 1 {
		return true
	}

	first := key(items[0])

	for _, item := range items[1:] {
		if key(item) != first {
			return false
		}
	}

	return true
}

// Float is a constraint for floating-point types.
type Float interface {
	~float32 | ~float64
}

// Mean returns the arithmetic mean of a float slice.
// Returns the zero value for empty slices.
func Mean[Num Float](values []Num) Num {
	if len(values) == 0 {
		var zero Num

		return zero
	}

	var sum Num

	for _, val := range values {
		sum += val
	}

	return sum / Num(len(values))
}

// Ratio computes after/before as a float64.
// Returns 0 if before is zero (avoids division by zero).
func Ratio(before, after int) float64 {
	if before == 0 {
		return 0
	}

	return float64(after) / float64(before)
}

// EnsureMap returns m if non-nil, otherwise returns a new empty map.
func EnsureMap[Key comparable, Val any](m map[Key]Val) map[Key]Val {
	if m == nil {
		return make(map[Key]Val)
	}

	return m
}

// Ptr returns a pointer to the given value.
func Ptr[Elem any](v Elem) *Elem {
	return &v
}

// FirstNonZero returns the first non-zero value from the arguments.
// Returns the zero value of T if no non-zero argument is found.
func FirstNonZero[T comparable](values ...T) T {
	var zero T

	for _, v := range values {
		if v != zero {
			return v
		}
	}

	return zero
}
