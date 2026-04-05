package lsp_test

// Journey: specs/journeys/JOURNEY-R8.1.md.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

var errFactoryFailed = errors.New("factory failed")

func mockFactory(_ context.Context, lang lsp.LanguageConfig, _ string) (*lsp.Server, error) {
	transport := &mockTransport{
		sendFunc: func(_ context.Context, _ string, _ any) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		},
	}

	srv := lsp.NewServer(transport, lang)

	return srv, nil
}

func failingFactory(_ context.Context, _ lsp.LanguageConfig, _ string) (*lsp.Server, error) {
	return nil, errFactoryFailed
}

func TestManager_LazyStartsServer(t *testing.T) {
	t.Parallel()

	mgr := lsp.NewManager(testRootURI, mockFactory)
	ctx := context.Background()

	srv, forFileErr := mgr.ForFile(ctx, "main.go")
	require.NoError(t, forFileErr)
	require.NotNil(t, srv)
	require.Equal(t, "go", srv.Language().ID)
}

func TestManager_ReusesRunningServer(t *testing.T) {
	t.Parallel()

	createCount := 0

	countingFactory := func(ctx context.Context, lang lsp.LanguageConfig, rootURI string) (*lsp.Server, error) {
		createCount++

		return mockFactory(ctx, lang, rootURI)
	}

	mgr := lsp.NewManager(testRootURI, countingFactory)
	ctx := context.Background()

	srv1, err1 := mgr.ForFile(ctx, "main.go")
	require.NoError(t, err1)

	srv2, err2 := mgr.ForFile(ctx, "other.go")
	require.NoError(t, err2)

	require.Same(t, srv1, srv2, "should reuse same server for same language")
	require.Equal(t, 1, createCount)
}

func TestManager_RestartsDeadServer(t *testing.T) {
	t.Parallel()

	createCount := 0

	countingFactory := func(ctx context.Context, lang lsp.LanguageConfig, rootURI string) (*lsp.Server, error) {
		createCount++

		return mockFactory(ctx, lang, rootURI)
	}

	mgr := lsp.NewManager(testRootURI, countingFactory)
	ctx := context.Background()

	srv1, err1 := mgr.ForFile(ctx, "main.go")
	require.NoError(t, err1)

	// Simulate server crash.
	srv1.SetAlive(false)

	srv2, err2 := mgr.ForFile(ctx, "other.go")
	require.NoError(t, err2)
	require.NotSame(t, srv1, srv2, "should create new server after crash")
	require.Equal(t, 2, createCount)
}

func TestManager_DifferentLanguages(t *testing.T) {
	t.Parallel()

	mgr := lsp.NewManager(testRootURI, mockFactory)
	ctx := context.Background()

	goSrv, err1 := mgr.ForFile(ctx, "main.go")
	require.NoError(t, err1)

	pySrv, err2 := mgr.ForFile(ctx, "script.py")
	require.NoError(t, err2)

	require.NotSame(t, goSrv, pySrv)
	require.Equal(t, "go", goSrv.Language().ID)
	require.Equal(t, "python", pySrv.Language().ID)
}

func TestManager_UnsupportedLanguage(t *testing.T) {
	t.Parallel()

	mgr := lsp.NewManager(testRootURI, mockFactory)
	ctx := context.Background()

	_, forFileErr := mgr.ForFile(ctx, "data.xyz")
	require.ErrorIs(t, forFileErr, lsp.ErrUnsupportedLanguage)
}

func TestManager_FactoryError(t *testing.T) {
	t.Parallel()

	mgr := lsp.NewManager(testRootURI, failingFactory)
	ctx := context.Background()

	_, forFileErr := mgr.ForFile(ctx, "main.go")
	require.ErrorIs(t, forFileErr, errFactoryFailed)
}

func TestManager_Close(t *testing.T) {
	t.Parallel()

	mgr := lsp.NewManager(testRootURI, mockFactory)
	ctx := context.Background()

	_, err1 := mgr.ForFile(ctx, "main.go")
	require.NoError(t, err1)

	_, err2 := mgr.ForFile(ctx, "script.py")
	require.NoError(t, err2)

	closeErr := mgr.Close(ctx)
	require.NoError(t, closeErr)
}

func TestManager_CloseEmpty(t *testing.T) {
	t.Parallel()

	mgr := lsp.NewManager(testRootURI, mockFactory)
	ctx := context.Background()

	require.NoError(t, mgr.Close(ctx))
}
