package syncmap

import (
	"strconv"
	"testing"
)

func BenchmarkMap_Set(b *testing.B) {
	m := New[string, int]()

	b.ResetTimer()

	for i := range b.N {
		m.Set(strconv.Itoa(i%1000), i)
	}
}

func BenchmarkMap_Get(b *testing.B) {
	m := New[string, int]()

	const populateSize = 1000

	for i := range populateSize {
		m.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()

	for i := range b.N {
		m.Get(strconv.Itoa(i % populateSize))
	}
}

func BenchmarkMap_GetOrCreate(b *testing.B) {
	m := New[string, int]()

	const populateSize = 1000

	for i := range populateSize {
		m.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()

	for i := range b.N {
		m.GetOrCreate(strconv.Itoa(i%populateSize), func() int { return i })
	}
}

func BenchmarkMap_Range(b *testing.B) {
	m := New[string, int]()

	const populateSize = 100

	for i := range populateSize {
		m.Set(strconv.Itoa(i), i)
	}

	b.ResetTimer()

	for range b.N {
		m.Range(func(_ string, _ int) bool { return true })
	}
}
