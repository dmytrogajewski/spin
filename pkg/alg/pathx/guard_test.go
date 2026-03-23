package pathx

// Journey: specs/journeys/JOURNEY-R6.md.

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsUnsafeWorkDir(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name    string
		workDir string
		want    bool
	}{
		{name: "root_is_unsafe", workDir: "/", want: true},
		{name: "tmp_is_unsafe", workDir: "/tmp", want: true},
		{name: "var_is_unsafe", workDir: "/var", want: true},
		{name: "home_is_unsafe", workDir: home, want: true},
		{name: "project_dir_is_safe", workDir: "/home/user/project", want: false},
		{name: "deep_nested_is_safe", workDir: "/opt/app/data", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsUnsafeWorkDir(tt.workDir)
			require.Equal(t, tt.want, got)
		})
	}
}
