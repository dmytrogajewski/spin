package ds

// Pass represents a single matching strategy in a [Chain].
type Pass[Input, Output any] struct {
	// Name identifies this pass for diagnostics.
	Name string
	// Find runs the matching strategy on the input.
	Find func(Input) []Output
}

// Chain runs passes in order, returning results from the first successful pass.
type Chain[Input, Output any] struct {
	passes []Pass[Input, Output]
}

// NewChain creates a chain with the given passes.
func NewChain[Input, Output any](passes ...Pass[Input, Output]) *Chain[Input, Output] {
	return &Chain[Input, Output]{passes: passes}
}

// Find runs passes in order, returning the first result from the first
// successful pass. Returns (nil, "") if no pass matches.
func (c *Chain[Input, Output]) Find(input Input) (result *Output, passName string) {
	for _, pass := range c.passes {
		matches := pass.Find(input)
		if len(matches) > 0 {
			return &matches[0], pass.Name
		}
	}

	return nil, ""
}

// FindAll runs passes in order, returning all results from the first
// successful pass. Returns (nil, "") if no pass matches.
func (c *Chain[Input, Output]) FindAll(input Input) (results []Output, passName string) {
	for _, pass := range c.passes {
		matches := pass.Find(input)
		if len(matches) > 0 {
			return matches, pass.Name
		}
	}

	return nil, ""
}
