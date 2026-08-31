package skills_test

// Journey: specs/journeys/JOURNEY-001-parse-and-validate-agent-skills.md.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	fixtureRoot        = "testdata"
	shippedSkillsRel   = "../../.agents/skills"
	catalogSkillCount  = 200
	catalogP99Percent  = 99
	catalogP99Limit    = 50 * time.Millisecond
	dirPerm            = 0o750
	filePerm           = 0o600
	nameAtLimitLen     = 64
	validMinimalBody   = "\n# Valid minimal\n\nBody stays after the fence.\n"
	validFullBody      = "\n# Full skill\n\nInstructions here.\n"
	validBodyFenceBody = "\n# Title\n\n---\n\nMore after a body fence.\n"
)

func TestParse_Fixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dir     string
		wantErr error
		check   func(*testing.T, skills.Skill)
	}{
		{
			name:  "valid_minimal",
			dir:   "valid-minimal",
			check: checkValidMinimal,
		},
		{
			name:  "valid_full",
			dir:   "valid-full",
			check: checkValidFull,
		},
		{
			name:  "valid_body_fence",
			dir:   "valid-body-fence",
			check: checkValidBodyFence,
		},
		{
			name:    "uppercase",
			dir:     "uppercase",
			wantErr: skills.ErrInvalidName,
		},
		{
			name:    "leading_hyphen",
			dir:     "leading-hyphen",
			wantErr: skills.ErrInvalidName,
		},
		{
			name:    "trailing_hyphen",
			dir:     "trailing-hyphen",
			wantErr: skills.ErrInvalidName,
		},
		{
			name:    "consecutive_hyphens",
			dir:     "consecutive-hyphens",
			wantErr: skills.ErrInvalidName,
		},
		{
			name:    "name_too_long",
			dir:     "name-too-long",
			wantErr: skills.ErrInvalidName,
		},
		{
			name:    "name_mismatch",
			dir:     "name-mismatch",
			wantErr: skills.ErrNameMismatch,
		},
		{
			name:    "empty_description",
			dir:     "empty-description",
			wantErr: skills.ErrInvalidDescription,
		},
		{
			name:    "description_too_long",
			dir:     "description-too-long",
			wantErr: skills.ErrInvalidDescription,
		},
		{
			name:    "missing_file",
			dir:     "missing-file",
			wantErr: skills.ErrMissingFile,
		},
		{
			name:    "missing_frontmatter",
			dir:     "missing-frontmatter",
			wantErr: skills.ErrInvalidFrontmatter,
		},
		{
			name:    "invalid_yaml",
			dir:     "invalid-yaml",
			wantErr: skills.ErrInvalidFrontmatter,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			skill, err := skills.Parse(filepath.Join(fixtureRoot, tc.dir))
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Zero(t, skill)

				return
			}

			require.NoError(t, err)
			tc.check(t, skill)
		})
	}
}

func TestValidate_NameAndDescription(t *testing.T) {
	t.Parallel()

	valid := skills.Skill{
		Name:        "hello-world",
		Description: "Use when parsing a valid constructed skill.",
		Dir:         "hello-world",
	}
	require.NoError(t, skills.Validate(valid))

	tooLongName := strings.Repeat("n", nameAtLimitLen+1)
	err := skills.Validate(skills.Skill{
		Name:        tooLongName,
		Description: valid.Description,
		Dir:         tooLongName,
	})
	require.ErrorIs(t, err, skills.ErrInvalidName)

	atLimit := strings.Repeat("n", nameAtLimitLen)
	require.NoError(t, skills.Validate(skills.Skill{
		Name:        atLimit,
		Description: valid.Description,
		Dir:         atLimit,
	}))

	err = skills.Validate(skills.Skill{
		Name:        valid.Name,
		Description: "",
		Dir:         valid.Dir,
	})
	require.ErrorIs(t, err, skills.ErrInvalidDescription)
}

func TestParse_ShippedSkills(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(shippedSkillsRel)
	require.NoError(t, err)

	entries, err := os.ReadDir(root)
	require.NoError(t, err)

	found := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(root, entry.Name())
		if _, statErr := os.Stat(filepath.Join(skillDir, skills.FileName)); statErr != nil {
			continue
		}

		found++

		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()

			skill, parseErr := skills.Parse(skillDir)
			require.NoError(t, parseErr)
			require.Equal(t, entry.Name(), skill.Name)
			require.NotEmpty(t, skill.Description)
			require.False(t, strings.HasPrefix(skill.Body, "---"))
		})
	}

	require.NotZero(t, found)
}

func TestParse_CatalogP99(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dirs := make([]string, catalogSkillCount)

	for i := range catalogSkillCount {
		name := catalogSkillName(i)
		dir := filepath.Join(root, name)
		require.NoError(t, os.Mkdir(dir, dirPerm))

		content := "---\nname: " + name + "\ndescription: Synthetic skill for catalog parse budget.\n---\n\n# " + name + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, skills.FileName), []byte(content), filePerm))

		dirs[i] = dir
	}

	durations := make([]time.Duration, catalogSkillCount)

	for i, dir := range dirs {
		start := time.Now()
		_, err := skills.Parse(dir)
		durations[i] = time.Since(start)

		require.NoError(t, err)
	}

	require.Less(t, percentile(durations, catalogP99Percent), catalogP99Limit)
}

func checkValidMinimal(t *testing.T, skill skills.Skill) {
	t.Helper()
	require.Equal(t, "valid-minimal", skill.Name)
	require.Equal(t, "A minimal valid skill used as a parse fixture.", skill.Description)
	require.Empty(t, skill.License)
	require.Empty(t, skill.Compatibility)
	require.Nil(t, skill.Metadata)
	require.Empty(t, skill.AllowedTools)
	require.Equal(t, validMinimalBody, skill.Body)
	require.Equal(t, filepath.Join(fixtureRoot, "valid-minimal"), skill.Dir)
}

func checkValidFull(t *testing.T, skill skills.Skill) {
	t.Helper()
	require.Equal(t, "valid-full", skill.Name)
	require.Equal(t, "A full valid skill with every optional frontmatter field.", skill.Description)
	require.Equal(t, "MIT", skill.License)
	require.Equal(t, "go >= 1.25", skill.Compatibility)
	require.Equal(t, map[string]string{"author": "spin", "origin": "testdata"}, skill.Metadata)
	require.Equal(t, "read_file grep", skill.AllowedTools)
	require.Equal(t, validFullBody, skill.Body)
	require.NotContains(t, skill.Body, "license:")
}

func checkValidBodyFence(t *testing.T, skill skills.Skill) {
	t.Helper()
	require.Equal(t, "valid-body-fence", skill.Name)
	require.Equal(t, validBodyFenceBody, skill.Body)
	require.Contains(t, skill.Body, "---")
	require.Contains(t, skill.Body, "More after a body fence.")
}

func catalogSkillName(index int) string {
	return fmt.Sprintf("skill-%03d", index)
}

func percentile(durations []time.Duration, pct int) time.Duration {
	sorted := slices.Clone(durations)
	slices.Sort(sorted)

	idx := max(int(math.Ceil(float64(len(sorted))*float64(pct)/100))-1, 0)
	idx = min(idx, len(sorted)-1)

	return sorted[idx]
}
