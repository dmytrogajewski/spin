package fuzzy

import "github.com/dmytrogajewski/spin/pkg/alg/ds"

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

// matchInput bundles the two string parameters for the generic chain.
type matchInput struct {
	fileContent string
	oldContent  string
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
	inner *ds.Chain[matchInput, MatchResult]
}

// NewChain creates a chain with the given passes.
func NewChain(passes ...Pass) *Chain {
	adapted := make([]ds.Pass[matchInput, MatchResult], len(passes))

	for idx, pass := range passes {
		fn := pass.Find // Capture for closure.

		adapted[idx] = ds.Pass[matchInput, MatchResult]{
			Name: pass.Name,
			Find: func(input matchInput) []MatchResult {
				return fn(input.fileContent, input.oldContent)
			},
		}
	}

	return &Chain{inner: ds.NewChain(adapted...)}
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
	result, passName := c.inner.Find(matchInput{
		fileContent: fileContent,
		oldContent:  oldContent,
	})

	if result == nil {
		return nil
	}

	result.PassName = passName

	return result
}

// FindAll runs passes in order, returning all matches from the first successful pass.
// Returns nil if no pass matches.
func (c *Chain) FindAll(fileContent, oldContent string) []MatchResult {
	results, passName := c.inner.FindAll(matchInput{
		fileContent: fileContent,
		oldContent:  oldContent,
	})

	for idx := range results {
		results[idx].PassName = passName
	}

	return results
}
