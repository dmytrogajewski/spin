package lsp_test

// Journey: specs/journeys/JOURNEY-R8.1.md.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

const (
	testURI         = "file:///test.go"
	testRootURI     = "file:///workspace"
	testNewName     = "HandleRequest"
	testLangID      = "go"
	testFileContent = "package main"
	testLine        = 10
	testChar        = 5
	testVersion     = 2
)

// mockTransport implements lsp.Transport for testing server methods.
type mockTransport struct {
	sendFunc   func(ctx context.Context, method string, params any) (json.RawMessage, error)
	notifyFunc func(ctx context.Context, method string, params any) error
	closeFunc  func() error
}

func (mt *mockTransport) Send(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if mt.sendFunc != nil {
		return mt.sendFunc(ctx, method, params)
	}

	return json.RawMessage(`{}`), nil
}

func (mt *mockTransport) Notify(ctx context.Context, method string, params any) error {
	if mt.notifyFunc != nil {
		return mt.notifyFunc(ctx, method, params)
	}

	return nil
}

func (mt *mockTransport) Close() error {
	if mt.closeFunc != nil {
		return mt.closeFunc()
	}

	return nil
}

func newTestServer(transport lsp.Transport) *lsp.Server {
	lang := lsp.LanguageConfig{
		ID:            testLangID,
		Extensions:    []string{".go"},
		ServerCommand: "gopls",
	}

	return lsp.NewServer(transport, lang)
}

func TestServer_InitializeHandshake(t *testing.T) {
	t.Parallel()

	var methods []string

	transport := &mockTransport{
		sendFunc: func(_ context.Context, method string, _ any) (json.RawMessage, error) {
			methods = append(methods, method)

			return json.RawMessage(`{"capabilities":{}}`), nil
		},
		notifyFunc: func(_ context.Context, method string, _ any) error {
			methods = append(methods, method)

			return nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))
	require.Equal(t, []string{"initialize", "initialized"}, methods)
}

func TestServer_InitializeIdempotent(t *testing.T) {
	t.Parallel()

	callCount := 0
	transport := &mockTransport{
		sendFunc: func(_ context.Context, _ string, _ any) (json.RawMessage, error) {
			callCount++

			return json.RawMessage(`{}`), nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))
	require.NoError(t, srv.Initialize(ctx, testRootURI))
	require.Equal(t, 1, callCount, "initialize should only be called once")
}

func TestServer_FindDefinition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		response    string
		expectedURI string
	}{
		{
			name:        "array_response",
			response:    `[{"uri":"file:///def.go","range":{"start":{"line":5,"character":0},"end":{"line":5,"character":10}}}]`,
			expectedURI: "file:///def.go",
		},
		{
			name:        "single_object_response",
			response:    `{"uri":"file:///single.go","range":{"start":{"line":1,"character":0},"end":{"line":1,"character":5}}}`,
			expectedURI: "file:///single.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			respJSON := tt.response

			transport := &mockTransport{
				sendFunc: func(_ context.Context, method string, _ any) (json.RawMessage, error) {
					if method == "textDocument/definition" {
						return json.RawMessage(respJSON), nil
					}

					return json.RawMessage(`{}`), nil
				},
			}

			srv := newTestServer(transport)
			ctx := context.Background()

			require.NoError(t, srv.Initialize(ctx, testRootURI))

			locations, findErr := srv.FindDefinition(ctx, testURI, testLine, testChar)
			require.NoError(t, findErr)
			require.Len(t, locations, 1)
			require.Equal(t, tt.expectedURI, locations[0].URI)
		})
	}
}

func TestServer_FindReferences(t *testing.T) {
	t.Parallel()

	refsJSON := `[{"uri":"file:///a.go","range":{"start":{"line":1,"character":0},"end":{"line":1,"character":5}}},` +
		`{"uri":"file:///b.go","range":{"start":{"line":2,"character":0},"end":{"line":2,"character":5}}}]`

	transport := &mockTransport{
		sendFunc: func(_ context.Context, method string, _ any) (json.RawMessage, error) {
			if method == "textDocument/references" {
				return json.RawMessage(refsJSON), nil
			}

			return json.RawMessage(`{}`), nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))

	refs, findErr := srv.FindReferences(ctx, testURI, testLine, testChar)
	require.NoError(t, findErr)
	require.Len(t, refs, 2)
}

func TestServer_Rename(t *testing.T) {
	t.Parallel()

	editJSON := `{"changes":{"file:///test.go":[{"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":5}},"new_text":"NewName"}]}}`

	transport := &mockTransport{
		sendFunc: func(_ context.Context, method string, _ any) (json.RawMessage, error) {
			if method == "textDocument/rename" {
				return json.RawMessage(editJSON), nil
			}

			return json.RawMessage(`{}`), nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))

	edit, renameErr := srv.Rename(ctx, testURI, testLine, testChar, testNewName)
	require.NoError(t, renameErr)
	require.NotNil(t, edit)
	require.Contains(t, edit.Changes, testURI)
}

func TestServer_DidOpen(t *testing.T) {
	t.Parallel()

	var notifiedMethod string

	transport := &mockTransport{
		notifyFunc: func(_ context.Context, method string, _ any) error {
			notifiedMethod = method

			return nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))

	openErr := srv.DidOpen(ctx, testURI, testLangID, testFileContent)
	require.NoError(t, openErr)
	require.Equal(t, "textDocument/didOpen", notifiedMethod)
}

func TestServer_DidChange(t *testing.T) {
	t.Parallel()

	var notifiedMethod string

	transport := &mockTransport{
		notifyFunc: func(_ context.Context, method string, _ any) error {
			notifiedMethod = method

			return nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))

	changeErr := srv.DidChange(ctx, testURI, testVersion, testFileContent)
	require.NoError(t, changeErr)
	require.Equal(t, "textDocument/didChange", notifiedMethod)
}

func TestServer_NotInitialized(t *testing.T) {
	t.Parallel()

	transport := &mockTransport{}
	srv := newTestServer(transport)
	ctx := context.Background()

	_, defErr := srv.FindDefinition(ctx, testURI, testLine, testChar)
	require.ErrorIs(t, defErr, lsp.ErrServerNotInitialized)

	_, refErr := srv.FindReferences(ctx, testURI, testLine, testChar)
	require.ErrorIs(t, refErr, lsp.ErrServerNotInitialized)

	_, renameErr := srv.Rename(ctx, testURI, testLine, testChar, testNewName)
	require.ErrorIs(t, renameErr, lsp.ErrServerNotInitialized)

	openErr := srv.DidOpen(ctx, testURI, testLangID, testFileContent)
	require.ErrorIs(t, openErr, lsp.ErrServerNotInitialized)

	changeErr := srv.DidChange(ctx, testURI, testVersion, testFileContent)
	require.ErrorIs(t, changeErr, lsp.ErrServerNotInitialized)
}

func TestServer_DeadServer(t *testing.T) {
	t.Parallel()

	transport := &mockTransport{}
	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))

	srv.SetAlive(false)

	_, defErr := srv.FindDefinition(ctx, testURI, testLine, testChar)
	require.ErrorIs(t, defErr, lsp.ErrServerDead)
}

func TestServer_Shutdown(t *testing.T) {
	t.Parallel()

	var methods []string

	transport := &mockTransport{
		sendFunc: func(_ context.Context, method string, _ any) (json.RawMessage, error) {
			methods = append(methods, method)

			return json.RawMessage(`null`), nil
		},
		notifyFunc: func(_ context.Context, method string, _ any) error {
			methods = append(methods, method)

			return nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))

	methods = nil

	require.NoError(t, srv.Shutdown(ctx))
	require.Equal(t, []string{"shutdown", "exit"}, methods)
}

func TestServer_ShutdownNotInitialized(t *testing.T) {
	t.Parallel()

	transport := &mockTransport{}
	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Shutdown(ctx), "shutdown of uninitialized server should be no-op")
}

func TestServer_IsAlive(t *testing.T) {
	t.Parallel()

	transport := &mockTransport{}
	srv := newTestServer(transport)

	require.True(t, srv.IsAlive())

	srv.SetAlive(false)

	require.False(t, srv.IsAlive())
}

func TestServer_Language(t *testing.T) {
	t.Parallel()

	transport := &mockTransport{}
	srv := newTestServer(transport)

	require.Equal(t, testLangID, srv.Language().ID)
}

func TestServer_Close(t *testing.T) {
	t.Parallel()

	closeCalled := false
	transport := &mockTransport{
		closeFunc: func() error {
			closeCalled = true

			return nil
		},
	}

	srv := newTestServer(transport)
	ctx := context.Background()

	require.NoError(t, srv.Initialize(ctx, testRootURI))

	closeErr := srv.Close(ctx)
	require.NoError(t, closeErr)
	require.True(t, closeCalled)
}
