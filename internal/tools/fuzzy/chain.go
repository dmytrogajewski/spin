package fuzzy

// MatchResult holds the result of a fuzzy match.
type MatchResult struct {
	// Start is the byte offset in file content where the match begins.
	Start int
	// End is the byte offset end (exclusive).
	End int
	// Original is the actual matched substring from the file.
	Original string
	// PassName is the name of the pass that produced this match.
	PassName string
}

// PassFunc attempts to find oldContent in fileContent.
// Returns all match locations found, or nil if no match.
type PassFunc func(fileContent, oldContent string) []MatchResult

// Pass represents a single matching strategy in the chain.
type Pass struct {
	// Name identifies this pass for diagnostics.
	Name string
	// Find locates oldContent in fileContent.
	Find PassFunc
}

// Chain runs passes in order, returning the first successful match.
type Chain struct {
	passes []Pass
}

// NewChain creates a chain with the given passes.
func NewChain(passes ...Pass) *Chain {
	return &Chain{passes: passes}
}

// DefaultChain creates the standard 9-pass fuzzy matching chain.
func DefaultChain() *Chain {
	return NewChain(
		Pass{Name: "exact", Find: ExactFind},
		Pass{Name: "whitespace", Find: WhitespaceFind},
		Pass{Name: "indent", Find: IndentFind},
		Pass{Name: "escape", Find: EscapeFind},
		Pass{Name: "lineend", Find: LineEndFind},
		Pass{Name: "trim", Find: TrimFind},
		Pass{Name: "collapse", Find: CollapseFind},
		Pass{Name: "anchor", Find: AnchorFind},
		Pass{Name: "partial", Find: PartialFind},
	)
}

// Find runs passes in order, returning the first successful result.
// Returns nil if no pass matches.
func (c *Chain) Find(fileContent, oldContent string) *MatchResult {
	for _, pass := range c.passes {
		matches := pass.Find(fileContent, oldContent)
		if len(matches) > 0 {
			result := matches[0]
			result.PassName = pass.Name

			return &result
		}
	}

	return nil
}

// FindAll runs passes in order, returning all matches from the first successful pass.
// Returns nil if no pass matches.
func (c *Chain) FindAll(fileContent, oldContent string) []MatchResult {
	for _, pass := range c.passes {
		matches := pass.Find(fileContent, oldContent)
		if len(matches) > 0 {
			for idx := range matches {
				matches[idx].PassName = pass.Name
			}

			return matches
		}
	}

	return nil
}
