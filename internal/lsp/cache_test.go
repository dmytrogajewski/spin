package lsp_test

// Journey: specs/journeys/JOURNEY-R8.1.md.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

const (
	testCacheURI     = "file:///test.go"
	testCacheMethod  = "textDocument/definition"
	testHashA        = "abc123"
	testHashB        = "def456"
	testCacheRespStr = `{"uri":"file:///result.go"}`
)

func TestCache_L1Hit(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()
	resp := json.RawMessage(testCacheRespStr)

	cache.PutRaw(testCacheMethod, testCacheURI, testHashA, resp)

	got := cache.GetRaw(testCacheMethod, testCacheURI, testHashA)
	require.JSONEq(t, testCacheRespStr, string(got))
}

func TestCache_L1Miss(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()

	got := cache.GetRaw(testCacheMethod, testCacheURI, testHashA)
	require.Nil(t, got)
}

func TestCache_L1ContentHashInvalidation(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()
	resp := json.RawMessage(testCacheRespStr)

	cache.PutRaw(testCacheMethod, testCacheURI, testHashA, resp)

	// Different hash → cache miss.
	got := cache.GetRaw(testCacheMethod, testCacheURI, testHashB)
	require.Nil(t, got, "different content hash should miss")
}

func TestCache_L2Hit(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()
	symbols := []lsp.Symbol{
		{Name: "main", Kind: lsp.SymbolFunction},
		{Name: "Handler", Kind: lsp.SymbolType},
	}

	cache.PutSymbols(testCacheURI, testHashA, symbols)

	got := cache.GetSymbols(testCacheURI, testHashA)
	require.Len(t, got, 2)
	require.Equal(t, "main", got[0].Name)
	require.Equal(t, "Handler", got[1].Name)
}

func TestCache_L2Miss(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()

	got := cache.GetSymbols(testCacheURI, testHashA)
	require.Nil(t, got)
}

func TestCache_L2ContentHashInvalidation(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()
	symbols := []lsp.Symbol{
		{Name: "old", Kind: lsp.SymbolVariable},
	}

	cache.PutSymbols(testCacheURI, testHashA, symbols)

	got := cache.GetSymbols(testCacheURI, testHashB)
	require.Nil(t, got, "different content hash should miss")
}

func TestCache_Invalidate(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()

	cache.PutRaw(testCacheMethod, testCacheURI, testHashA, json.RawMessage(`{}`))
	cache.PutSymbols(testCacheURI, testHashA, []lsp.Symbol{{Name: "x"}})

	require.Equal(t, 2, cache.Size())

	cache.Invalidate(testCacheURI)

	require.Equal(t, 0, cache.Size())
	require.Nil(t, cache.GetRaw(testCacheMethod, testCacheURI, testHashA))
	require.Nil(t, cache.GetSymbols(testCacheURI, testHashA))
}

func TestCache_Size(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()

	require.Equal(t, 0, cache.Size())

	cache.PutRaw(testCacheMethod, testCacheURI, testHashA, json.RawMessage(`{}`))

	require.Equal(t, 1, cache.Size())

	cache.PutSymbols(testCacheURI, testHashA, []lsp.Symbol{})

	require.Equal(t, 2, cache.Size())
}

func TestContentHash(t *testing.T) {
	t.Parallel()

	hash1 := lsp.ContentHash([]byte("hello world"))
	hash2 := lsp.ContentHash([]byte("hello world"))
	hash3 := lsp.ContentHash([]byte("different"))

	require.Equal(t, hash1, hash2, "same content should produce same hash")
	require.NotEqual(t, hash1, hash3, "different content should produce different hash")
	require.Len(t, hash1, 32, "MD5 hex string should be 32 chars")
}

func TestCache_OverwriteEntry(t *testing.T) {
	t.Parallel()

	cache := lsp.NewCache()

	cache.PutRaw(testCacheMethod, testCacheURI, testHashA, json.RawMessage(`{"old":true}`))
	cache.PutRaw(testCacheMethod, testCacheURI, testHashB, json.RawMessage(`{"new":true}`))

	// Old hash should miss.
	require.Nil(t, cache.GetRaw(testCacheMethod, testCacheURI, testHashA))

	// New hash should hit.
	got := cache.GetRaw(testCacheMethod, testCacheURI, testHashB)
	require.JSONEq(t, `{"new":true}`, string(got))
}
