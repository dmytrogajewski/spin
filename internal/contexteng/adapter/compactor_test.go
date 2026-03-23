package adapter_test

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/contexteng/adapter"
	"github.com/dmytrogajewski/spin/internal/contexteng/compactor"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/pkg/tokenizer"
)

func TestCompactorAdapter_NoCompaction(t *testing.T) {
	t.Parallel()

	tok := &tokenizer.SimpleTokenizer{}
	maxCtx := 10000
	comp := compactor.NewCompactor(tok, maxCtx)
	adapt := adapter.NewCompactorAdapter(comp)

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "hello"},
	}

	result, changed, err := adapt.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if changed {
		t.Error("expected changed=false when no compaction needed")
	}

	if len(result) != len(msgs) {
		t.Errorf("expected %d messages, got %d", len(msgs), len(result))
	}
}

func TestCompactorAdapter_WarningStageReturnsFalse(t *testing.T) {
	t.Parallel()

	tok := &tokenizer.SimpleTokenizer{}
	// Set maxCtx so pressure is between 0.70 and 0.80 (warning stage).
	// Simple tokenizer: "word" = 1 token * 1.3 = 1 + 4 overhead = 5 per msg.
	// 10 messages x 5 tokens = 50 tokens. maxCtx = 65 -> pressure ~ 0.77.
	maxCtx := 65
	comp := compactor.NewCompactor(tok, maxCtx)
	adapt := adapter.NewCompactorAdapter(comp)

	msgs := make([]message.Message, 10)
	for i := range msgs {
		msgs[i] = message.Message{Role: message.RoleUser, Content: "word"}
	}

	_, changed, err := adapt.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if changed {
		t.Error("expected changed=false for warning stage (no modification)")
	}
}

func TestCompactorAdapter_ModifyingStagesReturnTrue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		maxCtx int
	}{
		{name: "observation_mask_stage", maxCtx: 60},
		{name: "fast_prune_stage", maxCtx: 55},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tok := &tokenizer.SimpleTokenizer{}
			comp := compactor.NewCompactor(tok, tt.maxCtx)
			adapt := adapter.NewCompactorAdapter(comp)

			msgs := make([]message.Message, 10)
			for i := range msgs {
				msgs[i] = message.Message{
					Role:    message.RoleTool,
					Content: "some tool output here",
				}
			}

			result, changed, err := adapt.Compact(context.Background(), msgs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !changed {
				t.Error("expected changed=true for modifying stage")
			}

			if len(result) != len(msgs) {
				t.Errorf("expected %d messages, got %d", len(msgs), len(result))
			}
		})
	}
}

func TestCompactorAdapter_NilCompactor(t *testing.T) {
	t.Parallel()

	adapt := adapter.NewCompactorAdapter(nil)

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "hello"},
	}

	result, changed, err := adapt.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if changed {
		t.Error("expected changed=false for nil compactor")
	}

	if len(result) != len(msgs) {
		t.Errorf("expected %d messages, got %d", len(msgs), len(result))
	}
}
