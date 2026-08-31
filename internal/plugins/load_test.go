package plugins_test

// Journey: specs/journeys/JOURNEY-004-parse-agent-plugins.md.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/plugins"
)

const (
	fixtureRoot       = "testdata"
	fixtureValid      = "valid-plugin"
	fixtureEscape     = "escape-path"
	fixtureNested     = "nested-skill"
	fixtureUnknown    = "unknown-field"
	skillSummarize    = "summarize"
	skillDeploy       = "deploy"
	skillNested       = "nested"
	unknownFieldName  = "commands"
	inRootSkillRel    = "./skills/summarize/SKILL.md"
	escapeRel         = "../secret"
	bareDataRel       = "data"
	dotDotPrefixedRel = "./../secret"
	dirPerm           = 0o750
	filePerm          = 0o600
)

func TestLoad_ValidPlugin(t *testing.T) {
	t.Parallel()

	plugin, err := plugins.Load(filepath.Join(fixtureRoot, fixtureValid))
	require.NoError(t, err)
	require.Equal(t, "valid-plugin", plugin.Manifest.Name)
	require.Equal(t, plugins.SchemaV1, plugin.Manifest.Schema)
	require.Equal(t, "1.2.0", plugin.Manifest.Version)
	require.Equal(t, "Brief plugin description", plugin.Manifest.Description)
	require.NotNil(t, plugin.Manifest.Author)
	require.Equal(t, "Author Name", plugin.Manifest.Author.Name)
	require.Equal(t, []string{"keyword1", "keyword2"}, plugin.Manifest.Keywords)
	require.Contains(t, plugin.Manifest.Extensions, "com.example.client")
	require.Empty(t, plugin.Manifest.UnknownFields)
	require.Len(t, plugin.Skills, 1)
	require.Equal(t, skillSummarize, plugin.Skills[0].Name)
}

func TestLoad_UnknownFieldIgnored(t *testing.T) {
	t.Parallel()

	plugin, err := plugins.Load(filepath.Join(fixtureRoot, fixtureUnknown))
	require.NoError(t, err)
	require.Equal(t, fixtureUnknown, plugin.Manifest.Name)
	require.Equal(t, []string{unknownFieldName}, plugin.Manifest.UnknownFields)
	require.Empty(t, plugin.Skills)
	require.True(t, hasUnknownFieldWarning(plugin.Warnings, unknownFieldName))
}

func TestLoad_NestedSkillIgnored(t *testing.T) {
	t.Parallel()

	plugin, err := plugins.Load(filepath.Join(fixtureRoot, fixtureNested))
	require.NoError(t, err)
	require.Len(t, plugin.Skills, 1)
	require.Equal(t, skillDeploy, plugin.Skills[0].Name)

	for _, skill := range plugin.Skills {
		require.NotEqual(t, skillNested, skill.Name)
	}
}

func TestLoad_MissingManifest(t *testing.T) {
	t.Parallel()

	_, err := plugins.Load(t.TempDir())
	require.ErrorIs(t, err, plugins.ErrMissingManifest)
}

func TestLoad_MissingSkillsValid(t *testing.T) {
	t.Parallel()

	plugin, err := plugins.Load(filepath.Join(fixtureRoot, fixtureUnknown))
	require.NoError(t, err)
	require.Empty(t, plugin.Skills)
}

func TestContain_EscapePathFixture(t *testing.T) {
	t.Parallel()

	root := filepath.Join(fixtureRoot, fixtureEscape)

	got, err := plugins.Contain(root, inRootSkillRel)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(filepath.ToSlash(got), "skills/summarize/SKILL.md"))

	_, err = plugins.Contain(root, escapeRel)
	require.ErrorIs(t, err, plugins.ErrNotPluginRelative)

	_, err = plugins.Contain(root, bareDataRel)
	require.ErrorIs(t, err, plugins.ErrNotPluginRelative)
}

func TestContain_DotDotAfterPrefix(t *testing.T) {
	t.Parallel()

	_, err := plugins.Contain(filepath.Join(fixtureRoot, fixtureEscape), dotDotPrefixedRel)
	require.ErrorIs(t, err, plugins.ErrPathEscape)
}

func TestContain_EmptyPath(t *testing.T) {
	t.Parallel()

	_, err := plugins.Contain(filepath.Join(fixtureRoot, fixtureEscape), "")
	require.ErrorIs(t, err, plugins.ErrEmptyPath)
}

func TestParseManifest_SchemaAndNameRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		wantErr error
	}{
		{
			name:    "missing_schema",
			payload: `{"name":"ok-plugin"}`,
			wantErr: plugins.ErrInvalidSchema,
		},
		{
			name:    "missing_name",
			payload: `{"$schema":"` + plugins.SchemaV1 + `"}`,
			wantErr: plugins.ErrInvalidName,
		},
		{
			name:    "empty_name",
			payload: `{"$schema":"` + plugins.SchemaV1 + `","name":""}`,
			wantErr: plugins.ErrInvalidName,
		},
		{
			name:    "wrong_schema",
			payload: `{"$schema":"https://example.com/schema.json","name":"ok-plugin"}`,
			wantErr: plugins.ErrInvalidSchema,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := plugins.ParseManifest([]byte(tc.payload))
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestParseManifest_NameConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{name: "uppercase", value: "My-Plugin"},
		{name: "leading_hyphen", value: "-start"},
		{name: "trailing_hyphen", value: "end-"},
		{name: "leading_period", value: ".hidden"},
		{name: "trailing_period", value: "end."},
		{name: "consecutive_hyphens", value: "has--double"},
		{name: "consecutive_periods", value: "too.many..dots"},
		{name: "too_long", value: strings.Repeat("a", plugins.MaxNameLen+1)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload := `{"$schema":"` + plugins.SchemaV1 + `","name":"` + tc.value + `"}`
			_, err := plugins.ParseManifest([]byte(payload))
			require.ErrorIs(t, err, plugins.ErrInvalidName)
		})
	}
}

func TestParseManifest_OtherViolationsReject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		wantErr error
	}{
		{
			name:    "top_level_array",
			payload: `[]`,
			wantErr: plugins.ErrInvalidManifest,
		},
		{
			name:    "version_not_string",
			payload: `{"$schema":"` + plugins.SchemaV1 + `","name":"ok-plugin","version":1}`,
			wantErr: plugins.ErrInvalidField,
		},
		{
			name:    "author_unknown_key",
			payload: `{"$schema":"` + plugins.SchemaV1 + `","name":"ok-plugin","author":{"twitter":"x"}}`,
			wantErr: plugins.ErrInvalidField,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := plugins.ParseManifest([]byte(tc.payload))
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestParseManifest_ExtensionsNonObjectIgnored(t *testing.T) {
	t.Parallel()

	payload := `{"$schema":"` + plugins.SchemaV1 + `","name":"ok-plugin","extensions":["nope"]}`
	manifest, err := plugins.ParseManifest([]byte(payload))
	require.NoError(t, err)
	require.True(t, manifest.ExtensionsIgnored)
	require.Nil(t, manifest.Extensions)
}

func TestParseManifest_ValidNames(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"my-plugin", "acme.tools", "lint3r", "a"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			payload := `{"$schema":"` + plugins.SchemaV1 + `","name":"` + name + `"}`
			manifest, err := plugins.ParseManifest([]byte(payload))
			require.NoError(t, err)
			require.Equal(t, name, manifest.Name)
		})
	}
}

func TestLoad_SymlinkSkillEscapeSkipped(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	outsideSkill := filepath.Join(outside, "leak")
	require.NoError(t, os.MkdirAll(outsideSkill, dirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(outsideSkill, "SKILL.md"), skillMarkdown("leak"), filePerm))

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile), minimalManifest("symlink-escape"), filePerm))
	require.NoError(t, os.Mkdir(filepath.Join(root, plugins.SkillsDir), dirPerm))
	require.NoError(t, os.Symlink(outsideSkill, filepath.Join(root, plugins.SkillsDir, "leak")))

	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.Empty(t, plugin.Skills)
	require.NotEmpty(t, plugin.Warnings)
}

func TestLoad_SkillsFileIgnored(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile), minimalManifest("skills-file"), filePerm))
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.SkillsDir), []byte("not a dir"), filePerm))

	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.Empty(t, plugin.Skills)
	require.Contains(t, plugin.Warnings, "skills is not a directory; ignored")
}

func TestParseManifest_RoundTripUnknownNotStored(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join(fixtureRoot, fixtureUnknown, plugins.ManifestFile))
	require.NoError(t, err)

	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Contains(t, payload, unknownFieldName)

	manifest, parseErr := plugins.ParseManifest(raw)
	require.NoError(t, parseErr)
	require.Equal(t, []string{unknownFieldName}, manifest.UnknownFields)
	require.Empty(t, manifest.Version)
}

func hasUnknownFieldWarning(warnings []string, field string) bool {
	needle := `unknown field "` + field + `"`
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}

	return false
}

func minimalManifest(name string) []byte {
	return []byte(`{"$schema":"` + plugins.SchemaV1 + `","name":"` + name + `"}`)
}

func skillMarkdown(name string) []byte {
	return []byte("---\nname: " + name + "\ndescription: Temp skill for plugin containment tests.\n---\n\n# " + name + "\n")
}
