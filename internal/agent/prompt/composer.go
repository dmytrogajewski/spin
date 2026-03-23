package prompt

import (
	"sort"
	"strings"

	"github.com/dmytrogajewski/spin/internal/agent"
)

// sectionSeparator joins resolved sections.
const sectionSeparator = "\n\n"

// Composer assembles the system prompt from registered sections.
// Sections are evaluated against the environment, sorted by priority,
// and joined with variable substitution applied.
type Composer struct {
	sections []Section
	vars     map[string]string
}

// NewComposer creates an empty Composer.
func NewComposer() *Composer {
	return &Composer{
		vars: make(map[string]string),
	}
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
// Equivalent to joining the two parts from [Composer.ComposeTwoPart].
func (c *Composer) Compose(env *agent.Environment) string {
	stable, dynamic := c.ComposeTwoPart(env)

	if dynamic == "" {
		return stable
	}

	if stable == "" {
		return dynamic
	}

	return stable + sectionSeparator + dynamic
}

// ComposeTwoPart splits the prompt into stable (cacheable) and dynamic parts.
// Stable sections are joined first; dynamic sections follow.
func (c *Composer) ComposeTwoPart(env *agent.Environment) (stable, dynamic string) {
	active := c.activeSections(env)
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

// activeSections returns sections whose conditions pass for the given environment.
func (c *Composer) activeSections(env *agent.Environment) []Section {
	active := make([]Section, 0, len(c.sections))

	for _, sec := range c.sections {
		if sec.Condition == nil || sec.Condition(env) {
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
