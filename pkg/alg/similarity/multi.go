package similarity

// Strategy is a function that computes similarity between two strings.
// Returns a value in [0.0, 1.0] where 1.0 means identical.
type Strategy func(a, b string) float64

// MultiStrategySimilarity runs multiple similarity strategies and returns the
// maximum score. Returns 0 if no strategies are provided.
func MultiStrategySimilarity(first, second string, strategies ...Strategy) float64 {
	var best float64

	for _, strategy := range strategies {
		score := strategy(first, second)
		if score > best {
			best = score
		}
	}

	return best
}
