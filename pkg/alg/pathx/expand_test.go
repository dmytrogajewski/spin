package pathx

// Journey: specs/journeys/JOURNEY-extract-pathutil.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandHome(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err, "UserHomeDir must succeed for tests")

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "bare tilde expands to home",
			path: "~",
			want: home,
		},
		{
			name: "tilde slash expands to home subpath",
			path: "~/foo/bar",
			want: filepath.Join(home, "foo", "bar"),
		},
		{
			name: "absolute path unchanged",
			path: "/absolute/path",
			want: "/absolute/path",
		},
		{
			name: "relative path unchanged",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "empty string unchanged",
			path: "",
			want: "",
		},
		{
			name: "tilde user form unchanged",
			path: "~user/foo",
			want: "~user/foo",
		},
		{
			name: "tilde with single subdir",
			path: "~/data",
			want: filepath.Join(home, "data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := ExpandHome(tt.path)
			if tt.wantErr {
				assert.Error(t, gotErr)

				return
			}

			require.NoError(t, gotErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpandHome_NoError(t *testing.T) {
	t.Parallel()

	// Verify that non-tilde paths never produce errors.
	paths := []string{"", "/usr/local", "relative", "~user/foo"}

	for _, p := range paths {
		_, err := ExpandHome(p)
		assert.NoError(t, err, "ExpandHome(%q) should not error", p)
	}
}
