package tui

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// Journey: specs/bugs/BUG-tui-blocks-and-thinking.md.

func TestMapper_ThinkingStartMarkerOnFirstDelta(t *testing.T) {
	t.Parallel()

	mapper := NewMapper(newFakeUI())

	stream := mapper.StartStreaming()
	defer mapper.Close()

	err := mapper.MapEvent(context.Background(), thinkingEvent("Let"))
	if err != nil {
		t.Fatalf("thinking delta: %v", err)
	}

	got := drainStream(stream)
	if !strings.Contains(got, thinkingStartMark) {
		t.Fatalf("expected thinking start marker %q in stream, got %q", thinkingStartMark, got)
	}
}

func TestMapper_ThinkingSummaryOnToolStart(t *testing.T) {
	t.Parallel()

	mapper := NewMapper(newFakeUI())

	stream := mapper.StartStreaming()
	defer mapper.Close()

	if err := mapper.MapEvent(context.Background(), thinkingEvent("plan")); err != nil {
		t.Fatalf("thinking delta: %v", err)
	}

	if err := mapper.MapEvent(context.Background(), writeFileStart("tool_think_1")); err != nil {
		t.Fatalf("tool start: %v", err)
	}

	got := drainStream(stream)
	if !strings.Contains(got, thinkingDonePrefix) {
		t.Fatalf("expected thinking summary before tool block, got %q", got)
	}
}

func TestMapper_ThinkingTimerResetsAfterTool(t *testing.T) {
	t.Parallel()

	mapper := NewMapper(newFakeUI())

	stream := mapper.StartStreaming()
	defer mapper.Close()

	if err := mapper.MapEvent(context.Background(), thinkingEvent("first")); err != nil {
		t.Fatalf("thinking delta: %v", err)
	}

	time.Sleep(thinkingPhaseGap)

	if err := mapper.MapEvent(context.Background(), writeFileStart("tool_think_2")); err != nil {
		t.Fatalf("tool start: %v", err)
	}

	if err := mapper.MapEvent(context.Background(), thinkingEvent("second")); err != nil {
		t.Fatalf("second thinking delta: %v", err)
	}

	if err := mapper.MapEvent(context.Background(), contentEvent("done")); err != nil {
		t.Fatalf("content delta: %v", err)
	}

	got := drainStream(stream)
	if strings.Count(got, thinkingDonePrefix) != 2 {
		t.Fatalf("expected two thinking summaries (one per phase), got %q", got)
	}
}

func TestMapper_ThinkingTokensFromShortChunks(t *testing.T) {
	t.Parallel()

	mapper := NewMapper(newFakeUI())

	stream := mapper.StartStreaming()
	defer mapper.Close()

	for range shortThinkingChunks {
		if err := mapper.MapEvent(context.Background(), thinkingEvent("x")); err != nil {
			t.Fatalf("thinking delta: %v", err)
		}
	}

	if err := mapper.MapEvent(context.Background(), contentEvent("ok")); err != nil {
		t.Fatalf("content delta: %v", err)
	}

	got := drainStream(stream)

	wantTokens := max(shortThinkingChunks/stringsx.CharsPerToken, 1)

	want := thinkingDonePrefix
	if !strings.Contains(got, want) {
		t.Fatalf("expected summary %q in %q", want, got)
	}

	if !strings.Contains(got, tokenCountNeedle(wantTokens)) {
		t.Fatalf("expected ~%d tokens in summary, got %q", wantTokens, got)
	}
}

const (
	thinkingPhaseGap    = 15 * time.Millisecond
	shortThinkingChunks = 8
	thinkingDonePrefix  = " [thought for "
)

func thinkingEvent(content string) events.Event {
	return events.Event{
		Type: events.EventThinkingDelta,
		Data: events.ThinkingDeltaData{Content: content},
	}
}

func contentEvent(content string) events.Event {
	return events.Event{
		Type: events.EventContentDelta,
		Data: events.ContentDeltaData{Content: content, Role: "assistant"},
	}
}

func writeFileStart(toolID string) events.Event {
	args, _ := tools.FromMap(map[string]any{
		"path":    "hello.txt",
		"content": "hello",
	})

	return events.Event{
		Type: events.EventToolCallStart,
		Data: events.ToolCallStartData{
			ToolName:   "write_file",
			ToolID:     toolID,
			Parameters: args,
		},
	}
}

func drainStream(ch <-chan string) string {
	var b strings.Builder

	for {
		select {
		case s, ok := <-ch:
			if !ok {
				return b.String()
			}

			b.WriteString(s)
		default:
			return b.String()
		}
	}
}

func tokenCountNeedle(n int) string {
	return " ~" + strconv.Itoa(n) + " tokens"
}

type statusRecordingUI struct {
	fakeUI

	eventTypes []events.EventType
}

func (s *statusRecordingUI) ProcessEvent(event *events.Event) {
	s.eventTypes = append(s.eventTypes, event.Type)
}

func TestMapper_TurnStartWithoutData_UpdatesStatus(t *testing.T) {
	t.Parallel()

	ui := &statusRecordingUI{fakeUI: *newFakeUI()}
	mapper := NewMapper(ui)

	err := mapper.MapEvent(context.Background(), events.Event{Type: events.EventTurnStart})
	if err != nil {
		t.Fatalf("MapEvent: %v", err)
	}

	if len(ui.eventTypes) != 1 || ui.eventTypes[0] != events.EventTurnStart {
		t.Fatalf("ProcessEvent types = %v, want [turn_start]", ui.eventTypes)
	}
}
