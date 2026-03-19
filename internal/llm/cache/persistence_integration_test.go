package cache_test

// Journey: specs/journeys/JOURNEY-3.3.md.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/llm/cache"
)

func TestProviderCache_PersistsAcrossInstances(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// First startup: cache miss, fetch capabilities, persist.
	fetcher := &mockFetcher{
		caps: cache.ModelCapabilities{
			ModelID:         testModel,
			Provider:        testProvider,
			ContextLength:   testContextLen,
			Vision:          true,
			Thinking:        true,
			Streaming:       true,
			FunctionCalling: true,
		},
	}

	pc1, err := cache.NewProviderCache(dir, fetcher)
	require.NoError(t, err)

	ctx := context.Background()
	got := pc1.Get(ctx, testProvider, testModel)
	require.Equal(t, testContextLen, got.ContextLength, "first startup should fetch and return capabilities")
	require.Equal(t, 1, fetcher.getFetchCount(), "first startup should trigger exactly one fetch")

	// Second startup: cache hit, no fetch needed.
	fetcher2 := &mockFetcher{
		caps: cache.ModelCapabilities{
			ModelID:       testModel,
			Provider:      testProvider,
			ContextLength: 999_999, // Different value to prove it was NOT fetched.
		},
	}

	pc2, err := cache.NewProviderCache(dir, fetcher2)
	require.NoError(t, err)

	got2 := pc2.Get(ctx, testProvider, testModel)
	require.Equal(t, testContextLen, got2.ContextLength, "second startup should use persisted cache")
	require.True(t, got2.Vision, "vision capability should survive persistence")
	require.True(t, got2.Thinking, "thinking capability should survive persistence")
	require.True(t, got2.Streaming, "streaming capability should survive persistence")
	require.True(t, got2.FunctionCalling, "function_calling capability should survive persistence")
	require.Equal(t, 0, fetcher2.getFetchCount(), "second startup should not fetch when cache is fresh")
}

func TestProviderCache_StaleEntry_TriggersRefresh(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// First startup: populate with stale entry.
	var timeMu sync.Mutex

	currentTime := staleTime()

	timeFunc := func() time.Time {
		timeMu.Lock()
		defer timeMu.Unlock()

		return currentTime
	}

	pc1, err := cache.NewProviderCache(dir, nil, cache.WithTimeFunc(timeFunc))
	require.NoError(t, err)

	staleCaps := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextAlt,
	}
	require.NoError(t, pc1.Put(context.Background(), testProvider, testModel, staleCaps))

	// Second startup: stale entry should trigger background refresh.
	fetchSignal := make(chan struct{}, 1)
	refreshedCaps := cache.ModelCapabilities{
		ModelID:       testModel,
		Provider:      testProvider,
		ContextLength: testContextLen,
		Vision:        true,
	}
	fetcher := &mockFetcher{
		caps:        refreshedCaps,
		fetchSignal: fetchSignal,
	}

	// New instance reads from disk; uses real time so the entry appears stale.
	pc2, err := cache.NewProviderCache(dir, fetcher)
	require.NoError(t, err)

	ctx := context.Background()
	got := pc2.Get(ctx, testProvider, testModel)

	// Should return stale data immediately.
	require.Equal(t, testContextAlt, got.ContextLength, "should return stale data immediately")

	// Wait for background refresh to complete.
	select {
	case <-fetchSignal:
		// Background refresh triggered.
	case <-time.After(testFetchWaitTimeout):
		t.Fatal("timeout waiting for background refresh on stale entry")
	}

	require.Equal(t, 1, fetcher.getFetchCount(), "stale entry should trigger exactly one background fetch")
}

func TestProviderCache_DifferentProviders_Isolated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	pc, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	providerA := "anthropic"
	providerB := "openai"
	modelA := "claude-4"
	modelB := "gpt-4o"

	capsA := cache.ModelCapabilities{
		ModelID:       modelA,
		Provider:      providerA,
		ContextLength: 200_000,
		Vision:        true,
		Thinking:      true,
	}
	capsB := cache.ModelCapabilities{
		ModelID:       modelB,
		Provider:      providerB,
		ContextLength: 128_000,
		Vision:        true,
		Thinking:      false,
	}

	ctx := context.Background()
	require.NoError(t, pc.Put(ctx, providerA, modelA, capsA))
	require.NoError(t, pc.Put(ctx, providerB, modelB, capsB))

	// Verify isolation within the same instance.
	gotA := pc.Get(ctx, providerA, modelA)
	gotB := pc.Get(ctx, providerB, modelB)

	require.Equal(t, 200_000, gotA.ContextLength)
	require.True(t, gotA.Thinking)
	require.Equal(t, 128_000, gotB.ContextLength)
	require.False(t, gotB.Thinking)

	// Verify isolation persists across a new instance.
	pc2, err := cache.NewProviderCache(dir, nil)
	require.NoError(t, err)

	gotA2 := pc2.Get(ctx, providerA, modelA)
	gotB2 := pc2.Get(ctx, providerB, modelB)

	require.Equal(t, 200_000, gotA2.ContextLength, "provider A caps should persist independently")
	require.True(t, gotA2.Thinking, "provider A thinking should persist")
	require.Equal(t, 128_000, gotB2.ContextLength, "provider B caps should persist independently")
	require.False(t, gotB2.Thinking, "provider B thinking=false should persist")

	// Cross-provider lookup should miss (return defaults).
	gotCross := pc2.Get(ctx, providerA, modelB)
	require.Equal(t, cache.DefaultContextWindow, gotCross.ContextLength,
		"looking up provider A with model B should return defaults")
}
