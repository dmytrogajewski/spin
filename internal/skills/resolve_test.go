package skills_test

// Journey: specs/journeys/JOURNEY-003-activate-skill-body.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
)

func TestResolve_AcceptsInRootRelative(t *testing.T) {
	t.Parallel()

	root := writeActivateSkillWithRefs(t)
	got, err := skills.Resolve(root, "references/extra.md")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "references", "extra.md"), got)
}

func TestResolve_RejectsDotDot(t *testing.T) {
	t.Parallel()

	root := writeActivateSkillWithRefs(t)

	tests := []string{
		"../secrets.md",
		"references/../../../outside.md",
		"references/../SKILL.md",
	}

	for _, rel := range tests {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()

			_, err := skills.Resolve(root, rel)
			require.ErrorIs(t, err, skills.ErrPathEscape)
		})
	}
}

func TestResolve_RejectsAbsolute(t *testing.T) {
	t.Parallel()

	_, err := skills.Resolve(t.TempDir(), "/etc/passwd")
	require.ErrorIs(t, err, skills.ErrPathEscape)
}

func TestResolve_RejectsEmpty(t *testing.T) {
	t.Parallel()

	_, err := skills.Resolve(t.TempDir(), "")
	require.ErrorIs(t, err, skills.ErrEmptyPath)
}

func TestReadResource_OneHopOnly(t *testing.T) {
	t.Parallel()

	root := writeActivateSkillWithRefs(t)
	data, err := skills.ReadResource(root, "references/extra.md")
	require.NoError(t, err)
	require.Contains(t, string(data), activateExtraSentinel)
	require.NotContains(t, string(data), activateNestedSentinel)
}

func TestReadResource_RejectsEscape(t *testing.T) {
	t.Parallel()

	root := writeActivateSkillWithRefs(t)
	_, err := skills.ReadResource(root, "../secrets.md")
	require.ErrorIs(t, err, skills.ErrPathEscape)
}

func TestReadResource_MissingFile(t *testing.T) {
	t.Parallel()

	root := writeActivateSkillWithRefs(t)
	_, err := skills.ReadResource(root, "references/missing.md")
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}
