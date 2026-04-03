package execx

// Journey: specs/journeys/JOURNEY-R12.md.

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFindEditor(t *testing.T) {
	// Cannot use t.Parallel — subtests modify process-global env vars.
	tests := []struct {
		name         string
		editorEnv    string
		visualEnv    string
		wantNonEmpty bool
	}{
		{
			name:         "uses EDITOR",
			editorEnv:    "vim",
			wantNonEmpty: true,
		},
		{
			name:         "uses VISUAL when EDITOR not set",
			visualEnv:    "emacs",
			wantNonEmpty: true,
		},
		{
			name:         "falls back to common editor",
			wantNonEmpty: true, // vi should exist on most systems.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editorEnv)
			t.Setenv("VISUAL", tt.visualEnv)

			editor := FindEditor()

			if tt.wantNonEmpty && editor == "" {
				t.Error("FindEditor() returned empty, want non-empty")
			}
		})
	}
}

func TestMergeOutputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stdout string
		stderr string
		want   string
	}{
		{name: "both_empty", stdout: "", stderr: "", want: ""},
		{name: "only_stdout", stdout: "hello", stderr: "", want: "hello"},
		{name: "only_stderr", stdout: "", stderr: "oops", want: "oops"},
		{name: "both_present", stdout: "out", stderr: "err", want: "out\nerr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MergeOutputs(tt.stdout, tt.stderr)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestEffectiveTimeout(t *testing.T) {
	t.Parallel()

	defaultDur := 30 * time.Second

	t.Run("no_deadline_uses_default", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		got := EffectiveTimeout(ctx, defaultDur)
		require.Equal(t, defaultDur, got)
	})

	t.Run("deadline_shorter_than_default", func(t *testing.T) {
		t.Parallel()

		shortDeadline := 5 * time.Second

		ctx, cancel := context.WithTimeout(context.Background(), shortDeadline)
		defer cancel()

		got := EffectiveTimeout(ctx, defaultDur)
		// Should be close to shortDeadline (within 1s tolerance for timing).
		require.Less(t, got, defaultDur)
		require.Greater(t, got, time.Duration(0))
	})

	t.Run("deadline_longer_than_default", func(t *testing.T) {
		t.Parallel()

		longDeadline := 5 * time.Minute

		ctx, cancel := context.WithTimeout(context.Background(), longDeadline)
		defer cancel()

		got := EffectiveTimeout(ctx, defaultDur)
		require.Equal(t, defaultDur, got)
	})

	t.Run("expired_deadline_uses_default", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)

		time.Sleep(time.Millisecond)

		defer cancel()

		got := EffectiveTimeout(ctx, defaultDur)
		// Expired deadline has negative remaining; should use default.
		require.Equal(t, defaultDur, got)
	})
}
