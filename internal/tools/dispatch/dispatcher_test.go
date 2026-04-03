package dispatch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type testTool struct {
	name string
}

type testParams struct {
	data map[string]string
}

func (p testParams) GetStringOr(key, fallback string) string {
	if v, ok := p.data[key]; ok {
		return v
	}

	return fallback
}

func (p testParams) GetIntOr(_ string, fallback int) int {
	return fallback
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	d := New[*testTool]()
	d.Register("greet", func(_ context.Context, tool *testTool, params Params) (Result, error) {
		name := params.GetStringOr("name", "world")

		return OK("hello " + name + " from " + tool.name)
	})

	tool := &testTool{name: "test"}
	params := testParams{data: map[string]string{"operation": "greet", "name": "alice"}}

	result, err := d.Dispatch(context.Background(), tool, "operation", params)
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Equal(t, "hello alice from test", result.Content)
}

func TestDispatcher_UnknownOperation(t *testing.T) {
	t.Parallel()

	d := New[*testTool]()
	d.Register("greet", func(_ context.Context, _ *testTool, _ Params) (Result, error) {
		return OK("hi")
	})

	params := testParams{data: map[string]string{"operation": "unknown"}}

	result, err := d.Dispatch(context.Background(), &testTool{}, "operation", params)
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Contains(t, result.Content, "unknown operation: unknown")
}

func TestDispatcher_MissingOperation(t *testing.T) {
	t.Parallel()

	d := New[*testTool]()

	params := testParams{data: map[string]string{}}

	result, err := d.Dispatch(context.Background(), &testTool{}, "operation", params)
	require.NoError(t, err)
	require.True(t, result.IsError)
	require.Contains(t, result.Content, "operation parameter is required")
}

func TestDispatcher_Operations(t *testing.T) {
	t.Parallel()

	d := New[*testTool]()
	d.Register("alpha", func(_ context.Context, _ *testTool, _ Params) (Result, error) { return OK("a") })
	d.Register("beta", func(_ context.Context, _ *testTool, _ Params) (Result, error) { return OK("b") })

	ops := d.Operations()
	require.Equal(t, []string{"alpha", "beta"}, ops)
}

func TestDispatcher_MultipleHandlers(t *testing.T) {
	t.Parallel()

	d := New[*testTool]()
	d.Register("add", func(_ context.Context, _ *testTool, _ Params) (Result, error) { return OK("added") })
	d.Register("remove", func(_ context.Context, _ *testTool, _ Params) (Result, error) { return OK("removed") })

	t.Run("add", func(t *testing.T) {
		t.Parallel()

		result, err := d.Dispatch(context.Background(), &testTool{}, "op", testParams{data: map[string]string{"op": "add"}})
		require.NoError(t, err)
		require.Equal(t, "added", result.Content)
	})

	t.Run("remove", func(t *testing.T) {
		t.Parallel()

		result, err := d.Dispatch(context.Background(), &testTool{}, "op", testParams{data: map[string]string{"op": "remove"}})
		require.NoError(t, err)
		require.Equal(t, "removed", result.Content)
	})
}
