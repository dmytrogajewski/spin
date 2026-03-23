package execx

// Journey: specs/journeys/JOURNEY-S6.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// defaultPrefixes mirrors the prefixes used by the agent environment filter.
var defaultPrefixes = []string{"AWS_", "GCP_", "OPENAI_", "ANTHROPIC_"}

// defaultSubstrings mirrors the substrings used by the agent environment filter.
var defaultSubstrings = []string{"TOKEN", "KEY", "SECRET", "PASSWORD", "AUTH"}

func TestFilterEnvironment(t *testing.T) {
	t.Parallel()

	env := []string{
		"HOME=/home/user",
		"PATH=/usr/bin",
		"AWS_SECRET_KEY=supersecret",
		"OPENAI_API_KEY=sk-xxx",
		"MY_TOKEN=abc123",
		"GIT_AUTH_TOKEN=ghp_xxx",
		"SAFE_VAR=hello",
		"SHELL=/bin/bash",
	}

	t.Run("strips_sensitive_preserves_safe", func(t *testing.T) {
		t.Parallel()

		got := FilterEnvironment(env, defaultPrefixes, defaultSubstrings)

		require.Equal(t, "/home/user", got["HOME"])
		require.Equal(t, "/usr/bin", got["PATH"])
		require.Equal(t, "hello", got["SAFE_VAR"])
		require.Equal(t, "/bin/bash", got["SHELL"])

		require.NotContains(t, got, "AWS_SECRET_KEY")
		require.NotContains(t, got, "OPENAI_API_KEY")
		require.NotContains(t, got, "MY_TOKEN")
		require.NotContains(t, got, "GIT_AUTH_TOKEN")
	})

	t.Run("empty_env", func(t *testing.T) {
		t.Parallel()

		got := FilterEnvironment(nil, defaultPrefixes, defaultSubstrings)
		require.Empty(t, got)
	})

	t.Run("no_filters_keeps_all", func(t *testing.T) {
		t.Parallel()

		got := FilterEnvironment(env, nil, nil)
		require.Len(t, got, len(env))
	})

	t.Run("malformed_entries_skipped", func(t *testing.T) {
		t.Parallel()

		got := FilterEnvironment([]string{"NOEQUALS", "GOOD=val"}, nil, nil)
		require.Len(t, got, 1)
		require.Equal(t, "val", got["GOOD"])
	})

	t.Run("case_insensitive_prefix", func(t *testing.T) {
		t.Parallel()

		got := FilterEnvironment([]string{"aws_key=x"}, []string{"AWS_"}, nil)
		require.Empty(t, got)
	})

	t.Run("case_insensitive_substring", func(t *testing.T) {
		t.Parallel()

		got := FilterEnvironment([]string{"my_password=x"}, nil, []string{"PASSWORD"})
		require.Empty(t, got)
	})
}
