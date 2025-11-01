package prompt

import (
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder_Default(t *testing.T) {
	builder := NewBuilder()

	require.NotNil(t, builder)
	assert.Equal(t, "You are a helpful assistant.", builder.systemPrompt)
	assert.False(t, builder.includeIL)
}

func TestNewBuilder_WithOptions(t *testing.T) {
	builder := NewBuilder(
		WithSystemPrompt("Custom prompt"),
		WithItemizedLearning(),
	)

	require.NotNil(t, builder)
	assert.Equal(t, "Custom prompt", builder.systemPrompt)
	assert.True(t, builder.includeIL)
}

func TestBuilder_FormatBullet(t *testing.T) {
	builder := NewBuilder()
	b, err := bullet.New("Always validate input")
	require.NoError(t, err)

	formatted := builder.FormatBullet(b, 0)
	assert.Equal(t, "[B0] Always validate input", formatted)

	formatted = builder.FormatBullet(b, 5)
	assert.Equal(t, "[B5] Always validate input", formatted)
}

func TestBuilder_BuildSystemPrompt_Empty(t *testing.T) {
	builder := NewBuilder()
	prompt := builder.BuildSystemPrompt([]*bullet.Bullet{})

	assert.Contains(t, prompt, "You are a helpful assistant")
	assert.NotContains(t, prompt, "# Context Playbook")
	assert.NotContains(t, prompt, "# Instructions")
}

func TestBuilder_BuildSystemPrompt_WithBullets(t *testing.T) {
	builder := NewBuilder()

	b1, err := bullet.New("Validate input")
	require.NoError(t, err)
	b2, err := bullet.New("Use context.Context")
	require.NoError(t, err)

	prompt := builder.BuildSystemPrompt([]*bullet.Bullet{b1, b2})

	assert.Contains(t, prompt, "You are a helpful assistant")
	assert.Contains(t, prompt, "# Context Playbook")
	assert.Contains(t, prompt, "[B0] Validate input")
	assert.Contains(t, prompt, "[B1] Use context.Context")
	assert.NotContains(t, prompt, "# Instructions")
}

func TestBuilder_BuildSystemPrompt_WithIL(t *testing.T) {
	builder := NewBuilder(WithItemizedLearning())

	b1, err := bullet.New("Test bullet")
	require.NoError(t, err)

	prompt := builder.BuildSystemPrompt([]*bullet.Bullet{b1})

	assert.Contains(t, prompt, "# Context Playbook")
	assert.Contains(t, prompt, "[B0] Test bullet")
	assert.Contains(t, prompt, "# Instructions")
	assert.Contains(t, prompt, "HELPFUL:")
	assert.Contains(t, prompt, "HARMFUL:")
}

func TestBuilder_BuildSystemPrompt_CustomSystem(t *testing.T) {
	builder := NewBuilder(WithSystemPrompt("You are an expert Go developer"))

	b1, err := bullet.New("Use goroutines")
	require.NoError(t, err)

	prompt := builder.BuildSystemPrompt([]*bullet.Bullet{b1})

	assert.Contains(t, prompt, "You are an expert Go developer")
	assert.NotContains(t, prompt, "helpful assistant")
	assert.Contains(t, prompt, "[B0] Use goroutines")
}

func TestBuilder_BuildSystemPrompt_MultipleBullets(t *testing.T) {
	builder := NewBuilder(WithItemizedLearning())

	bullets := make([]*bullet.Bullet, 5)
	for i := 0; i < 5; i++ {
		b, err := bullet.New("Bullet " + string(rune('A'+i)))
		require.NoError(t, err)
		bullets[i] = b
	}

	prompt := builder.BuildSystemPrompt(bullets)

	// Check all bullets are present with correct indices
	for i := 0; i < 5; i++ {
		expected := "[B" + string(rune('0'+i)) + "]"
		assert.Contains(t, prompt, expected)
	}

	// Check structure
	lines := strings.Split(prompt, "\n")
	assert.NotEmpty(t, lines)
}
