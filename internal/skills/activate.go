package skills

import (
	"fmt"
	"strings"
)

// Activation is the result of loading one skill body into the current turn.
// References under scripts/, references/, and assets/ are not opened here.
type Activation struct {
	// Name is the skill identifier.
	Name string
	// Body is the Markdown after the frontmatter fence.
	Body string
	// Root is the skill directory (parent of SKILL.md).
	Root string
	// Source is the catalog source tag of the winning entry.
	Source Source
	// AllowedTools is the experimental frontmatter allowlist when present.
	AllowedTools string
}

// UnknownSkillError is returned when the requested name is not in the catalog.
// Catalog lists names only — never bodies.
type UnknownSkillError struct {
	// Name is the requested skill identifier.
	Name string
	// Catalog is the list of available skill names.
	Catalog []string
}

// Error implements error.
func (e *UnknownSkillError) Error() string {
	if len(e.Catalog) == 0 {
		return fmt.Sprintf("unknown skill %q (available: none): %v", e.Name, ErrUnknownSkill)
	}

	return fmt.Sprintf("unknown skill %q (available: %s): %v", e.Name, strings.Join(e.Catalog, ", "), ErrUnknownSkill)
}

// Unwrap returns ErrUnknownSkill.
func (e *UnknownSkillError) Unwrap() error {
	return ErrUnknownSkill
}

// Activate looks up name in catalog, parses that skill's SKILL.md, and
// returns the body plus root. It does not read scripts/, references/, or assets/.
func Activate(catalog Catalog, name string) (Activation, error) {
	if name == "" {
		return Activation{}, ErrEmptyName
	}

	entry, ok := lookupEntry(catalog, name)
	if !ok {
		return Activation{}, unknownSkill(catalog, name)
	}

	skill, err := Parse(entry.Location)
	if err != nil {
		return Activation{}, err
	}

	return Activation{
		Name:         skill.Name,
		Body:         skill.Body,
		Root:         skill.Dir,
		Source:       entry.Source,
		AllowedTools: skill.AllowedTools,
	}, nil
}

func lookupEntry(catalog Catalog, name string) (Entry, bool) {
	for _, entry := range catalog {
		if entry.Name == name {
			return entry, true
		}
	}

	return Entry{}, false
}

func unknownSkill(catalog Catalog, name string) error {
	names := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		names = append(names, entry.Name)
	}

	return &UnknownSkillError{Name: name, Catalog: names}
}
