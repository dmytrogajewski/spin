package compactor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/contexteng/compactor"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/pkg/tokenizer"
)

// Journey: specs/journeys/JOURNEY-2.3.md.

const testMaxContext = 1000

// fixedTokenizer always returns a fixed count per call.
type fixedTokenizer struct {
	countPerCall int
}

func (f *fixedTokenizer) Count(_ string) int {
	return f.countPerCall
}

func makeMessages(n int, role message.Role, content string) []message.Message {
	msgs := make([]message.Message, n)

	for idx := range msgs {
		msgs[idx] = message.Message{
			Role:    role,
			Content: content,
		}
	}

	return msgs
}

// TestPressure_Calculation verifies pressure ratio calculation.
// Kills mutant: wrong division would give incorrect pressure.
func TestPressure_Calculation(t *testing.T) {
	t.Parallel()

	tok := &tokenizer.SimpleTokenizer{}
	comp := compactor.NewCompactor(tok, testMaxContext)

	msgs := makeMessages(1, message.RoleUser, "hello world")
	pressure := comp.Pressure(msgs)

	assert.Positive(t, pressure)
	assert.Less(t, pressure, 1.0)
}

// TestPressure_EmptyMessages verifies zero pressure for empty input.
// Kills mutant: empty messages with overhead would give non-zero.
func TestPressure_EmptyMessages(t *testing.T) {
	t.Parallel()

	tok := &fixedTokenizer{countPerCall: 0}
	comp := compactor.NewCompactor(tok, testMaxContext)

	pressure := comp.Pressure(nil)

	assert.Equal(t, 0.0, pressure)
}

// TestPressure_ZeroMaxContext verifies zero pressure when max context is zero.
// Kills mutant: division by zero would panic.
func TestPressure_ZeroMaxContext(t *testing.T) {
	t.Parallel()

	tok := &fixedTokenizer{countPerCall: 100}
	comp := compactor.NewCompactor(tok, 0)

	pressure := comp.Pressure(makeMessages(1, message.RoleUser, "hello"))

	assert.Equal(t, 0.0, pressure)
}

// TestCompact_BelowWarning verifies StageNone when pressure is low.
// Kills mutant: applying compaction below threshold would damage messages.
func TestCompact_BelowWarning(t *testing.T) {
	t.Parallel()

	// 10 messages * (50 tokens + 4 overhead) = 540 tokens. 540/1000 = 0.54.
	tok := &fixedTokenizer{countPerCall: 50}
	comp := compactor.NewCompactor(tok, testMaxContext)
	msgs := makeMessages(10, message.RoleUser, "content")

	result, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageNone, stage)
	assert.Equal(t, msgs, result)
}

// TestCompact_WarningStage verifies StageWarning at 70-80% pressure.
// Kills mutant: modifying messages at warning stage would lose data.
func TestCompact_WarningStage(t *testing.T) {
	t.Parallel()

	// 10 messages * (71 tokens + 4 overhead) = 750 tokens. 750/1000 = 0.75.
	tok := &fixedTokenizer{countPerCall: 71}
	comp := compactor.NewCompactor(tok, testMaxContext)
	msgs := makeMessages(10, message.RoleUser, "content")

	result, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageWarning, stage)
	assert.Equal(t, msgs, result)
}

// TestCompact_ObservationMask verifies tool outputs are masked at 80-85% pressure.
// Kills mutant: not masking tool outputs would leave pressure high.
func TestCompact_ObservationMask(t *testing.T) {
	t.Parallel()

	// 10 messages * (78 tokens + 4 overhead) = 820 tokens. 820/1000 = 0.82.
	tok := &fixedTokenizer{countPerCall: 78}
	comp := compactor.NewCompactor(tok, testMaxContext)

	msgs := make([]message.Message, 10)
	for idx := range msgs {
		msgs[idx] = message.Message{
			Role:    message.RoleTool,
			Content: "long tool output here",
		}
	}

	result, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageObservationMask, stage)

	// First 4 messages (10-6 recent protected) should be masked.
	for idx := range 4 {
		assert.Contains(t, result[idx].Content, "[observation:")
		assert.Contains(t, result[idx].Content, "tokens")
	}

	// Last 6 (recent protected) should be unchanged.
	for idx := 4; idx < 10; idx++ {
		assert.Equal(t, "long tool output here", result[idx].Content)
	}
}

// TestCompact_FastPrune verifies pruning at >= 85% pressure.
// Kills mutant: not pruning would leave context too large.
func TestCompact_FastPrune(t *testing.T) {
	t.Parallel()

	// 10 messages * (83 tokens + 4 overhead) = 870 tokens. 870/1000 = 0.87.
	tok := &fixedTokenizer{countPerCall: 83}
	comp := compactor.NewCompactor(tok, testMaxContext)

	msgs := make([]message.Message, 10)
	for idx := range msgs {
		msgs[idx] = message.Message{
			Role:    message.RoleTool,
			Content: "tool result data",
		}
	}

	result, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageFastPrune, stage)

	// First 4 should be pruned.
	for idx := range 4 {
		assert.Equal(t, "[pruned]", result[idx].Content)
	}

	// Last 6 should be preserved.
	for idx := 4; idx < 10; idx++ {
		assert.Equal(t, "tool result data", result[idx].Content)
	}
}

// TestCompact_ObservationMask_PreservesNonToolMessages verifies only tool messages are masked.
// Kills mutant: masking user/assistant messages would corrupt conversation.
func TestCompact_ObservationMask_PreservesNonToolMessages(t *testing.T) {
	t.Parallel()

	// 12 messages * (63 tokens + 4 overhead) = 804 tokens. 804/1000 = 0.804 → observation mask.
	tok := &fixedTokenizer{countPerCall: 63}
	comp := compactor.NewCompactor(tok, testMaxContext)

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "question"},
		{Role: message.RoleTool, Content: "tool output"},
		{Role: message.RoleAssistant, Content: "answer"},
		{Role: message.RoleTool, Content: "another tool output"},
		// Recent protected (6 slots, so with 4 messages all could be protected).
	}

	// Need enough messages to have some outside the protection window.
	// With 4 messages and protection=6, all are protected. Add more.
	for range 8 {
		msgs = append(msgs, message.Message{
			Role: message.RoleTool, Content: "filler tool output",
		})
	}

	result, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageObservationMask, stage)

	// Non-tool messages should be preserved.
	assert.Equal(t, "question", result[0].Content)
	assert.Equal(t, "answer", result[2].Content)

	// Tool messages outside protection window should be masked.
	assert.Contains(t, result[1].Content, "[observation:")
	assert.Contains(t, result[3].Content, "[observation:")
}

// TestCompact_FastPrune_PreservesNonToolMessages verifies only tool messages are pruned.
// Kills mutant: pruning user messages would corrupt conversation.
func TestCompact_FastPrune_PreservesNonToolMessages(t *testing.T) {
	t.Parallel()

	tok := &fixedTokenizer{countPerCall: 83}
	comp := compactor.NewCompactor(tok, testMaxContext)

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "question"},
		{Role: message.RoleTool, Content: "tool output"},
		{Role: message.RoleAssistant, Content: "answer"},
	}

	for range 8 {
		msgs = append(msgs, message.Message{
			Role: message.RoleTool, Content: "filler",
		})
	}

	result, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageFastPrune, stage)
	assert.Equal(t, "question", result[0].Content)
	assert.Equal(t, "answer", result[2].Content)
	assert.Equal(t, "[pruned]", result[1].Content)
}

// TestCompact_WithCustomThresholds verifies configurable thresholds.
// Kills mutant: ignoring options would use wrong thresholds.
func TestCompact_WithCustomThresholds(t *testing.T) {
	t.Parallel()

	// Custom thresholds: warning at 50%, observe at 60%, prune at 70%.
	tok := &fixedTokenizer{countPerCall: 61}
	comp := compactor.NewCompactor(tok, testMaxContext,
		compactor.WithThresholds(0.50, 0.60, 0.70),
	)

	// 10 messages * (61 + 4) = 650 tokens. 650/1000 = 0.65 → observation mask with custom thresholds.
	msgs := makeMessages(10, message.RoleTool, "output")

	_, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageObservationMask, stage)
}

// TestCompact_EmptyMessages verifies no-op for empty input.
// Kills mutant: processing empty input would waste cycles or panic.
func TestCompact_EmptyMessages(t *testing.T) {
	t.Parallel()

	tok := &fixedTokenizer{countPerCall: 100}
	comp := compactor.NewCompactor(tok, testMaxContext)

	result, stage, err := comp.Compact(t.Context(), nil)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageNone, stage)
	assert.Empty(t, result)
}

// TestStage_String verifies stage string representation.
// Kills mutant: wrong string mapping would confuse logging.
func TestStage_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "none", compactor.StageNone.String())
	assert.Equal(t, "warning", compactor.StageWarning.String())
	assert.Equal(t, "observation_mask", compactor.StageObservationMask.String())
	assert.Equal(t, "fast_prune", compactor.StageFastPrune.String())
	assert.Contains(t, compactor.Stage(99).String(), "unknown")
}

// TestWithRecentProtected verifies custom protection window.
// Kills mutant: ignoring protection count would prune recent messages.
func TestWithRecentProtected(t *testing.T) {
	t.Parallel()

	const protectedCount = 2

	tok := &fixedTokenizer{countPerCall: 83}
	comp := compactor.NewCompactor(tok, testMaxContext,
		compactor.WithRecentProtected(protectedCount),
	)

	msgs := makeMessages(10, message.RoleTool, "tool output")

	result, stage, err := comp.Compact(t.Context(), msgs)

	require.NoError(t, err)
	assert.Equal(t, compactor.StageFastPrune, stage)

	// First 8 should be pruned (10 - 2 protected).
	for idx := range 8 {
		assert.Equal(t, "[pruned]", result[idx].Content)
	}

	// Last 2 should be preserved.
	for idx := 8; idx < 10; idx++ {
		assert.Equal(t, "tool output", result[idx].Content)
	}
}
