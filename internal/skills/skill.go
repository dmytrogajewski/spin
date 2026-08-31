// Package skills parses and validates Agent Skills SKILL.md directories.
package skills

import "errors"

// FileName is the required manifest filename inside a skill directory.
const FileName = "SKILL.md"

const (
	// MaxNameLen is the Agent Skills maximum length for name.
	MaxNameLen = 64
	// MaxDescriptionLen is the Agent Skills maximum length for description.
	MaxDescriptionLen = 1024
	// MaxCompatibilityLen is the Agent Skills maximum length for compatibility.
	MaxCompatibilityLen = 500

	frontmatterFence = "---"

	fieldName          = "name"
	fieldDescription   = "description"
	fieldLicense       = "license"
	fieldCompatibility = "compatibility"
	fieldMetadata      = "metadata"
	fieldAllowedTools  = "allowed-tools"
)

var (
	// ErrInvalidName is returned when name violates Agent Skills rules.
	ErrInvalidName = errors.New("invalid skill name")
	// ErrInvalidDescription is returned when description is empty or too long.
	ErrInvalidDescription = errors.New("invalid skill description")
	// ErrNameMismatch is returned when name does not match the parent directory.
	ErrNameMismatch = errors.New("skill name does not match directory")
	// ErrMissingFile is returned when SKILL.md is absent.
	ErrMissingFile = errors.New("SKILL.md not found")
	// ErrInvalidFrontmatter is returned when the YAML fence or payload is invalid.
	ErrInvalidFrontmatter = errors.New("invalid SKILL.md frontmatter")
	// ErrInvalidCompatibility is returned when compatibility exceeds the limit.
	ErrInvalidCompatibility = errors.New("invalid skill compatibility")
	// ErrInvalidMetadata is returned when metadata is not a string map.
	ErrInvalidMetadata = errors.New("invalid skill metadata")
	// ErrInvalidField is returned when a known field has the wrong YAML type.
	ErrInvalidField = errors.New("invalid skill field")
	// ErrUnknownSkill is returned when Activate cannot find the requested name.
	ErrUnknownSkill = errors.New("unknown skill")
	// ErrPathEscape is returned when a relative skill path leaves the skill root.
	ErrPathEscape = errors.New("path escapes skill root")
	// ErrEmptyName is returned when Activate is called with an empty name.
	ErrEmptyName = errors.New("skill name is empty")
	// ErrEmptyPath is returned when Resolve is called with an empty relative path.
	ErrEmptyPath = errors.New("skill path is empty")
)

// Skill is a parsed Agent Skill (frontmatter plus Markdown body).
type Skill struct {
	// Name is the skill identifier; it must match the parent directory.
	Name string
	// Description is what the skill does and when to use it.
	Description string
	// License is an optional license name or bundled-file reference.
	License string
	// Compatibility is an optional environment-requirement string.
	Compatibility string
	// Metadata is optional client-specific string key-value pairs.
	Metadata map[string]string
	// AllowedTools is an optional space-separated tool allowlist.
	AllowedTools string
	// Body is the Markdown after the closing frontmatter fence.
	Body string
	// Dir is the skill directory passed to Parse.
	Dir string
}
