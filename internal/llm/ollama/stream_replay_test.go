package ollama

import (
	"testing"

	"github.com/ollama/ollama/api"
)

// Journey: specs/bugs/BUG-tui-startup-and-double-reply.md.

func TestIsFullStreamReplay_DoneRepeatsVisibleText(t *testing.T) {
	t.Parallel()

	already := mergeThinkingContent("plan", "") + mergeThinkingContent("", "Hello")
	done := mergeThinkingContent("plan", "Hello")

	if !isFullStreamReplay(done, already) {
		t.Fatalf("done chunk %q should be a replay of %q", done, already)
	}
}

func TestIsFullStreamReplay_LastTokenIsNotReplay(t *testing.T) {
	t.Parallel()

	already := "Hell"
	last := "o"

	if isFullStreamReplay(last, already) {
		t.Fatalf("last token %q must still stream after %q", last, already)
	}
}

func TestApplyStreamMerge_SkipsDoneReplay(t *testing.T) {
	t.Parallel()

	streamed := mergeThinkingContent("ab", "") + "Hi"

	resp := api.ChatResponse{
		Done: true,
		Message: api.Message{
			Thinking: "ab",
			Content:  "Hi",
		},
	}

	got, skip := applyStreamMerge(resp, streamed)
	if !skip || got != "" {
		t.Fatalf("replay should skip, got %q skip=%v", got, skip)
	}
}

func TestApplyStreamMerge_KeepsIncremental(t *testing.T) {
	t.Parallel()

	resp := api.ChatResponse{
		Message: api.Message{Thinking: "a"},
	}

	got, skip := applyStreamMerge(resp, "")
	want := mergeThinkingContent("a", "")

	if skip || got != want {
		t.Fatalf("incremental thinking: got %q skip=%v, want %q", got, skip, want)
	}
}
