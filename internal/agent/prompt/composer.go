package prompt

import (
	"sort"
	"strings"

	"github.com/dmytrogajewski/spin/internal/agent"
)

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
func (c *Composer) AddSection(s Section) {
	c.sections = append(c.sections, s)
}

// SetVar registers a variable for ${VAR} substitution.
func (c *Composer) SetVar(name, value string) {
	c.vars[name] = value
}

// Compose evaluates conditions, sorts by priority, resolves variables, and joins.
func (c *Composer) Compose(env *agent.Environment) string {
	active := c.activeSections(env)
	if len(active) == 0 {
		return ""
	}

	sort.Slice(active, func(i, j int) bool {
		return active[i].Priority < active[j].Priority
	})

	var sb strings.Builder

	for i, s := range active {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		sb.WriteString(c.resolve(s.Template))
	}

	return sb.String()
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

	for _, s := range active {
		resolved := c.resolve(s.Template)
		if s.Cacheable {
			stableParts = append(stableParts, resolved)
		} else {
			dynamicParts = append(dynamicParts, resolved)
		}
	}

	return strings.Join(stableParts, "\n\n"), strings.Join(dynamicParts, "\n\n")
}

// activeSections returns sections whose conditions pass for the given environment.
func (c *Composer) activeSections(env *agent.Environment) []Section {
	active := make([]Section, 0, len(c.sections))

	for _, s := range c.sections {
		if s.Condition == nil || s.Condition(env) {
			active = append(active, s)
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
