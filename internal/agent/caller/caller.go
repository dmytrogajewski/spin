// Package caller provides LLM interaction, streaming, and retry logic.
package caller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/sanitizer"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
	spinerrors "github.com/dmytrogajewski/spin/pkg/apperr"
)

const (
	defaultTimeout = 5 * time.Minute
	maxRetries     = 3
)

// ErrContextDeadlineExceeded is returned when context deadline is exceeded before LLM call.
var ErrContextDeadlineExceeded = errors.New("caller: context canceled: context deadline exceeded")

// SystemPromptBuilder builds system prompts and applies ACE enhancements.
type SystemPromptBuilder interface {
	BuildSystemPrompt(ctx context.Context, basePrompt string, toolList []tools.Tool) string
	ApplyACEPrompt(ctx context.Context, prompt string, bullets []*bullet.Bullet) string
}

// LLMCaller encapsulates all LLM interaction: message construction, streaming,
// response parsing, and retry logic. It is the only component that directly
// uses openai-go types.
type LLMCaller struct {
	provider      llm.Provider
	promptBuilder SystemPromptBuilder
	emitter       *events.EventEmitter
	logger        *slog.Logger
	temperature   float64
	maxTokens     int
}

// Config holds configuration for constructing an LLMCaller.
type Config struct {
	// Provider is used directly when Router is nil (backward compatibility).
	Provider llm.Provider

	// Router enables multi-model routing. When set, the provider is resolved
	// via Router.ForRole(Role) and Provider field is ignored.
	Router *llm.Router

	// Role selects which model role this caller uses (default: RoleAction).
	// Only used when Router is set.
	Role llm.Role

	PromptBuilder SystemPromptBuilder
	Emitter       *events.EventEmitter
	Logger        *slog.Logger
	Temperature   float64
	MaxTokens     int
}

// New creates a new LLMCaller.
// When cfg.Router is set, the provider is resolved from the router using cfg.Role.
// Otherwise cfg.Provider is used directly.
func New(cfg Config) *LLMCaller {
	provider := cfg.Provider

	if cfg.Router != nil {
		role := cfg.Role
		if role == "" {
			role = llm.RoleAction
		}

		provider = cfg.Router.ForRole(role)
	}

	return &LLMCaller{
		provider:      provider,
		promptBuilder: cfg.PromptBuilder,
		emitter:       cfg.Emitter,
		logger:        cfg.Logger,
		temperature:   cfg.Temperature,
		maxTokens:     cfg.MaxTokens,
	}
}

// Call calls the LLM provider with streaming, returning the accumulated response.
func (lc *LLMCaller) Call(
	ctx context.Context, messages []message.Message,
	cp agent.CallParams, toolList []tools.Tool, bullets []*bullet.Bullet,
) (*openai.ChatCompletion, error) {
	openaiMessages := lc.buildOpenAIMessages(ctx, messages, cp, bullets)

	lc.logToolCallMessages(ctx, messages)

	params := lc.buildLLMParams(ctx, openaiMessages, cp, toolList)

	chunks, err := lc.provider.Stream(ctx, params)
	if err != nil {
		return nil, spinerrors.New(spinerrors.CodeLLM, "LLMCaller.Call", "failed to start LLM stream", err)
	}

	response, err := lc.processStream(ctx, chunks)
	if err != nil {
		return nil, err
	}

	lc.applyFinishReasonFallback(response)
	lc.extractXMLToolCalls(ctx, response)

	return response, nil
}

// CallWithTimeout wraps Call with a per-call timeout.
func (lc *LLMCaller) CallWithTimeout(
	ctx context.Context, messages []message.Message,
	cp agent.CallParams, toolList []tools.Tool, bullets []*bullet.Bullet,
) (*openai.ChatCompletion, error) {
	timeout := defaultTimeout

	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, ErrContextDeadlineExceeded
		}

		if remaining < timeout {
			timeout = remaining
		}
	}

	lc.logger.DebugContext(ctx, "calling LLM with timeout",
		"timeout", timeout, "message_count", len(messages),
		"bullets_count", len(bullets))

	llmCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := lc.Call(llmCtx, messages, cp, toolList, bullets)
	if err != nil {
		lc.logger.ErrorContext(ctx, "LLM call error", "error", err, "timeout", timeout)
	}

	return resp, err
}

// CallWithRetries wraps CallWithTimeout with retry logic for transient errors.
func (lc *LLMCaller) CallWithRetries(
	ctx context.Context, messages []message.Message,
	cp agent.CallParams, toolList []tools.Tool, bullets []*bullet.Bullet,
	turn int, resp *agent.Response,
) (content string, toolCalls []agent.ToolCall, finishReason string, err error) {
	for retry := 0; retry <= maxRetries; retry++ {
		llmResp, callErr := lc.CallWithTimeout(ctx, messages, cp, toolList, bullets)
		if callErr != nil {
			shouldReturn, retErr := lc.handleLLMError(ctx, callErr, retry, turn, resp)
			if shouldReturn {
				return "", nil, "", retErr
			}

			continue
		}

		respContent := getContent(llmResp)
		respToolCalls := getToolCalls(llmResp)
		respFinishReason := getFinishReason(llmResp)

		if llmResp != nil && (respContent != "" || len(respToolCalls) > 0) {
			if retry > 0 {
				lc.logger.InfoContext(ctx, "LLM retry succeeded", "turn", turn+1, "retry", retry)
			}

			return respContent, respToolCalls, respFinishReason, nil
		}

		if done, retErr := lc.handleEmptyResponse(ctx, retry, turn, resp, llmResp, respContent, respToolCalls); done {
			return "", nil, "", retErr
		}
	}

	return "", nil, "", nil
}

func (lc *LLMCaller) buildOpenAIMessages(
	ctx context.Context, messages []message.Message,
	cp agent.CallParams, bullets []*bullet.Bullet,
) []openai.ChatCompletionMessageParamUnion {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)

	var toolList []tools.Tool
	if !lc.provider.Capabilities().FunctionCalling {
		toolList = nil
	}

	enhancedSystemPrompt := lc.promptBuilder.BuildSystemPrompt(ctx, cp.SystemPrompt, toolList)

	if enhancedSystemPrompt != "" {
		enhancedSystemPrompt = lc.promptBuilder.ApplyACEPrompt(ctx, enhancedSystemPrompt, bullets)
		openaiMessages = append(openaiMessages, openai.SystemMessage(enhancedSystemPrompt))
	}

	for _, msg := range messages {
		openaiMessages = append(openaiMessages, convertMessageToOpenAI(msg))
	}

	return openaiMessages
}

func (lc *LLMCaller) buildLLMParams(
	ctx context.Context,
	openaiMessages []openai.ChatCompletionMessageParamUnion,
	cp agent.CallParams,
	toolList []tools.Tool,
) openai.ChatCompletionNewParams {
	maxTokens := lc.maxTokens

	if cp.MaxTokens > 0 {
		maxTokens = cp.MaxTokens
	}

	params := openai.ChatCompletionNewParams{
		Messages:    openaiMessages,
		Temperature: openai.Float(lc.temperature),
		MaxTokens:   openai.Int(int64(maxTokens)),
	}

	if len(toolList) > 0 {
		if lc.provider.Capabilities().FunctionCalling {
			params.Tools = convertToolsToOpenAI(toolList)
		} else {
			lc.logger.InfoContext(ctx, "using XML tool calling (provider does not support function calling)",
				"provider", lc.provider.Name(), "tool_count", len(toolList))
		}
	}

	lc.logger.DebugContext(ctx, "calling LLM", "tool_count", len(toolList), "message_count", len(openaiMessages))

	return params
}

func (lc *LLMCaller) processStream(ctx context.Context, chunks <-chan openai.ChatCompletionChunk) (*openai.ChatCompletion, error) {
	acc := openai.ChatCompletionAccumulator{}
	streamSanitizer := sanitizer.New()
	chunkCount := 0

	for chunk := range chunks {
		chunkCount++

		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, spinerrors.New(spinerrors.CodeTimeout, "LLMCaller.Call", "context canceled", ctxErr)
		}

		acc.AddChunk(chunk)

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		lc.emitStreamDeltas(ctx, choice.Delta, streamSanitizer, chunkCount)
		lc.handleToolCallFinished(ctx, &acc, choice)

		if choice.FinishReason != "" {
			lc.logger.DebugContext(ctx, "received finish chunk", "finish_reason", choice.FinishReason, "total_chunks", chunkCount)
		}
	}

	return lc.finalizeStreamResponse(ctx, &acc, chunkCount)
}

func (lc *LLMCaller) emitStreamDeltas(
	ctx context.Context, delta openai.ChatCompletionChunkChoiceDelta,
	streamSanitizer *sanitizer.Sanitizer, chunkCount int,
) {
	if delta.Content == "" {
		return
	}

	content, thought := streamSanitizer.Process(delta.Content)

	if content != "" {
		lc.emitter.Emit(events.Event{
			Type:      events.EventContentDelta,
			Timestamp: time.Now(),
			Data: events.ContentDeltaData{
				Content: content,
				Role:    string(message.RoleAssistant),
			},
		})
		lc.logger.DebugContext(ctx, "received content chunk", "count", chunkCount, "content_len", len(content))
	}

	if thought != "" {
		lc.emitter.Emit(events.Event{
			Type:      events.EventThinkingDelta,
			Timestamp: time.Now(),
			Data: events.ThinkingDeltaData{
				Content: thought,
			},
		})
		lc.logger.DebugContext(ctx, "received thinking chunk", "count", chunkCount, "content_len", len(thought))
	}
}

func (lc *LLMCaller) handleToolCallFinished(
	ctx context.Context, acc *openai.ChatCompletionAccumulator,
	choice openai.ChatCompletionChunkChoice,
) {
	toolCall, finished := acc.JustFinishedToolCall()
	if !finished && choice.FinishReason == llm.FinishReasonToolCalls {
		finished = true
	}

	if !finished {
		return
	}

	if toolCall.Name != "" {
		lc.logger.DebugContext(ctx, "tool call finished",
			"index", toolCall.Index, "name", toolCall.Name,
			"args_len", len(toolCall.Arguments))
	}
}

func (lc *LLMCaller) finalizeStreamResponse(
	ctx context.Context, acc *openai.ChatCompletionAccumulator, chunkCount int,
) (*openai.ChatCompletion, error) {
	response := &acc.ChatCompletion

	if len(response.Choices) == 0 {
		lc.logger.WarnContext(ctx, "stream ended with no choices", "total_chunks", chunkCount)

		if chunkCount == 0 {
			return nil, spinerrors.New(
				spinerrors.CodeLLM, "LLMCaller.Call",
				"stream returned no chunks - possible connection error or empty response from LLM",
				nil,
			)
		}

		return nil, spinerrors.New(spinerrors.CodeLLM, "LLMCaller.Call", "no choices in response after processing chunks", nil)
	}

	lc.logger.DebugContext(ctx, "stream ended",
		"total_chunks", chunkCount,
		"content_len", len(response.Choices[0].Message.Content),
		"tool_calls", len(response.Choices[0].Message.ToolCalls))

	if err := ctx.Err(); err != nil {
		return nil, spinerrors.New(spinerrors.CodeTimeout, "LLMCaller.Call", "context canceled", err)
	}

	return response, nil
}

func (lc *LLMCaller) applyFinishReasonFallback(response *openai.ChatCompletion) {
	if response.Choices[0].FinishReason != "" {
		return
	}

	if len(response.Choices[0].Message.ToolCalls) > 0 {
		response.Choices[0].FinishReason = llm.FinishReasonToolCalls
	} else {
		response.Choices[0].FinishReason = llm.FinishReasonStop
	}
}

func (lc *LLMCaller) extractXMLToolCalls(ctx context.Context, response *openai.ChatCompletion) {
	if len(response.Choices[0].Message.ToolCalls) > 0 {
		return
	}

	content := response.Choices[0].Message.Content
	xmlToolCalls, warnings := tool.ParseToolCallsFromXML(content)

	for _, w := range warnings {
		lc.logger.WarnContext(ctx, "XML tool call parse warning", "warning", w)
	}

	if len(xmlToolCalls) == 0 {
		return
	}

	if lc.provider.Capabilities().FunctionCalling {
		lc.logger.WarnContext(ctx, "model emitted XML tool calls despite supporting function calling — using fallback recovery",
			"provider", lc.provider.Name(), "count", len(xmlToolCalls))
	} else {
		lc.logger.InfoContext(ctx, "extracted XML tool calls", "count", len(xmlToolCalls))
	}

	response.Choices[0].Message.ToolCalls = xmlToolCalls
	response.Choices[0].FinishReason = llm.FinishReasonToolCalls

	s := sanitizer.New()
	cleanContent, cleanThought := s.Process(content)

	var sb strings.Builder
	if cleanThought != "" {
		sb.WriteString("<think>")
		sb.WriteString(cleanThought)
		sb.WriteString("</think>\n")
	}

	sb.WriteString(cleanContent)
	response.Choices[0].Message.Content = sb.String()
}

func (lc *LLMCaller) logToolCallMessages(ctx context.Context, messages []message.Message) {
	for i, msg := range messages {
		if msg.Role != message.RoleAssistant || len(msg.ToolCalls) == 0 {
			continue
		}

		ids := make([]string, len(msg.ToolCalls))
		for j, tc := range msg.ToolCalls {
			ids[j] = tc.ID
		}

		lc.logger.DebugContext(ctx, "LLMCaller: assistant message with tool_calls",
			"msg_index", i,
			"tool_call_count", len(msg.ToolCalls),
			"tool_call_ids", ids)
	}
}

func (lc *LLMCaller) handleLLMError(
	ctx context.Context, err error, retry, turn int,
	resp *agent.Response,
) (shouldReturn bool, retErr error) {
	if ctx.Err() != nil {
		lc.logger.ErrorContext(ctx, "LLM call failed (context canceled)", "turn", turn+1, "error", err)
		resp.Error = fmt.Errorf("llm call failed: %w", err)
		resp.FinishReason = "error"

		return true, err
	}

	if retry < maxRetries {
		if waitErr := lc.waitWithBackoff(ctx, retry, turn, resp, "LLM call failed, retrying"); waitErr != nil {
			return true, waitErr
		}

		return false, nil
	}

	lc.logger.ErrorContext(ctx, "LLM call failed after retries", "turn", turn+1, "retries", maxRetries, "error", err)
	resp.Error = fmt.Errorf("llm call failed: %w", err)
	resp.FinishReason = "error"

	return true, err
}

func (lc *LLMCaller) handleEmptyResponse(
	ctx context.Context, retry, turn int,
	resp *agent.Response, llmResp *openai.ChatCompletion,
	content string, toolCalls []agent.ToolCall,
) (bool, error) {
	if retry < maxRetries {
		if err := lc.waitWithBackoff(ctx, retry, turn, resp, "Received empty response from LLM, retrying"); err != nil {
			return true, err
		}

		return false, nil
	}

	lc.emitEmptyResponseWarning(ctx, turn, llmResp, content, toolCalls)

	resp.FinishReason = "empty_response"

	return true, nil
}

func (lc *LLMCaller) waitWithBackoff(ctx context.Context, retry, turn int, resp *agent.Response, logMsg string) error {
	lc.logger.WarnContext(ctx, logMsg, "turn", turn+1, "retry", retry+1)

	if err := concurrency.SleepWithBackoff(ctx, retry, time.Second); err != nil {
		resp.FinishReason = "timeout"

		return fmt.Errorf("retry context canceled: %w", ctx.Err())
	}

	return nil
}

func (lc *LLMCaller) emitEmptyResponseWarning(
	ctx context.Context, turn int,
	llmResp *openai.ChatCompletion, content string, toolCalls []agent.ToolCall,
) {
	lc.logger.WarnContext(ctx, "Received empty response from LLM after retries, breaking loop",
		"turn", turn+1, "retries_exhausted", maxRetries,
		"llm_resp_nil", llmResp == nil, "content_len", len(content), "tool_calls", len(toolCalls))

	lc.emitter.Emit(events.Event{
		Type:      events.EventWarning,
		Timestamp: time.Now(),
		Data: events.SystemEventData{
			Level:   "warning",
			Message: "LLM returned empty response after retries",
			Details: fmt.Sprintf("turn=%d, retries=%d", turn+1, maxRetries),
		},
	})
}
