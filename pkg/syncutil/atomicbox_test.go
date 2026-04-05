package syncutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtomicBox_ReadWrite(t *testing.T) {
	t.Parallel()

	box := NewAtomicBox(42)
	require.Equal(t, 42, box.Read())

	box.Write(100)
	require.Equal(t, 100, box.Read())
}

func TestAtomicBox_Update(t *testing.T) {
	t.Parallel()

	box := NewAtomicBox(10)
	result := box.Update(func(v int) int { return v + 5 })

	require.Equal(t, 15, result)
	require.Equal(t, 15, box.Read())
}

func TestAtomicBox_ReadWith(t *testing.T) {
	t.Parallel()

	box := NewAtomicBox([]string{"a", "b", "c"})
	length := box.ReadWith(func(v []string) []string { return v })

	require.Len(t, length, 3)
}

func TestAtomicBox_ZeroValue(t *testing.T) {
	t.Parallel()

	box := NewAtomicBox("")
	require.Empty(t, box.Read())

	box.Write("hello")
	require.Equal(t, "hello", box.Read())
}

func TestAtomicBox_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	box := NewAtomicBox(0)

	const goroutines = 100

	const increments = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			for range increments {
				box.Update(func(v int) int { return v + 1 })
			}
		}()
	}

	wg.Wait()
	require.Equal(t, goroutines*increments, box.Read())
}

func TestAtomicBox_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	box := NewAtomicBox("initial")

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()

			box.Write("updated")
		}()

		go func() {
			defer wg.Done()

			val := box.Read()
			assert.Contains(t, []string{"initial", "updated"}, val)
		}()
	}

	wg.Wait()
}
