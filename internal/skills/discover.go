package skills

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type skillRoot struct {
	path   string
	source Source
}

// Discover walks project, user, and bundled skill roots in priority order.
// Missing roots are ignored. The first name wins. Empty catalog is valid.
func Discover(opts Options) Catalog {
	seen := make(map[string]struct{})
	catalog := make(Catalog, 0)

	for _, root := range discoveryRoots(opts) {
		if root.source == SourceBundled {
			continue
		}

		catalog = appendRoot(catalog, seen, root)
	}

	catalog = appendPluginSkills(catalog, seen, opts.PluginSkills)
	catalog = appendBundled(catalog, seen, opts.BundledDir)

	slices.SortFunc(catalog, func(left, right Entry) int {
		return strings.Compare(left.Name, right.Name)
	})

	return catalog
}

func appendPluginSkills(catalog Catalog, seen map[string]struct{}, contribs []PluginSkill) Catalog {
	for _, contrib := range contribs {
		if contrib.Name == "" || contrib.PluginName == "" {
			continue
		}

		if _, exists := seen[contrib.Name]; exists {
			continue
		}

		seen[contrib.Name] = struct{}{}
		catalog = append(catalog, Entry{
			Name:        contrib.Name,
			Description: contrib.Description,
			Location:    contrib.Location,
			Source:      PluginSource(contrib.PluginName),
		})
	}

	return catalog
}

func appendBundled(catalog Catalog, seen map[string]struct{}, bundledDir string) Catalog {
	if bundledDir == "" {
		return catalog
	}

	return appendRoot(catalog, seen, skillRoot{path: bundledDir, source: SourceBundled})
}

// OptionsFor builds Discover options from a workdir, the process home, and SPIN_HOME.
func OptionsFor(workDir string) Options {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	return Options{
		WorkDir:    workDir,
		HomeDir:    home,
		BundledDir: bundledDirFromEnv(),
	}
}

func bundledDirFromEnv() string {
	root := os.Getenv(EnvSpinHome)
	if root == "" {
		return ""
	}

	return filepath.Join(root, bundledSubdir)
}

func discoveryRoots(opts Options) []skillRoot {
	roots := make([]skillRoot, 0, discoveryRootCap)

	if opts.WorkDir != "" {
		roots = append(roots,
			skillRoot{path: filepath.Join(opts.WorkDir, relAgentsSkills), source: SourceProject},
			skillRoot{path: filepath.Join(opts.WorkDir, relClaudeSkills), source: SourceProject},
		)
	}

	if opts.HomeDir != "" {
		roots = append(roots,
			skillRoot{path: filepath.Join(opts.HomeDir, relSpinSkills), source: SourceUser},
			skillRoot{path: filepath.Join(opts.HomeDir, relAgentsSkills), source: SourceUser},
		)
	}

	if opts.BundledDir != "" {
		roots = append(roots, skillRoot{path: opts.BundledDir, source: SourceBundled})
	}

	return roots
}

func appendRoot(catalog Catalog, seen map[string]struct{}, root skillRoot) Catalog {
	entries, err := os.ReadDir(root.path)
	if err != nil {
		return catalog
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if _, exists := seen[name]; exists {
			continue
		}

		skill, parseErr := Parse(filepath.Join(root.path, name))
		if parseErr != nil {
			continue
		}

		seen[name] = struct{}{}

		catalog = append(catalog, Entry{
			Name:        skill.Name,
			Description: skill.Description,
			Location:    skill.Dir,
			Source:      root.source,
		})
	}

	return catalog
}
