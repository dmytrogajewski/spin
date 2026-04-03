package syncutil

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errInitFailed = errors.New("init failed")

func TestLazyInit_Get(t *testing.T) {
	t.Parallel()

	lazy := NewLazyInit(func() (int, error) {
		return 42, nil
	})

	val, err := lazy.Get()
	require.NoError(t, err)
	require.Equal(t, 42, val)
}

func TestLazyInit_Error(t *testing.T) {
	t.Parallel()

	lazy := NewLazyInit(func() (int, error) {
		return 0, errInitFailed
	})

	val, err := lazy.Get()
	require.ErrorIs(t, err, errInitFailed)
	require.Equal(t, 0, val)
}

func TestLazyInit_CalledOnce(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	lazy := NewLazyInit(func() (string, error) {
		callCount.Add(1)

		return "result", nil
	})

	val1, err1 := lazy.Get()
	val2, err2 := lazy.Get()

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.Equal(t, "result", val1)
	require.Equal(t, "result", val2)
	require.Equal(t, int32(1), callCount.Load())
}

func TestLazyInit_ConcurrentGet(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	lazy := NewLazyInit(func() (int, error) {
		callCount.Add(1)

		return 99, nil
	})

	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			val, err := lazy.Get()
			assert.NoError(t, err)
			assert.Equal(t, 99, val)
		}()
	}

	wg.Wait()
	require.Equal(t, int32(1), callCount.Load())
}

func TestLazyInit_ErrorCached(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	lazy := NewLazyInit(func() (int, error) {
		callCount.Add(1)

		return 0, errInitFailed
	})

	_, err1 := lazy.Get()
	_, err2 := lazy.Get()

	require.ErrorIs(t, err1, errInitFailed)
	require.ErrorIs(t, err2, errInitFailed)
	require.Equal(t, int32(1), callCount.Load())
}
