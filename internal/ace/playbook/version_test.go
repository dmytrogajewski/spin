package playbook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

// Journey: specs/journeys/JOURNEY-1.6.md.

const testBulletID = "test-bullet-001"

// TestPlaybook_LoadWithoutVersion tests that loading a playbook JSON
// without a version field succeeds (backward compatibility).
// Kills mutant: rejecting versionless JSON would break old playbooks.
func TestPlaybook_LoadWithoutVersion(t *testing.T) {
	t.Parallel()

	oldJSON := `{
		"bullets": [
			{
				"id": "` + testBulletID + `",
				"content": "test bullet",
				"helpful_count": 1,
				"harmful_count": 0,
				"created_at": "2026-01-01T00:00:00Z",
				"updated_at": "2026-01-01T00:00:00Z"
			}
		]
	}`

	path := filepath.Join(t.TempDir(), "playbook.json")

	err := os.WriteFile(path, []byte(oldJSON), 0o600)
	require.NoError(t, err)

	pb, err := Load(path, nil, nil)
	require.NoError(t, err)

	b, found := pb.Get(testBulletID)
	assert.True(t, found)
	assert.Equal(t, "test bullet", b.Content)
}

// TestPlaybook_SaveIncludesVersion tests that saving a playbook
// includes the version field in the JSON output.
// Kills mutant: omitting version from JSON would break versioning.
func TestPlaybook_SaveIncludesVersion(t *testing.T) {
	t.Parallel()

	pb := New(nil, nil)

	now := time.Now()

	_ = pb.Add(t.Context(), &bullet.Bullet{
		ID:        testBulletID,
		Content:   "test bullet",
		CreatedAt: now,
		UpdatedAt: now,
	})

	path := filepath.Join(t.TempDir(), "playbook.json")

	err := pb.Save(t.Context(), path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var raw map[string]any

	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	version, exists := raw["version"]
	assert.True(t, exists, "JSON should contain version field")
	assert.InEpsilon(t, float64(CurrentPlaybookVersion), version, 0.001)
}

// TestPlaybook_ForwardCompatibility tests that unknown fields in JSON
// are tolerated (forward compatibility).
// Kills mutant: using DisallowUnknownFields would break forward compat.
func TestPlaybook_ForwardCompatibility(t *testing.T) {
	t.Parallel()

	futureJSON := `{
		"version": 1,
		"bullets": [],
		"future_field": "unknown",
		"another_new": 42
	}`

	path := filepath.Join(t.TempDir(), "playbook.json")

	err := os.WriteFile(path, []byte(futureJSON), 0o600)
	require.NoError(t, err)

	pb, err := Load(path, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, pb)
}
