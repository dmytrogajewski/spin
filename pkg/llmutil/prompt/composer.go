// Package prompt provides a composable system prompt builder with
// priority ordering, conditional sections, and variable substitution.
package prompt

import (
	"sort"
	"strings"
)

// sectionSeparator joins resolved sections.
const sectionSeparator = "\n\n"

// Section represents a modular prompt section with conditional inclusion.
type Section struct {
	// Name identifies this section uniquely.
	Name string
	// Priority controls ordering; lower values appear earlier.
	Priority int
	// Active determines whether this section is included.
	// A nil function means the section is always included.
	Active func() bool
	// Cacheable marks this section as part of the stable prefix for prompt caching.
	Cacheable bool
	// Template is the content with optional ${VAR} placeholders.
	Template string
}

// Composer assembles a prompt from registered sections.
type Composer struct {
	sections []Section
	vars     map[string]string
}

// NewComposer creates an empty Composer.
func NewComposer() *Composer {
	return &Composer{vars: make(map[string]string)}
}

// AddSection registers a prompt section.
func (c *Composer) AddSection(sec Section) {
	c.sections = append(c.sections, sec)
}

// SetVar registers a variable for ${VAR} substitution.
func (c *Composer) SetVar(name, value string) {
	c.vars[name] = value
}

// Compose evaluates conditions, sorts by priority, resolves variables, and joins.
func (c *Composer) Compose() string {
	stable, dynamic := c.ComposeTwoPart()

	if dynamic == "" {
		return stable
	}

	if stable == "" {
		return dynamic
	}

	return stable + sectionSeparator + dynamic
}

// ComposeTwoPart splits the prompt into stable (cacheable) and dynamic parts.
func (c *Composer) ComposeTwoPart() (stable, dynamic string) {
	active := c.activeSections()
	if len(active) == 0 {
		return "", ""
	}

	sort.Slice(active, func(i, j int) bool {
		return active[i].Priority < active[j].Priority
	})

	var stableParts, dynamicParts []string

	for _, sec := range active {
		resolved := c.resolve(sec.Template)
		if sec.Cacheable {
			stableParts = append(stableParts, resolved)
		} else {
			dynamicParts = append(dynamicParts, resolved)
		}
	}

	return strings.Join(stableParts, sectionSeparator), strings.Join(dynamicParts, sectionSeparator)
}

// activeSections returns sections whose conditions pass.
func (c *Composer) activeSections() []Section {
	active := make([]Section, 0, len(c.sections))

	for _, sec := range c.sections {
		if sec.Active == nil || sec.Active() {
			active = append(active, sec)
		}
	}

	return active
}

// resolve performs ${VAR} substitution on a template string.
func (c *Composer) resolve(tmpl string) string {
	result := tmpl

	for name, value := range c.vars {
		result = strings.ReplaceAll(result, "${"+name+"}", value)
	}

	return result
}
