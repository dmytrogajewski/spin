package plugins

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dmytrogajewski/spin/internal/skills"
)

// DiscoverOptions selects plugin search roots.
type DiscoverOptions struct {
	// WorkDir is scanned at <workDir>/.spin/plugins/*.
	WorkDir string
	// HomeDir is scanned at <home>/.spin/plugins/*.
	HomeDir string
	// ExtraPaths are config plugins.paths entries: a plugin root or a directory of roots.
	ExtraPaths []string
}

// Skip records a plugin root that failed fatal plugin.json validation.
type Skip struct {
	// Root is the rejected plugin directory.
	Root string
	// Err is the fatal Load error.
	Err error
}

// Result is a Discover scan: loaded plugins plus independent skips.
type Result struct {
	// Plugins loaded successfully (skills present even if MCP failed).
	Plugins []Plugin
	// Skipped roots whose plugin.json was fatal.
	Skipped []Skip
}

// Discover loads plugin roots independently. A fatal plugin.json skips that root only.
func Discover(opts DiscoverOptions) Result {
	seen := make(map[string]struct{})
	result := Result{
		Plugins: make([]Plugin, 0),
		Skipped: make([]Skip, 0),
	}

	for _, root := range collectPluginRoots(opts) {
		plugin, err := Load(root)
		if err != nil {
			result.Skipped = append(result.Skipped, Skip{Root: root, Err: err})

			continue
		}

		if _, exists := seen[plugin.Manifest.Name]; exists {
			continue
		}

		seen[plugin.Manifest.Name] = struct{}{}
		result.Plugins = append(result.Plugins, plugin)
	}

	return result
}

// SkillContributions converts loaded plugin skills into catalog rows.
func SkillContributions(loaded []Plugin) []skills.PluginSkill {
	contribs := make([]skills.PluginSkill, 0)

	for _, plugin := range loaded {
		for _, skill := range plugin.Skills {
			contribs = append(contribs, skills.PluginSkill{
				PluginName:  plugin.Manifest.Name,
				Name:        skill.Name,
				Description: skill.Description,
				Location:    skill.Dir,
			})
		}
	}

	return contribs
}

// CatalogOptions builds Discover options that include plugin skill contributions.
func CatalogOptions(workDir string, extraPaths []string) skills.Options {
	opts := skills.OptionsFor(workDir)
	result := Discover(DiscoverOptions{
		WorkDir:    workDir,
		HomeDir:    opts.HomeDir,
		ExtraPaths: extraPaths,
	})
	opts.PluginSkills = SkillContributions(result.Plugins)

	return opts
}

// DiscoverCatalog is Discover + skill merge for /skills and Composer.
func DiscoverCatalog(workDir string, extraPaths []string) skills.Catalog {
	return skills.Discover(CatalogOptions(workDir, extraPaths))
}

func collectPluginRoots(opts DiscoverOptions) []string {
	workRoots := pluginChildren(filepath.Join(opts.WorkDir, RelPluginsDir))
	homeRoots := pluginChildren(filepath.Join(opts.HomeDir, RelPluginsDir))
	roots := make([]string, 0, len(workRoots)+len(homeRoots)+len(opts.ExtraPaths))
	roots = append(roots, workRoots...)
	roots = append(roots, homeRoots...)

	for _, extra := range opts.ExtraPaths {
		roots = append(roots, expandExtraPath(extra)...)
	}

	return roots
}

func expandExtraPath(path string) []string {
	if path == "" {
		return nil
	}

	if hasManifest(path) {
		return []string{path}
	}

	return pluginChildren(path)
}

func pluginChildren(dir string) []string {
	if dir == "" {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	roots := make([]string, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		roots = append(roots, filepath.Join(dir, entry.Name()))
	}

	return roots
}

func hasManifest(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ManifestFile))

	return err == nil && info.Mode().IsRegular()
}
