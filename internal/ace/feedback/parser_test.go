package feedback

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegexParser(t *testing.T) {
	parser := NewRegexParser()
	require.NotNil(t, parser)
	assert.NotNil(t, parser.helpfulPattern)
	assert.NotNil(t, parser.harmfulPattern)
	assert.NotNil(t, parser.explanationPattern)
}

func TestRegexParser_ParseEmpty(t *testing.T) {
	parser := NewRegexParser()
	feedback, err := parser.Parse("")

	require.NoError(t, err)
	assert.Empty(t, feedback.HelpfulBullets)
	assert.Empty(t, feedback.HarmfulBullets)
	assert.Empty(t, feedback.Explanation)
}

func TestRegexParser_ParseHelpful(t *testing.T) {
	parser := NewRegexParser()
	response := `The answer is 42.

HELPFUL: [B0, B1, B3]`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Len(t, feedback.HelpfulBullets, 3)
	assert.Equal(t, "B0", feedback.HelpfulBullets[0])
	assert.Equal(t, "B1", feedback.HelpfulBullets[1])
	assert.Equal(t, "B3", feedback.HelpfulBullets[2])
	assert.Empty(t, feedback.HarmfulBullets)
}

func TestRegexParser_ParseHarmful(t *testing.T) {
	parser := NewRegexParser()
	response := `The answer is incorrect.

HARMFUL: [B2]`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Empty(t, feedback.HelpfulBullets)
	assert.Len(t, feedback.HarmfulBullets, 1)
	assert.Equal(t, "B2", feedback.HarmfulBullets[0])
}

func TestRegexParser_ParseBoth(t *testing.T) {
	parser := NewRegexParser()
	response := `Solution here.

HELPFUL: [B0, B1]
HARMFUL: [B2, B3]`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Len(t, feedback.HelpfulBullets, 2)
	assert.Len(t, feedback.HarmfulBullets, 2)
	assert.Equal(t, "B0", feedback.HelpfulBullets[0])
	assert.Equal(t, "B1", feedback.HelpfulBullets[1])
	assert.Equal(t, "B2", feedback.HarmfulBullets[0])
	assert.Equal(t, "B3", feedback.HarmfulBullets[1])
}

func TestRegexParser_ParseWithExplanation(t *testing.T) {
	parser := NewRegexParser()
	response := `Answer: 42

HELPFUL: [B0]
HARMFUL: [B1]
EXPLANATION: B0 provided the correct formula while B1 was outdated`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Len(t, feedback.HelpfulBullets, 1)
	assert.Len(t, feedback.HarmfulBullets, 1)
	assert.Equal(t, "B0 provided the correct formula while B1 was outdated", feedback.Explanation)
}

func TestRegexParser_ParseNoMarkers(t *testing.T) {
	parser := NewRegexParser()
	response := `Just a regular response with no markers.`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Empty(t, feedback.HelpfulBullets)
	assert.Empty(t, feedback.HarmfulBullets)
	assert.Empty(t, feedback.Explanation)
}

func TestRegexParser_ParseEmptyBrackets(t *testing.T) {
	parser := NewRegexParser()
	response := `HELPFUL: []
HARMFUL: []`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Empty(t, feedback.HelpfulBullets)
	assert.Empty(t, feedback.HarmfulBullets)
}

func TestRegexParser_ParseSingleMarker(t *testing.T) {
	parser := NewRegexParser()
	response := `HELPFUL: [B5]`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Len(t, feedback.HelpfulBullets, 1)
	assert.Equal(t, "B5", feedback.HelpfulBullets[0])
}

func TestRegexParser_ParseWhitespace(t *testing.T) {
	parser := NewRegexParser()
	response := `HELPFUL: [ B0 , B1 , B2 ]
HARMFUL: [ B3 ]`

	feedback, err := parser.Parse(response)

	require.NoError(t, err)
	assert.Len(t, feedback.HelpfulBullets, 3)
	assert.Equal(t, "B0", feedback.HelpfulBullets[0])
	assert.Equal(t, "B1", feedback.HelpfulBullets[1])
	assert.Equal(t, "B2", feedback.HelpfulBullets[2])
	assert.Len(t, feedback.HarmfulBullets, 1)
	assert.Equal(t, "B3", feedback.HarmfulBullets[0])
}

func TestParseBulletMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", []string{}},
		{"single", "B0", []string{"B0"}},
		{"multiple", "B0, B1, B2", []string{"B0", "B1", "B2"}},
		{"whitespace", " B0 , B1 , B2 ", []string{"B0", "B1", "B2"}},
		{"no spaces", "B0,B1,B2", []string{"B0", "B1", "B2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBulletMarkers(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
