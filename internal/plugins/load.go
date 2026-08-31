package plugins

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dmytrogajewski/spin/internal/skills"
)

// Load validates plugin.json at dir and discovers immediate skills/ children.
func Load(dir string) (Plugin, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return Plugin{}, fmt.Errorf("plugin root: %w", err)
	}

	manifest, err := readManifest(root)
	if err != nil {
		return Plugin{}, err
	}

	plugin := Plugin{
		Root:     root,
		Manifest: manifest,
		Warnings: manifestWarnings(manifest),
	}

	found, skillWarnings, skillErr := discoverSkills(root)
	if skillErr != nil {
		return Plugin{}, skillErr
	}

	plugin.Skills = found
	plugin.Warnings = append(plugin.Warnings, skillWarnings...)
	applyLoadedMCP(&plugin, root)

	return plugin, nil
}

func applyLoadedMCP(plugin *Plugin, root string) {
	file, warnings, err := loadMCP(root)
	if err != nil {
		plugin.Warnings = append(plugin.Warnings, warnInvalidMCP+err.Error())

		return
	}

	if file.Schema == "" {
		return
	}

	plugin.MCP = file
	plugin.MCPValid = true
	plugin.Warnings = append(plugin.Warnings, warnings...)
}

func readManifest(root string) (Manifest, error) {
	path := filepath.Join(root, ManifestFile)

	if _, containErr := Contain(root, relPathPrefix+ManifestFile); containErr != nil {
		return Manifest{}, containErr
	}

	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return Manifest{}, fmt.Errorf("%w: %s", ErrMissingManifest, path)
		}

		return Manifest{}, fmt.Errorf("read %s: %w", path, readErr)
	}

	return ParseManifest(raw)
}

func discoverSkills(root string) ([]skills.Skill, []string, error) {
	skillsPath := filepath.Join(root, SkillsDir)

	_, err := os.Lstat(skillsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, nil
		}

		return nil, nil, fmt.Errorf("stat %s: %w", skillsPath, err)
	}

	if warning := containmentWarning(root, relPathPrefix+SkillsDir); warning != "" {
		return nil, []string{warning}, nil
	}

	info, statErr := os.Stat(skillsPath)
	if statErr != nil {
		return nil, nil, fmt.Errorf("stat %s: %w", skillsPath, statErr)
	}

	if !info.IsDir() {
		return nil, []string{warnSkillsNotDir}, nil
	}

	entries, readErr := os.ReadDir(skillsPath)
	if readErr != nil {
		return nil, nil, fmt.Errorf("read %s: %w", skillsPath, readErr)
	}

	found, warnings := loadImmediateSkills(root, skillsPath, entries)

	return found, warnings, nil
}

func loadImmediateSkills(root, skillsPath string, entries []fs.DirEntry) (found []skills.Skill, warnings []string) {
	found = make([]skills.Skill, 0)
	warnings = make([]string, 0)

	for _, entry := range entries {
		skill, warning := loadImmediateSkill(root, skillsPath, entry)
		if warning != "" {
			warnings = append(warnings, warning)

			continue
		}

		if skill.Name == "" {
			continue
		}

		found = append(found, skill)
	}

	slices.SortFunc(found, func(left, right skills.Skill) int {
		return strings.Compare(left.Name, right.Name)
	})

	return found, warnings
}

func loadImmediateSkill(root, skillsPath string, entry fs.DirEntry) (skill skills.Skill, warning string) {
	childPath := filepath.Join(skillsPath, entry.Name())

	info, err := os.Stat(childPath)
	if err != nil {
		return skills.Skill{}, skippedSkill(entry.Name(), err)
	}

	if !info.IsDir() {
		return skills.Skill{}, ""
	}

	rel := relPathPrefix + SkillsDir + "/" + entry.Name() + "/" + skills.FileName
	if _, containErr := Contain(root, rel); containErr != nil {
		return skills.Skill{}, skippedSkill(entry.Name(), containErr)
	}

	parsed, parseErr := skills.Parse(childPath)
	if parseErr != nil {
		return skills.Skill{}, skippedSkill(entry.Name(), parseErr)
	}

	return parsed, ""
}

func containmentWarning(root, rel string) string {
	if _, err := Contain(root, rel); err != nil {
		return err.Error()
	}

	return ""
}

func skippedSkill(name string, err error) string {
	return warnSkippedSkill + name + ": " + err.Error()
}
