package skills

import (
	"fmt"
	"strings"
)

const (
	// EmptyCatalogMessage is the operator-facing text when Discover finds nothing.
	EmptyCatalogMessage = "No skills found."
)

// Source identifies which discovery root produced a catalog entry.
type Source string

const (
	// SourceProject is a skill from the workdir (.agents or .claude).
	SourceProject Source = "project"
	// SourceUser is a skill from the user's home (.spin or .agents).
	SourceUser Source = "user"
	// SourceBundled is a skill from the bundled / SPIN_HOME tree.
	SourceBundled Source = "bundled"

	// SourcePluginPrefix prefixes catalog sources from a loaded plugin.
	SourcePluginPrefix = "plugin:"

	// EnvSpinHome is the environment variable that points at bundled skills.
	EnvSpinHome = "SPIN_HOME"

	relAgentsSkills  = ".agents/skills"
	relClaudeSkills  = ".claude/skills"
	relSpinSkills    = ".spin/skills"
	bundledSubdir    = "skills"
	discoveryRootCap = 5
)

// PluginSource returns the catalog source tag for a plugin name.
func PluginSource(name string) Source {
	return Source(SourcePluginPrefix + name)
}

// Entry is a metadata-only catalog row (no skill body).
type Entry struct {
	// Name is the skill identifier.
	Name string
	// Description is what the skill does and when to use it.
	Description string
	// Location is the winning skill directory.
	Location string
	// Source is project, user, bundled, or plugin:<name>.
	Source Source
}

// PluginSkill is one skill contributed by a loaded plugin.
type PluginSkill struct {
	// PluginName is the plugin.json name used in source=plugin:<name>.
	PluginName string
	// Name is the skill identifier.
	Name string
	// Description is what the skill does and when to use it.
	Description string
	// Location is the skill directory.
	Location string
}

// Catalog is a deterministic list of discovered skills.
type Catalog []Entry

// Options selects the roots Discover walks. Empty paths are skipped.
type Options struct {
	// WorkDir is the project root (scanned for .agents/skills and .claude/skills).
	WorkDir string
	// HomeDir is the user home (scanned for .spin/skills and .agents/skills).
	HomeDir string
	// BundledDir is the bundled skills directory (already the skills/ folder).
	BundledDir string
	// PluginSkills are merged after project/user roots and before bundled.
	PluginSkills []PluginSkill
}

// Format prints one line per skill: name, source, description.
func Format(catalog Catalog) string {
	if len(catalog) == 0 {
		return EmptyCatalogMessage
	}

	lines := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		lines = append(lines, fmt.Sprintf("%s  %s  %s", entry.Name, entry.Source, entry.Description))
	}

	return strings.Join(lines, "\n")
}
