package prompt

import (
	"strings"

	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	// SectionSkillCatalog is the Composer section that lists discovered skills.
	SectionSkillCatalog = "skill-catalog"

	prioritySkillCatalog = 16

	catalogHeading = "# Skills"
)

// SkillCatalogSection renders name + description only (no bodies, paths, or source).
func SkillCatalogSection(catalog skills.Catalog) Section {
	lines := make([]string, 0, len(catalog)+1)
	lines = append(lines, catalogHeading)

	for _, entry := range catalog {
		lines = append(lines, "- "+entry.Name+": "+entry.Description)
	}

	return Section{
		Name:      SectionSkillCatalog,
		Priority:  prioritySkillCatalog,
		Cacheable: true,
		Template:  strings.Join(lines, "\n"),
	}
}

// ApplyCatalog adds a metadata-only skill catalog section when the catalog is non-empty.
func ApplyCatalog(composer *Composer, catalog skills.Catalog) {
	if composer == nil || len(catalog) == 0 {
		return
	}

	composer.AddSection(SkillCatalogSection(catalog))
}
