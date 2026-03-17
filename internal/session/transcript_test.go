package session_test

// Journey: specs/journeys/JOURNEY-R6.1.md.

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/session"
)

const (
	testMsgContent    = "hello world"
	testMsgUpdated    = "hello updated"
	testMsgRole       = message.RoleUser
	testMsgAssistant  = "assistant reply"
	testCorruptedLine = "{invalid json\n"
	testValidLine     = `{"id":"msg-1","role":"user","content":"valid","timestamp":"2024-01-01T00:00:00Z","tokens":0}` + "\n"
)

func newTestMessage(content string) message.Message {
	return message.Message{
		ID:        "msg-1",
		Role:      testMsgRole,
		Content:   content,
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func transcriptPath(dir string) string {
	return filepath.Join(dir, "test.transcript.jsonl")
}

func TestTranscriptWriter_AppendAndReadAll(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer.Close()

	msg := newTestMessage(testMsgContent)

	require.NoError(t, writer.Append(msg))

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, 1)
	require.Equal(t, testMsgContent, msgs[0].Content)
	require.Equal(t, testMsgRole, msgs[0].Role)
}

func TestTranscriptWriter_MultipleAppends(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer.Close()

	msg1 := newTestMessage(testMsgContent)
	msg2 := message.Message{
		ID:        "msg-2",
		Role:      message.RoleAssistant,
		Content:   testMsgAssistant,
		Timestamp: time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC),
	}

	require.NoError(t, writer.Append(msg1))
	require.NoError(t, writer.Append(msg2))

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, 2)
	require.Equal(t, testMsgContent, msgs[0].Content)
	require.Equal(t, testMsgAssistant, msgs[1].Content)
}

func TestTranscriptWriter_CorruptedLineSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	// Write a corrupted line followed by a valid line directly to file.
	err := os.WriteFile(path, []byte(testCorruptedLine+testValidLine), 0o600)
	require.NoError(t, err)

	writer, openErr := session.NewTranscriptWriter(path)
	require.NoError(t, openErr)

	defer writer.Close()

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, 1)
	require.Equal(t, "valid", msgs[0].Content)
}

func TestTranscriptWriter_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer.Close()

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Empty(t, msgs)
}

func TestTranscriptWriter_ConcurrentAppend(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer.Close()

	const goroutineCount = 10

	var waitGroup sync.WaitGroup

	errs := make([]error, goroutineCount)

	waitGroup.Add(goroutineCount)

	for idx := range goroutineCount {
		go func(index int) {
			defer waitGroup.Done()

			msg := message.Message{
				ID:        "msg-concurrent",
				Role:      testMsgRole,
				Content:   testMsgContent,
				Timestamp: time.Date(2024, 1, 1, 0, 0, index, 0, time.UTC),
			}

			errs[index] = writer.Append(msg)
		}(idx)
	}

	waitGroup.Wait()

	for idx, appendErr := range errs {
		require.NoError(t, appendErr, "goroutine %d failed", idx)
	}

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, goroutineCount)
}

func TestTranscriptWriter_CloseIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	require.NoError(t, writer.Close())
}

func TestTranscriptWriter_AppendAfterClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	appendErr := writer.Append(newTestMessage(testMsgContent))
	require.Error(t, appendErr)
}

func TestTranscriptWriter_ReadAllAfterClose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	require.NoError(t, writer.Append(newTestMessage(testMsgContent)))
	require.NoError(t, writer.Close())

	// ReadAll should still work after close — it opens a new file handle.
	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, 1)
}

func TestTranscriptWriter_PreservesToolCalls(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer.Close()

	msg := message.Message{
		ID:        "msg-tools",
		Role:      message.RoleAssistant,
		Content:   testMsgAssistant,
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		ToolCalls: []message.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "/tmp/test.go"}`,
				},
			},
		},
	}

	require.NoError(t, writer.Append(msg))

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].ToolCalls, 1)
	require.Equal(t, "read_file", msgs[0].ToolCalls[0].Function.Name)
}

func TestTranscriptWriter_PreservesMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer.Close()

	msg := message.Message{
		ID:        "msg-meta",
		Role:      testMsgRole,
		Content:   testMsgContent,
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Tokens:    42,
		Metadata:  message.Metadata{"key": "value"},
	}

	require.NoError(t, writer.Append(msg))

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, 1)

	const expectedTokens = 42

	require.Equal(t, expectedTokens, msgs[0].Tokens)
	require.Equal(t, "value", msgs[0].Metadata["key"])
}

func TestTranscriptWriter_Count(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	writer, err := session.NewTranscriptWriter(path)
	require.NoError(t, err)

	defer writer.Close()

	require.Equal(t, 0, writer.Count())

	require.NoError(t, writer.Append(newTestMessage(testMsgContent)))
	require.Equal(t, 1, writer.Count())

	require.NoError(t, writer.Append(newTestMessage(testMsgUpdated)))
	require.Equal(t, 2, writer.Count())
}

func TestTranscriptWriter_EmptyLinesSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := transcriptPath(dir)

	// Write empty lines interspersed with valid line.
	content := "\n\n" + testValidLine + "\n"

	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)

	writer, openErr := session.NewTranscriptWriter(path)
	require.NoError(t, openErr)

	defer writer.Close()

	msgs, readErr := writer.ReadAll()
	require.NoError(t, readErr)
	require.Len(t, msgs, 1)
}
