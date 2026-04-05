package lsp_test

// Journey: specs/journeys/JOURNEY-R8.1.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

func TestDetectLanguage_Go(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("main.go")
	require.NoError(t, err)
	require.Equal(t, "go", lang.ID)
	require.Equal(t, "gopls", lang.ServerCommand)
	require.Contains(t, lang.Extensions, ".go")
	require.Contains(t, lang.RootMarkers, "go.mod")
}

func TestDetectLanguage_Python(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("script.py")
	require.NoError(t, err)
	require.Equal(t, "python", lang.ID)
	require.Equal(t, "pylsp", lang.ServerCommand)
}

func TestDetectLanguage_PythonStub(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("types.pyi")
	require.NoError(t, err)
	require.Equal(t, "python", lang.ID)
}

func TestDetectLanguage_TypeScript(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("app.ts")
	require.NoError(t, err)
	require.Equal(t, "typescript", lang.ID)
	require.Equal(t, "typescript-language-server", lang.ServerCommand)
}

func TestDetectLanguage_TSX(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("component.tsx")
	require.NoError(t, err)
	require.Equal(t, "typescript", lang.ID)
}

func TestDetectLanguage_JavaScript(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("index.js")
	require.NoError(t, err)
	require.Equal(t, "javascript", lang.ID)
}

func TestDetectLanguage_Rust(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("lib.rs")
	require.NoError(t, err)
	require.Equal(t, "rust", lang.ID)
	require.Equal(t, "rust-analyzer", lang.ServerCommand)
}

func TestDetectLanguage_Java(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("App.java")
	require.NoError(t, err)
	require.Equal(t, "java", lang.ID)
}

func TestDetectLanguage_C(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("main.c")
	require.NoError(t, err)
	require.Equal(t, "c", lang.ID)
	require.Equal(t, "clangd", lang.ServerCommand)
}

func TestDetectLanguage_CHeader(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("header.h")
	require.NoError(t, err)
	require.Equal(t, "c", lang.ID)
}

func TestDetectLanguage_Cpp(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("main.cpp")
	require.NoError(t, err)
	require.Equal(t, "cpp", lang.ID)
}

func TestDetectLanguage_Ruby(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("app.rb")
	require.NoError(t, err)
	require.Equal(t, "ruby", lang.ID)
	require.Equal(t, "solargraph", lang.ServerCommand)
}

func TestDetectLanguage_Unknown(t *testing.T) {
	t.Parallel()

	_, err := lsp.DetectLanguage("data.xyz")
	require.ErrorIs(t, err, lsp.ErrUnsupportedLanguage)
}

func TestDetectLanguage_NoExtension(t *testing.T) {
	t.Parallel()

	_, err := lsp.DetectLanguage("Makefile")
	require.ErrorIs(t, err, lsp.ErrUnsupportedLanguage)
}

func TestDetectLanguage_CaseInsensitive(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("MAIN.GO")
	require.NoError(t, err)
	require.Equal(t, "go", lang.ID)
}

func TestDetectLanguage_FullPath(t *testing.T) {
	t.Parallel()

	lang, err := lsp.DetectLanguage("/home/user/project/src/main.go")
	require.NoError(t, err)
	require.Equal(t, "go", lang.ID)
}
