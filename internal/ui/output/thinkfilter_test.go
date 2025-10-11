package output

import (
	"strings"
	"testing"
	"time"
)

func TestThinkFilter_SimpleThinkBlock(t *testing.T) {
	f := NewThinkFilter()

	// Simulate streaming chunks
	chunks := []string{
		"Before ",
		"<think>I need to ",
		"think about this",
		"</think> After",
	}

	var output strings.Builder
	for _, chunk := range chunks {
		output.WriteString(f.Process(chunk))
	}

	result := output.String()

	// Should have "Before ", dim thinking content, summary, " After"
	if !strings.Contains(result, "Before ") {
		t.Error("Missing 'Before ' text")
	}
	if !strings.Contains(result, " After") {
		t.Error("Missing ' After' text")
	}
	if !strings.Contains(result, "I need to think about this") {
		t.Error("Missing thinking content")
	}
	if !strings.Contains(result, "[thought for") {
		t.Error("Missing thinking summary")
	}
	if !strings.Contains(result, "tokens]") {
		t.Error("Missing token count")
	}

	// Should contain ANSI codes for dimming
	if !strings.Contains(result, "\x1b[2m") {
		t.Error("Missing dim ANSI code")
	}
}

func TestThinkFilter_NoThinkTags(t *testing.T) {
	f := NewThinkFilter()

	input := "Regular content without think tags"
	output := f.Process(input)

	if output != input {
		t.Errorf("Content without think tags should pass through unchanged")
	}
}

func TestThinkFilter_MultipleThinkBlocks(t *testing.T) {
	f := NewThinkFilter()

	chunks := []string{
		"<think>First thought</think> Middle ",
		"<think>Second thought</think> End",
	}

	var output strings.Builder
	for _, chunk := range chunks {
		output.WriteString(f.Process(chunk))
	}

	result := output.String()

	// Should have both thinking contents
	if !strings.Contains(result, "First thought") {
		t.Error("Missing first thinking content")
	}
	if !strings.Contains(result, "Second thought") {
		t.Error("Missing second thinking content")
	}

	// Should have "Middle" and "End" text
	if !strings.Contains(result, "Middle ") {
		t.Error("Missing middle text")
	}
	if !strings.Contains(result, "End") {
		t.Error("Missing end text")
	}

	// Should have two summaries
	count := strings.Count(result, "[thought for")
	if count != 2 {
		t.Errorf("Expected 2 thinking summaries, got %d", count)
	}
}

func TestThinkFilter_PartialTagsAcrossChunks(t *testing.T) {
	f := NewThinkFilter()

	// Tags split across chunks
	chunks := []string{
		"<th",
		"ink>Content",
		"</th",
		"ink>",
	}

	var output strings.Builder
	for _, chunk := range chunks {
		output.WriteString(f.Process(chunk))
	}

	result := output.String()

	// Should handle split tags correctly
	if !strings.Contains(result, "Content") {
		t.Error("Missing thinking content with split tags")
	}
	if !strings.Contains(result, "[thought for") {
		t.Error("Missing thinking summary with split tags")
	}
}

func TestThinkFilter_Flush(t *testing.T) {
	f := NewThinkFilter()

	// Start a think block but don't close it - content streams immediately
	processed := f.Process("<think>Incomplete thought")

	// Content should already be output with dim formatting
	if !strings.Contains(processed, "Incomplete thought") {
		t.Error("Think content should stream in real-time")
	}
	if !strings.Contains(processed, "\x1b[2m") {
		t.Error("Think content should have dim formatting")
	}

	// Flush should output reset and incomplete summary
	flushed := f.Flush()

	if !strings.Contains(flushed, "incomplete") {
		t.Error("Flush should mark as incomplete")
	}
	if !strings.Contains(flushed, "[thought for") {
		t.Error("Flush should include timing summary")
	}
	if !strings.Contains(flushed, "\x1b[0m") {
		t.Error("Flush should reset formatting")
	}

	// After flush, filter should be reset
	if f.inThink {
		t.Error("Filter should not be in think state after flush")
	}
}

func TestThinkFilter_FlushEmpty(t *testing.T) {
	f := NewThinkFilter()

	// Flush with nothing buffered
	flushed := f.Flush()

	if flushed != "" {
		t.Error("Flush with empty buffer should return empty string")
	}
}

func TestThinkFilter_TokenCounting(t *testing.T) {
	f := NewThinkFilter()

	// Small delay to ensure measurable time
	time.Sleep(10 * time.Millisecond)

	chunks := []string{
		"<think>This is a test with several words</think>",
	}

	var output strings.Builder
	for _, chunk := range chunks {
		output.WriteString(f.Process(chunk))
	}

	result := output.String()

	// Should have rough token count (spaces)
	if !strings.Contains(result, "~") {
		t.Error("Missing approximate token count indicator")
	}
	if !strings.Contains(result, "tokens]") {
		t.Error("Missing token count")
	}

	// Time should be > 0
	if !strings.Contains(result, "0.0") && !strings.Contains(result, "0.1") {
		// Should show at least some time
		t.Log("Thinking summary:", result)
	}
}

func TestThinkFilter_Reset(t *testing.T) {
	f := NewThinkFilter()

	// Start a think block
	f.Process("<think>Some content")

	// Verify state is set
	if !f.inThink {
		t.Error("Should be in think state")
	}

	// Reset
	f.Reset()

	// Verify state is cleared
	if f.inThink {
		t.Error("Should not be in think state after reset")
	}
	if f.tagBuffer.Len() > 0 {
		t.Error("Tag buffer should be empty after reset")
	}
}

func TestThinkFilter_StripTags(t *testing.T) {
	f := NewThinkFilter()

	input := "Before<think>Hidden</think>After"
	output := f.Process(input)

	// Should not contain the literal tag strings in final output
	// (they should be stripped, content should be dimmed)
	if strings.Contains(output, "<think>") {
		t.Error("Should strip <think> tag")
	}
	if strings.Contains(output, "</think>") {
		t.Error("Should strip </think> tag")
	}

	// Should contain content
	if !strings.Contains(output, "Hidden") {
		t.Error("Should contain thinking content")
	}
	if !strings.Contains(output, "Before") {
		t.Error("Should contain before text")
	}
	if !strings.Contains(output, "After") {
		t.Error("Should contain after text")
	}
}
