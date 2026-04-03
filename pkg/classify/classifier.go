// Package classify provides a generic rule-based classification engine.
//
// It replaces hand-rolled switch/map classification logic across the codebase
// with a composable, testable classifier that supports ordered rules and defaults.
package classify

// Rule defines a single classification rule with a match predicate.
type Rule[T any, R any] struct {
	// Name identifies this rule for debugging.
	Name string
	// Match returns true if the input matches this rule.
	Match func(T) bool
	// Label is the classification result when this rule matches.
	Label R
}

// Classifier classifies inputs of type T into labels of type R.
// Rules are evaluated in registration order; first match wins.
type Classifier[T any, R any] struct {
	rules      []Rule[T, R]
	defaultVal R
}

// New creates a Classifier with the given default label.
// The default is returned when no rule matches.
func New[T any, R any](defaultVal R) *Classifier[T, R] {
	return &Classifier[T, R]{
		defaultVal: defaultVal,
	}
}

// AddRule appends a classification rule.
// Rules are evaluated in the order they are added.
func (c *Classifier[T, R]) AddRule(name string, match func(T) bool, label R) *Classifier[T, R] {
	c.rules = append(c.rules, Rule[T, R]{
		Name:  name,
		Match: match,
		Label: label,
	})

	return c
}

// Classify returns the label for the first matching rule, or the default.
func (c *Classifier[T, R]) Classify(input T) R {
	for _, rule := range c.rules {
		if rule.Match(input) {
			return rule.Label
		}
	}

	return c.defaultVal
}

// ClassifyBatch classifies multiple inputs.
func (c *Classifier[T, R]) ClassifyBatch(inputs []T) []R {
	results := make([]R, len(inputs))

	for i, input := range inputs {
		results[i] = c.Classify(input)
	}

	return results
}

// RuleCount returns the number of registered rules.
func (c *Classifier[T, R]) RuleCount() int {
	return len(c.rules)
}
