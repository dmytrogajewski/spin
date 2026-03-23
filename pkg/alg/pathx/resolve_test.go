package pathx

// Journey: specs/journeys/JOURNEY-R6.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		workDir string
		path    string
		want    string
	}{
		{name: "absolute_passthrough", workDir: "/work", path: "/abs/path", want: "/abs/path"},
		{name: "relative_joined", workDir: "/work", path: "rel", want: "/work/rel"},
		{name: "empty_workdir_passthrough", workDir: "", path: "rel", want: "rel"},
		{name: "dot_path", workDir: "/work", path: ".", want: "/work"},
		{name: "empty_path", workDir: "/work", path: "", want: "/work"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ResolvePath(tt.workDir, tt.path)
			require.Equal(t, tt.want, got)
		})
	}
}
