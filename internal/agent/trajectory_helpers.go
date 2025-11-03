package agent

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/message"
)

// extractInitialQuery extracts the user's initial query from messages.
// Returns first user message content, or empty string if none.
func extractInitialQuery(messages []message.Message) string {
	for _, msg := range messages {
		if msg.Role == message.RoleUser {
			return msg.Content
		}
	}
	return ""
}

// extractNewSteps extracts TrajectorySteps from messages starting from lastStepNumber.
// Returns steps in chronological order with proper step numbering.
func extractNewSteps(messages []message.Message, lastStepNumber int) []generator.TrajectoryStep {
	steps := make([]generator.TrajectoryStep, 0)
	stepNum := lastStepNumber

	for _, msg := range messages {
		timestamp := msg.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now()
		}

		switch msg.Role {
		case message.RoleAssistant:
			// Reasoning
			if msg.Content != "" {
				steps = append(steps, generator.TrajectoryStep{
					StepNumber: stepNum,
					Type:       "reasoning",
					Content:    msg.Content,
					Timestamp:  timestamp,
				})
				stepNum++
			}

			// Tool calls
			for _, tc := range msg.ToolCalls {
				content := fmt.Sprintf("Tool: %s\nArguments: %s",
					tc.Function.Name, tc.Function.Arguments)
				steps = append(steps, generator.TrajectoryStep{
					StepNumber: stepNum,
					Type:       "tool_call",
					Content:    content,
					Timestamp:  timestamp,
				})
				stepNum++
			}

		case message.RoleTool:
			// Tool results
			content := fmt.Sprintf("Tool Result (ID: %s):\n%s",
				msg.ToolCallID, msg.Content)
			steps = append(steps, generator.TrajectoryStep{
				StepNumber: stepNum,
				Type:       "tool_result",
				Content:    content,
				Timestamp:  timestamp,
			})
			stepNum++
		}
	}

	return steps
}

// extractBulletIDs extracts bullet IDs from bullet slice.
func extractBulletIDs(bullets []*bullet.Bullet) []string {
	ids := make([]string, len(bullets))
	for i, b := range bullets {
		ids[i] = b.ID
	}
	return ids
}
