package classify

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type category int

const (
	catUnknown category = iota
	catAnimal
	catPlant
	catMineral
)

func newTestClassifier() *Classifier[string, category] {
	c := New[string, category](catUnknown)
	c.AddRule("animal", func(s string) bool {
		return s == "dog" || s == "cat" || s == "fish"
	}, catAnimal)
	c.AddRule("plant", func(s string) bool {
		return s == "tree" || s == "flower"
	}, catPlant)
	c.AddRule("mineral", func(s string) bool {
		return s == "rock" || s == "diamond"
	}, catMineral)

	return c
}

func TestClassifier_Classify(t *testing.T) {
	t.Parallel()

	c := newTestClassifier()

	tests := []struct {
		name  string
		input string
		want  category
	}{
		{name: "animal", input: "dog", want: catAnimal},
		{name: "plant", input: "tree", want: catPlant},
		{name: "mineral", input: "rock", want: catMineral},
		{name: "unknown", input: "water", want: catUnknown},
		{name: "empty", input: "", want: catUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := c.Classify(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClassifier_FirstMatchWins(t *testing.T) {
	t.Parallel()

	c := New[string, string]("none")
	c.AddRule("all", func(_ string) bool { return true }, "first")
	c.AddRule("also_all", func(_ string) bool { return true }, "second")

	got := c.Classify("anything")
	require.Equal(t, "first", got)
}

func TestClassifier_ClassifyBatch(t *testing.T) {
	t.Parallel()

	c := newTestClassifier()

	results := c.ClassifyBatch([]string{"dog", "tree", "water", "diamond"})
	require.Equal(t, []category{catAnimal, catPlant, catUnknown, catMineral}, results)
}

func TestClassifier_EmptyRules(t *testing.T) {
	t.Parallel()

	c := New[string, string]("default")
	require.Equal(t, "default", c.Classify("anything"))
	require.Equal(t, 0, c.RuleCount())
}

func TestClassifier_RuleCount(t *testing.T) {
	t.Parallel()

	c := newTestClassifier()
	require.Equal(t, 3, c.RuleCount())
}

func TestClassifier_ClassifyBatchEmpty(t *testing.T) {
	t.Parallel()

	c := newTestClassifier()

	results := c.ClassifyBatch([]string{})
	require.Empty(t, results)
}

func TestClassifier_IntType(t *testing.T) {
	t.Parallel()

	c := New[int, string]("other")
	c.AddRule("even", func(n int) bool { return n%2 == 0 }, "even")
	c.AddRule("odd", func(n int) bool { return n%2 != 0 }, "odd")

	require.Equal(t, "even", c.Classify(4))
	require.Equal(t, "odd", c.Classify(7))
}
