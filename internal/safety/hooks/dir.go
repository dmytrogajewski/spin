package hooks

import "github.com/dmytrogajewski/spin/pkg/alg/pathx"

// defaultGlobalHooksRel is the user-level hooks directory under $HOME.
const defaultGlobalHooksRel = "~/.spin/hooks"

// DefaultGlobalDir returns the expanded user-level hooks directory.
// A leading ~ is resolved with [os.UserHomeDir] via pathx.ExpandHome.
func DefaultGlobalDir() string {
	expanded, err := pathx.ExpandHome(defaultGlobalHooksRel)
	if err != nil {
		return ""
	}

	return expanded
}
