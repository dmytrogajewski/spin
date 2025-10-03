package stream

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuffer(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		expected int
	}{
		{"positive capacity", 1024, 1024},
		{"zero capacity uses default", 0, DefaultBufferCapacity},
		{"negative capacity uses default", -1, DefaultBufferCapacity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffer(tt.capacity)
			assert.NotNil(t, buf)
			assert.Equal(t, tt.expected, buf.Capacity())
			assert.True(t, buf.IsEmpty())
			assert.False(t, buf.IsFull())
		})
	}
}

func TestBuffer_WriteRead(t *testing.T) {
	buf := NewBuffer(1024)

	// Write data
	data := []byte("hello world")
	n, err := buf.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, len(data), buf.Available())
	assert.False(t, buf.IsEmpty())

	// Read data
	read := make([]byte, 1024)
	n, err = buf.Read(read)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, read[:n])
	assert.True(t, buf.IsEmpty())
}

func TestBuffer_MultipleWrites(t *testing.T) {
	buf := NewBuffer(1024)

	writes := []string{"first", "second", "third"}
	totalSize := 0

	for _, s := range writes {
		data := []byte(s)
		n, err := buf.Write(data)
		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		totalSize += len(data)
	}

	assert.Equal(t, totalSize, buf.Available())
}

func TestBuffer_MultipleReads(t *testing.T) {
	buf := NewBuffer(1024)

	// Write some data
	data := []byte("hello world from buffer")
	buf.Write(data)

	// Read in chunks
	chunk1 := make([]byte, 5)
	n, err := buf.Read(chunk1)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(chunk1))

	chunk2 := make([]byte, 6)
	n, err = buf.Read(chunk2)
	require.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, " world", string(chunk2))

	remaining := make([]byte, 100)
	n, err = buf.Read(remaining)
	require.NoError(t, err)
	assert.Equal(t, len(data)-5-6, n)
	assert.Equal(t, " from buffer", string(remaining[:n]))

	assert.True(t, buf.IsEmpty())
}

func TestBuffer_Overflow(t *testing.T) {
	buf := NewBuffer(10)

	// Write data larger than capacity
	data := make([]byte, 15)
	for i := range data {
		data[i] = byte('a' + i)
	}

	n, err := buf.Write(data)
	assert.Error(t, err)
	assert.Equal(t, ErrBufferFull, err)
	assert.Equal(t, 10, n) // Only capacity bytes written
	assert.True(t, buf.IsFull())
}

func TestBuffer_ReadEmpty(t *testing.T) {
	buf := NewBuffer(1024)

	read := make([]byte, 10)
	n, err := buf.Read(read)
	assert.Error(t, err)
	assert.Equal(t, ErrBufferEmpty, err)
	assert.Equal(t, 0, n)
}

func TestBuffer_Reset(t *testing.T) {
	buf := NewBuffer(1024)

	// Write some data
	data := []byte("test data")
	buf.Write(data)
	assert.False(t, buf.IsEmpty())
	assert.Equal(t, len(data), buf.Available())

	// Reset
	buf.Reset()
	assert.True(t, buf.IsEmpty())
	assert.Equal(t, 0, buf.Available())
}

func TestBuffer_WrapAround(t *testing.T) {
	buf := NewBuffer(10)

	// Write to fill buffer
	data1 := []byte("12345")
	n, err := buf.Write(data1)
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	// Read some
	read1 := make([]byte, 3)
	n, err = buf.Read(read1)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "123", string(read1))

	// Write more (should wrap around)
	data2 := []byte("ABCDEFGH")
	n, err = buf.Write(data2)
	require.NoError(t, err)
	assert.Equal(t, 8, n)

	// Read all
	readAll := make([]byte, 20)
	n, err = buf.Read(readAll)
	require.NoError(t, err)
	assert.Equal(t, "45ABCDEFGH", string(readAll[:n]))
}

func TestBuffer_Concurrent(t *testing.T) {
	buf := NewBuffer(1024)
	var wg sync.WaitGroup

	// Concurrent writes
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()
			data := []byte{byte(id)}
			buf.Write(data)
		}(i)
	}

	// Concurrent reads
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			read := make([]byte, 1)
			buf.Read(read)
		}()
	}

	wg.Wait()
	// Should not panic or deadlock
}

func TestBuffer_Available(t *testing.T) {
	buf := NewBuffer(100)

	assert.Equal(t, 0, buf.Available())

	buf.Write([]byte("12345"))
	assert.Equal(t, 5, buf.Available())

	read := make([]byte, 2)
	buf.Read(read)
	assert.Equal(t, 3, buf.Available())

	buf.Write([]byte("67890"))
	assert.Equal(t, 8, buf.Available())
}

func TestBuffer_Capacity(t *testing.T) {
	capacities := []int{256, 1024, 4096, 8192}

	for _, cap := range capacities {
		buf := NewBuffer(cap)
		assert.Equal(t, cap, buf.Capacity())
	}
}

func TestBuffer_IsFullEmpty(t *testing.T) {
	buf := NewBuffer(10)

	// Initially empty
	assert.True(t, buf.IsEmpty())
	assert.False(t, buf.IsFull())

	// Partially filled
	buf.Write([]byte("12345"))
	assert.False(t, buf.IsEmpty())
	assert.False(t, buf.IsFull())

	// Full
	buf.Write([]byte("67890"))
	assert.False(t, buf.IsEmpty())
	assert.True(t, buf.IsFull())

	// Read all
	read := make([]byte, 20)
	buf.Read(read)
	assert.True(t, buf.IsEmpty())
	assert.False(t, buf.IsFull())
}

func TestBuffer_MinMaxCapacity(t *testing.T) {
	// Small capacity allowed for testing
	bufMin := NewBuffer(10)
	assert.Equal(t, 10, bufMin.Capacity())

	// Max capacity (shouldn't exceed)
	bufMax := NewBuffer(MaxBufferCapacity + 1000)
	assert.Equal(t, MaxBufferCapacity, bufMax.Capacity())
}

func TestBuffer_WriteZeroLength(t *testing.T) {
	buf := NewBuffer(1024)

	n, err := buf.Write([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.True(t, buf.IsEmpty())
}

func TestBuffer_ReadZeroLength(t *testing.T) {
	buf := NewBuffer(1024)
	buf.Write([]byte("test"))

	n, err := buf.Read([]byte{})
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func BenchmarkBuffer_Write(b *testing.B) {
	buf := NewBuffer(4096)
	data := []byte("benchmark data for write operation")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(data)
	}
}

func BenchmarkBuffer_Read(b *testing.B) {
	buf := NewBuffer(4096)
	data := []byte("benchmark data for read operation")
	buf.Write(data)

	read := make([]byte, len(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(data)
		buf.Read(read)
	}
}

func BenchmarkBuffer_WriteRead(b *testing.B) {
	buf := NewBuffer(4096)
	data := []byte("benchmark data")
	read := make([]byte, len(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Write(data)
		buf.Read(read)
	}
}

func BenchmarkBuffer_Concurrent(b *testing.B) {
	buf := NewBuffer(4096)
	data := []byte("concurrent benchmark")

	b.RunParallel(func(pb *testing.PB) {
		read := make([]byte, len(data))
		for pb.Next() {
			buf.Write(data)
			buf.Read(read)
		}
	})
}
