// Package prompt provides system prompt building and ACE enhancement logic.
package prompt

import (
	"context"
	"log/slog"
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace"
	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/agentsmd"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Builder constructs system prompts and message lists for LLM calls.
//
// It layers multiple prompt sources:
//   - AGENTS.md project instructions
//   - Task-specific system prompt
//   - XML tool definitions (for non-function-calling providers)
//   - ACE bullet context
type Builder struct {
	agentsMD   *agentsmd.Service
	aceService *ace.Service
	llm        llm.Provider
	logger     *slog.Logger
}

// New creates a Builder with the given dependencies.
func New(provider llm.Provider, logger *slog.Logger) *Builder {
	return &Builder{
		llm:    provider,
		logger: logger,
	}
}

// SetAgentsMD sets the AGENTS.md service for project-specific instructions.
func (pb *Builder) SetAgentsMD(svc *agentsmd.Service) {
	pb.agentsMD = svc
}

// SetACEService sets the ACE service for bullet-enhanced prompts.
func (pb *Builder) SetACEService(svc *ace.Service) {
	pb.aceService = svc
}

// BuildConversation builds the initial conversation messages from a request.
func (pb *Builder) BuildConversation(req *agent.Request) []message.Message {
	messages := make([]message.Message, 0, len(req.History)+1)
	if len(req.History) > 0 {
		messages = append(messages, req.History...)
	}

	messages = append(messages, message.Message{
		Role:    message.RoleUser,
		Content: req.Input,
	})

	return messages
}

// BuildSystemPrompt constructs the layered system prompt.
//
// When the provider does not support function calling, tool definitions are
// injected into the system prompt as XML formatting instructions so the model
// can emit tool calls in a parseable format.
func (pb *Builder) BuildSystemPrompt(ctx context.Context, basePrompt string, toolList []tools.Tool) string {
	var b strings.Builder

	if pb.agentsMD != nil && pb.agentsMD.IsLoaded() {
		b.WriteString("# Project Instructions\n\n")
		b.WriteString(pb.agentsMD.Content())
		b.WriteString("\n\n---\n\n")
		pb.logger.DebugContext(ctx, "injected AGENTS.md into system prompt",
			"path", pb.agentsMD.Path(), "size", len(pb.agentsMD.Content()))
	}

	if basePrompt != "" {
		b.WriteString(basePrompt)
	}

	if !pb.llm.Capabilities().FunctionCalling && len(toolList) > 0 {
		b.WriteString("\n\n")
		b.WriteString(tool.FormatToolsAsXMLPrompt(toolList))
		pb.logger.DebugContext(ctx, "injected XML tool definitions into system prompt",
			"tool_count", len(toolList))
	}

	return b.String()
}

// ApplyACEPrompt enhances the system prompt with ACE bullets if available.
func (pb *Builder) ApplyACEPrompt(ctx context.Context, prompt string, bullets []*bullet.Bullet) string {
	if pb.aceService == nil {
		return prompt
	}

	acePrompt, err := pb.aceService.BuildPrompt(ctx, prompt, bullets)
	if err != nil {
		pb.logger.WarnContext(ctx, "ACE prompt building failed", "error", err)

		return prompt
	}

	pb.logger.DebugContext(ctx, "ACE enhanced system prompt", "bullets_count", len(bullets))

	return acePrompt
}
