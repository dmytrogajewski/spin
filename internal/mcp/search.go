package mcp

import (
	"strings"

	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/pkg/alg/search"
)

const (
	defaultSearchMinScore  = 0.3
	descriptionScoreWeight = 0.6
)

// SearchOptions configures search behavior.
type SearchOptions struct {
	// FuzzyMatch enables fuzzy string matching (default: true).
	FuzzyMatch bool

	// MatchDescription searches tool descriptions (default: true).
	MatchDescription bool

	// MinScore is the minimum relevance score (0.0-1.0, default: 0.3).
	MinScore float64
}

// DefaultSearchOptions returns the default search options.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		FuzzyMatch:       true,
		MatchDescription: true,
		MinScore:         defaultSearchMinScore,
	}
}

// SearchTools searches through tools and returns matches sorted by relevance.
func SearchTools(toolList []tools.Tool, query string, maxResults int, opts SearchOptions) []tools.Tool {
	if query == "" {
		if maxResults > 0 && len(toolList) > maxResults {
			return toolList[:maxResults]
		}

		return toolList
	}

	query = strings.ToLower(strings.TrimSpace(query))

	return search.RankedSearch(toolList, func(tool tools.Tool) float64 {
		return scoreTool(tool, query, opts)
	}, opts.MinScore, maxResults)
}

// scoreTool calculates a relevance score for a tool given a query.
func scoreTool(t tools.Tool, query string, opts SearchOptions) float64 {
	name := strings.ToLower(t.Name())
	desc := strings.ToLower(t.Description())

	var maxScore float64

	// Score against name.
	nameScore := search.ScoreString(name, query, opts.FuzzyMatch)
	if nameScore > maxScore {
		maxScore = nameScore
	}

	// Score against description (with penalty).
	if opts.MatchDescription && desc != "" {
		descScore := search.ScoreString(desc, query, opts.FuzzyMatch) * descriptionScoreWeight // Description matches worth less.
		if descScore > maxScore {
			maxScore = descScore
		}
	}

	return maxScore
}

