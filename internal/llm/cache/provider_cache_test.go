package cache_test

// Journey: specs/journeys/JOURNEY-R7.1.md.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/llm/cache"
)

const (
	testProvider    = "openai"
	testModel       = "gpt-4o"
	testModelAlt    = "gpt-4o-mini"
	testContextLen  = 200_000
	testContextAlt  = 128_000
	testCacheSubdir = "cache-test"
)

var errFetchFailed = errors.New("network unavailable")

type mockFetcher struct {
	mu          sync.Mutex
	caps        cache.ModelCapabilities
	err         error
	fetchCount  int
	fetchSignal chan struct{}
}

func (m *mockFetcher) FetchCapabilities(_ context.Context, _, _ string) (cache.ModelCapabilities, error) {
	m.mu.Lock()
	m.fetchCount++
	m.mu.Unlock()

	if m.fetchSignal != nil {
		m.fetchSignal <- struct{}{}
	}

	return m.caps, m.err
}

func (m *mockFetcher) getFetchCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.fetchCount
}

func staleTime() time.Time {
	return time.Now().Add(-25 * time.Hour)
}

func TestProviderCache_FreshCacheReturnsImmediately(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)
	fetcher := &mockFetcher{}

	pc, err := cache.NewProviderCache(dir, fetcher)
	require.NoError(t, err)

	caps := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextLen,
		Vision:        true,
	}

	require.NoError(t, pc.Put(testProvider, testModel, caps))

	ctx := context.Background()
	got := pc.Get(ctx, testProvider, testModel)
	require.Equal(t, testContextLen, got.ContextLength)
	require.True(t, got.Vision)
	require.Equal(t, 0, fetcher.getFetchCount(), "fresh cache should not trigger fetch")
}

func TestProviderCache_StaleCacheReturnsAndRefreshes(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)

	fetchSignal := make(chan struct{}, 1)
	fetcher := &mockFetcher{
		caps: cache.ModelCapabilities{
			ModelID:       testModel,
			Provider:      testProvider,
			ContextLength: testContextLen,
			Vision:        true,
		},
		fetchSignal: fetchSignal,
	}

	// Use a switchable time: start at stale time for Put, then switch to now for Get.
	var timeMu sync.Mutex

	currentTime := staleTime()

	timeFunc := func() time.Time {
		timeMu.Lock()
		defer timeMu.Unlock()

		return currentTime
	}

	pc, err := cache.NewProviderCache(dir, fetcher, cache.WithTimeFunc(timeFunc))
	require.NoError(t, err)

	// Pre-populate with stale entry (FetchedAt = 25h ago).
	staleCaps := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextAlt,
	}
	require.NoError(t, pc.Put(testProvider, testModel, staleCaps))

	// Switch to current time so isFresh sees it as stale.
	timeMu.Lock()
	currentTime = time.Now()
	timeMu.Unlock()

	// Get should return stale data immediately.
	ctx := context.Background()
	got := pc.Get(ctx, testProvider, testModel)
	require.Equal(t, testContextAlt, got.ContextLength, "should return stale data")

	// Wait for background refresh.
	select {
	case <-fetchSignal:
		// Background refresh happened.
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for background refresh")
	}

	require.Equal(t, 1, fetcher.getFetchCount(), "should have triggered background fetch")
}

func TestProviderCache_MissingCacheFetchesSynchronously(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)
	fetcher := &mockFetcher{
		caps: cache.ModelCapabilities{
			ModelID:       testModel,
			Provider:      testProvider,
			ContextLength: testContextLen,
		},
	}

	pc, err := cache.NewProviderCache(dir, fetcher)
	require.NoError(t, err)

	ctx := context.Background()
	got := pc.Get(ctx, testProvider, testModel)
	require.Equal(t, testContextLen, got.ContextLength)
	require.Equal(t, 1, fetcher.getFetchCount(), "missing cache should trigger sync fetch")
}

func TestProviderCache_NetworkFailureReturnsDefaults(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)
	fetcher := &mockFetcher{
		err: errFetchFailed,
	}

	pc, err := cache.NewProviderCache(dir, fetcher)
	require.NoError(t, err)

	ctx := context.Background()
	got := pc.Get(ctx, testProvider, testModel)
	require.Equal(t, cache.DefaultContextWindow, got.ContextLength, "should return safe default")
	require.True(t, got.Streaming, "default should have streaming")
	require.True(t, got.FunctionCalling, "default should have function calling")
}

func TestProviderCache_NilFetcherReturnsDefaults(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)

	pc, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	ctx := context.Background()
	got := pc.Get(ctx, testProvider, testModel)
	require.Equal(t, cache.DefaultContextWindow, got.ContextLength)
}

func TestProviderCache_ContextLengthLookup(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)

	pc, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	caps := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextLen,
	}
	require.NoError(t, pc.Put(testProvider, testModel, caps))

	ctx := context.Background()
	require.Equal(t, testContextLen, pc.ContextLength(ctx, testProvider, testModel))
}

func TestProviderCache_ContextLengthDefault(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)

	pc, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	ctx := context.Background()
	require.Equal(t, cache.DefaultContextWindow, pc.ContextLength(ctx, testProvider, "unknown-model"))
}

func TestProviderCache_PersistAndReload(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)

	pc1, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	caps := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextLen,
		Vision:        true,
		Thinking:      true,
	}
	require.NoError(t, pc1.Put(testProvider, testModel, caps))

	// Create new cache pointing to same dir.
	pc2, reloadErr := cache.NewProviderCache(dir, nil)
	require.NoError(t, reloadErr)

	ctx := context.Background()
	got := pc2.Get(ctx, testProvider, testModel)
	require.Equal(t, testContextLen, got.ContextLength)
	require.True(t, got.Vision)
	require.True(t, got.Thinking)
}

func TestProviderCache_AtomicWrite(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)

	pc, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	caps := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextLen,
	}
	require.NoError(t, pc.Put(testProvider, testModel, caps))

	// Verify file is valid JSON.
	path := filepath.Join(dir, testProvider+".json")
	data, readErr := os.ReadFile(path)
	require.NoError(t, readErr)

	var pd cache.ProviderData
	require.NoError(t, json.Unmarshal(data, &pd))
	require.Len(t, pd.Models, 1)
	require.Equal(t, testContextLen, pd.Models[testModel].Capabilities.ContextLength)
}

func TestProviderCache_MultipleModels(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), testCacheSubdir)

	pc, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	caps1 := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextLen,
	}
	caps2 := cache.ModelCapabilities{
		ModelID:       testModelAlt,
		Provider:      testProvider,
		ContextLength: testContextAlt,
	}

	require.NoError(t, pc.Put(testProvider, testModel, caps1))
	require.NoError(t, pc.Put(testProvider, testModelAlt, caps2))

	ctx := context.Background()
	require.Equal(t, testContextLen, pc.Get(ctx, testProvider, testModel).ContextLength)
	require.Equal(t, testContextAlt, pc.Get(ctx, testProvider, testModelAlt).ContextLength)
}

func TestEntry_IsFresh(t *testing.T) {
	t.Parallel()

	fresh := cache.Entry{FetchedAt: time.Now()}
	require.True(t, fresh.IsFresh())

	stale := cache.Entry{FetchedAt: time.Now().Add(-25 * time.Hour)}
	require.False(t, stale.IsFresh())
}
