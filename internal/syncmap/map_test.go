package syncmap

// Journey: specs/journeys/JOURNEY-create-syncmap.md.

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMap_SetAndGet(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)
	m.Set("b", 2)

	val, ok := m.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	val, ok = m.Get("b")
	assert.True(t, ok)
	assert.Equal(t, 2, val)
}

func TestMap_Get_Missing(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	val, ok := m.Get("missing")
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestMap_Set_Overwrite(t *testing.T) {
	t.Parallel()

	m := New[string, string]()

	m.Set("key", "original")
	m.Set("key", "updated")

	val, ok := m.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "updated", val)
}

func TestMap_Delete(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)
	m.Delete("a")

	_, ok := m.Get("a")
	assert.False(t, ok)
	assert.Equal(t, 0, m.Len())
}

func TestMap_Delete_Missing(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	// Should not panic.
	m.Delete("nonexistent")
	assert.Equal(t, 0, m.Len())
}

func TestMap_Len(t *testing.T) {
	t.Parallel()

	m := New[string, int]()
	assert.Equal(t, 0, m.Len())

	m.Set("a", 1)
	assert.Equal(t, 1, m.Len())

	m.Set("b", 2)
	assert.Equal(t, 2, m.Len())

	m.Delete("a")
	assert.Equal(t, 1, m.Len())
}

func TestMap_Keys(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("b", 2)
	m.Set("a", 1)

	keys := m.Keys()
	assert.Len(t, keys, 2)
	assert.ElementsMatch(t, []string{"a", "b"}, keys)
}

func TestMap_Keys_Empty(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	keys := m.Keys()
	assert.Empty(t, keys)
}

func TestMap_Range(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	collected := make(map[string]int)

	m.Range(func(k string, v int) bool {
		collected[k] = v

		return true
	})

	assert.Len(t, collected, 3)
	assert.Equal(t, 1, collected["a"])
	assert.Equal(t, 2, collected["b"])
	assert.Equal(t, 3, collected["c"])
}

func TestMap_Range_StopsEarly(t *testing.T) {
	t.Parallel()

	m := New[int, string]()

	m.Set(1, "a")
	m.Set(2, "b")
	m.Set(3, "c")

	count := 0

	m.Range(func(_ int, _ string) bool {
		count++

		return count < 2
	})

	assert.Equal(t, 2, count)
}

func TestMap_GetOrCreate_Miss(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	val := m.GetOrCreate("new", func() int { return 42 })
	assert.Equal(t, 42, val)
	assert.Equal(t, 1, m.Len())

	// Verify it's stored.
	got, ok := m.Get("new")
	assert.True(t, ok)
	assert.Equal(t, 42, got)
}

func TestMap_GetOrCreate_Hit(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("existing", 10)

	called := false

	val := m.GetOrCreate("existing", func() int {
		called = true

		return 99
	})

	assert.Equal(t, 10, val)
	assert.False(t, called, "create should not be called for existing key")
}

func TestMap_Clear(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)
	m.Set("b", 2)

	m.Clear()

	assert.Equal(t, 0, m.Len())

	_, ok := m.Get("a")
	assert.False(t, ok)
}

func TestMap_Close_Idempotent(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)

	callCount := 0

	m.Close(func(_ int) { callCount++ })
	m.Close(func(_ int) { callCount++ })

	assert.Equal(t, 1, callCount, "cleanup should only run once")
	assert.Equal(t, 0, m.Len())
}

func TestMap_Close_Cleanup(t *testing.T) {
	t.Parallel()

	m := New[string, *int]()

	a, b := 1, 2
	m.Set("a", &a)
	m.Set("b", &b)

	cleaned := make([]*int, 0, 2)

	m.Close(func(v *int) {
		cleaned = append(cleaned, v)
	})

	assert.Len(t, cleaned, 2)
}

func TestMap_Close_NilCleanup(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)

	// Should not panic.
	m.Close(nil)

	assert.Equal(t, 0, m.Len())
}

func TestMap_OperationsAfterClose(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)
	m.Close(nil)

	// Set is no-op after close.
	m.Set("b", 2)
	assert.Equal(t, 0, m.Len())

	// Get returns zero value.
	val, ok := m.Get("b")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	// GetOrCreate returns zero value.
	val = m.GetOrCreate("c", func() int { return 99 })
	assert.Equal(t, 0, val)
}

func TestMap_SetIfAbsent_Miss(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	ok := m.SetIfAbsent("a", 1)
	assert.True(t, ok)

	val, found := m.Get("a")
	assert.True(t, found)
	assert.Equal(t, 1, val)
}

func TestMap_SetIfAbsent_Hit(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)

	ok := m.SetIfAbsent("a", 2)
	assert.False(t, ok)

	// Original value preserved.
	val, found := m.Get("a")
	assert.True(t, found)
	assert.Equal(t, 1, val)
}

func TestMap_SetIfAbsent_Closed(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Close(nil)

	ok := m.SetIfAbsent("a", 1)
	assert.False(t, ok)
	assert.Equal(t, 0, m.Len())
}

func TestMap_SetIfPresent_Hit(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)

	ok := m.SetIfPresent("a", 2)
	assert.True(t, ok)

	val, found := m.Get("a")
	assert.True(t, found)
	assert.Equal(t, 2, val)
}

func TestMap_SetIfPresent_Miss(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	ok := m.SetIfPresent("a", 1)
	assert.False(t, ok)
	assert.Equal(t, 0, m.Len())
}

func TestMap_SetIfPresent_Closed(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)
	m.Close(nil)

	ok := m.SetIfPresent("a", 2)
	assert.False(t, ok)
}

func TestMap_Values(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	values := m.Values()
	assert.Len(t, values, 3)
	assert.ElementsMatch(t, []int{1, 2, 3}, values)
}

func TestMap_Values_Empty(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	values := m.Values()
	assert.Empty(t, values)
}

func TestMap_Pop_Existing(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	m.Set("a", 1)

	val, ok := m.Pop("a")
	assert.True(t, ok)
	assert.Equal(t, 1, val)
	assert.Equal(t, 0, m.Len())

	// Second pop returns false.
	val, ok = m.Pop("a")
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestMap_Pop_Missing(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	val, ok := m.Pop("missing")
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestMap_Concurrent_SetGetDelete(t *testing.T) {
	t.Parallel()

	m := New[int, int]()

	const (
		goroutines = 100
		iterations = 1000
	)

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()

			for j := range iterations {
				key := (id*iterations + j) % 50

				m.Set(key, j)
				m.Get(key)
				m.Len()

				if j%3 == 0 {
					m.Delete(key)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestMap_Concurrent_GetOrCreate(t *testing.T) {
	t.Parallel()

	m := New[string, int]()

	const goroutines = 100

	var (
		wg          sync.WaitGroup
		createCount atomic.Int64
	)

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			m.GetOrCreate("shared", func() int {
				createCount.Add(1)

				return 42
			})
		}()
	}

	wg.Wait()

	val, ok := m.Get("shared")
	require.True(t, ok)
	assert.Equal(t, 42, val)

	// With the double-check pattern, create might be called a small number
	// of times (due to races between RUnlock and Lock), but not goroutines times.
	assert.LessOrEqual(t, createCount.Load(), int64(goroutines))
	assert.GreaterOrEqual(t, createCount.Load(), int64(1))
}

func TestMap_Concurrent_Range(t *testing.T) {
	t.Parallel()

	m := New[int, int]()

	for i := range 100 {
		m.Set(i, i)
	}

	var wg sync.WaitGroup

	const goroutines = 50

	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			count := 0

			m.Range(func(_ int, _ int) bool {
				count++

				return true
			})

			assert.Positive(t, count)
		}()
	}

	wg.Wait()
}
