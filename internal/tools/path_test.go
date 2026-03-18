// Journey: specs/journeys/JOURNEY-RT4.md
package tools

import "testing"

func TestResolvePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		workDir string
		want    string
	}{
		{
			name:    "absolute_path_unchanged",
			path:    "/home/user/file.go",
			workDir: "/workspace",
			want:    "/home/user/file.go",
		},
		{
			name:    "relative_path_joined",
			path:    "src/main.go",
			workDir: "/workspace",
			want:    "/workspace/src/main.go",
		},
		{
			name:    "empty_workdir_returns_original",
			path:    "src/main.go",
			workDir: "",
			want:    "src/main.go",
		},
		{
			name:    "empty_path_returns_workdir",
			path:    "",
			workDir: "/workspace",
			want:    "/workspace",
		},
		{
			name:    "both_empty",
			path:    "",
			workDir: "",
			want:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolvePath(tc.path, tc.workDir)
			if got != tc.want {
				t.Errorf("resolvePath(%q, %q) = %q; want %q", tc.path, tc.workDir, got, tc.want)
			}
		})
	}
}
