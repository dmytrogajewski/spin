package prompt

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

// Builder constructs prompts with context bullets.
type Builder struct {
	systemPrompt string
	includeIL    bool // Include ItemizedLearning instructions
}

// Option configures Builder.
type Option func(*Builder)

// WithSystemPrompt sets custom system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(b *Builder) {
		b.systemPrompt = prompt
	}
}

// WithItemizedLearning enables IL instructions.
func WithItemizedLearning() Option {
	return func(b *Builder) {
		b.includeIL = true
	}
}

// NewBuilder creates a prompt builder.
func NewBuilder(opts ...Option) *Builder {
	b := &Builder{
		systemPrompt: "You are a helpful assistant.",
		includeIL:    false,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// BuildSystemPrompt constructs system prompt with bullets.
func (b *Builder) BuildSystemPrompt(bullets []*bullet.Bullet) string {
	var sb strings.Builder

	// Start with custom system prompt
	sb.WriteString(b.systemPrompt)
	sb.WriteString("\n\n")

	// Add context playbook section if bullets provided
	if len(bullets) > 0 {
		sb.WriteString("# Context Playbook\n\n")
		for i, bullet := range bullets {
			formatted := b.FormatBullet(bullet, i)
			sb.WriteString(formatted)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Add ItemizedLearning instructions if enabled
	if b.includeIL && len(bullets) > 0 {
		sb.WriteString(itemizedLearningInstructions)
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatBullet formats a bullet with marker for IL.
func (b *Builder) FormatBullet(bullet *bullet.Bullet, index int) string {
	return fmt.Sprintf("[B%d] %s", index, bullet.Content)
}

const itemizedLearningInstructions = `# Instructions for Using the Playbook

**IMPORTANT**: Read the playbook FIRST before solving the task. Explicitly leverage relevant sections in your approach.

## 1. ANALYSIS & STRATEGY

- Carefully analyze both the task and playbook before starting
- Search for and identify any applicable patterns, strategies, or examples within the playbook
- Create a structured approach to solving the problem at hand
- Review and document any limitations in the provided reference materials

## 2. SOLUTION DEVELOPMENT

- Present your solution using clear, logical steps that others can follow and review
- Explain your reasoning and methodology before presenting final conclusions
- Provide detailed explanations for each step of the process
- Check and verify all assumptions and intermediate calculations

## 3. CODING TASKS

When coding is required:
- Write clean, efficient code following best practices from the playbook
- Include clear inline comments to explain any complex logic
- Perform result validation after execution
- Apply optimization techniques from the playbook when applicable
- Handle errors gracefully and provide helpful error messages

## 4. FEEDBACK (IMPORTANT)

After solving the task, indicate which bullets were helpful or harmful:
- HELPFUL: [B0, B2, B5] - bullets that helped solve the task correctly
- HARMFUL: [B3] - bullets that misled or were incorrect
- EXPLANATION: Brief reasoning for your feedback

**NOTE**: Treat the playbook as a tool. Use only the parts that are relevant and applicable to your specific situation and task context. Use your own judgment for aspects not covered by the playbook.

Format your response with the feedback markers at the end.`
