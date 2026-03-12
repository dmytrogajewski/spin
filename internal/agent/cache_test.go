package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dmytrogajewski/spin/internal/security"
)

func TestNewCommandCache(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)
	assert.NotNil(t, cache)
	assert.Equal(t, 5*time.Second, cache.ttl)
	assert.Equal(t, int64(1024*1024), cache.maxSize)
	assert.Equal(t, int64(0), cache.Size())
}

func TestCommandCache_SetAndGet(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	// Test setting and getting a value.
	key := "test-key"
	result := &Result{
		Command:  &security.Command{Program: "echo", Args: []string{"hello"}},
		Stdout:   "test output",
		Stderr:   "",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
		Error:    nil,
	}
	cache.Set(key, result)

	retrieved, exists := cache.Get(key)
	assert.True(t, exists)
	assert.Equal(t, result, retrieved)
}

func TestCommandCache_GetNonExistent(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	// Test getting a non-existent key.
	retrieved, exists := cache.Get("non-existent")
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

func TestCommandCache_Expiration(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(100*time.Millisecond, 1024*1024)

	key := "test-key"
	result := &Result{
		Command:  &security.Command{Program: "echo", Args: []string{"hello"}},
		Stdout:   "test output",
		Stderr:   "",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
		Error:    nil,
	}
	cache.Set(key, result)

	// Verify it exists immediately.
	retrieved, exists := cache.Get(key)
	assert.True(t, exists)
	assert.Equal(t, result, retrieved)

	// Wait for expiration.
	time.Sleep(150 * time.Millisecond)

	// Verify it's expired.
	retrieved, exists = cache.Get(key)
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

func TestCommandCache_Clear(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	// Set multiple values.
	result1 := &Result{Command: &security.Command{Program: "echo", Args: []string{"1"}}, Stdout: "output1", ExitCode: 0, Duration: 100 * time.Millisecond}
	result2 := &Result{Command: &security.Command{Program: "echo", Args: []string{"2"}}, Stdout: "output2", ExitCode: 0, Duration: 100 * time.Millisecond}
	result3 := &Result{Command: &security.Command{Program: "echo", Args: []string{"3"}}, Stdout: "output3", ExitCode: 0, Duration: 100 * time.Millisecond}

	cache.Set("key1", result1)
	cache.Set("key2", result2)
	cache.Set("key3", result3)

	// Verify they exist.
	_, exists1 := cache.Get("key1")
	_, exists2 := cache.Get("key2")
	_, exists3 := cache.Get("key3")

	assert.True(t, exists1)
	assert.True(t, exists2)
	assert.True(t, exists3)

	// Clear the cache.
	cache.Clear()

	// Verify they're all gone.
	_, exists1 = cache.Get("key1")
	_, exists2 = cache.Get("key2")
	_, exists3 = cache.Get("key3")

	assert.False(t, exists1)
	assert.False(t, exists2)
	assert.False(t, exists3)
}

func TestCommandCache_Size(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	// Initially empty.
	assert.Equal(t, int64(0), cache.Size())

	// Add items.
	result1 := &Result{Command: &security.Command{Program: "echo", Args: []string{"1"}}, Stdout: "output1", ExitCode: 0, Duration: 100 * time.Millisecond}
	result2 := &Result{Command: &security.Command{Program: "echo", Args: []string{"2"}}, Stdout: "output2", ExitCode: 0, Duration: 100 * time.Millisecond}

	cache.Set("key1", result1)
	assert.Positive(t, cache.Size())

	cache.Set("key2", result2)
	assert.Positive(t, cache.Size())

	// Clear.
	cache.Clear()
	assert.Equal(t, int64(0), cache.Size())
}

func TestCommandCache_Key(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	cmd1 := &security.Command{
		Program: "echo",
		Args:    []string{"hello", "world"},
		WorkDir: "/tmp",
	}

	cmd2 := &security.Command{
		Program: "echo",
		Args:    []string{"hello", "world"},
		WorkDir: "/tmp",
	}

	cmd3 := &security.Command{
		Program: "echo",
		Args:    []string{"hello", "universe"},
		WorkDir: "/tmp",
	}

	key1 := cache.Key(cmd1)
	key2 := cache.Key(cmd2)
	key3 := cache.Key(cmd3)

	// Same commands should produce same keys.
	assert.Equal(t, key1, key2)
	// Different commands should produce different keys.
	assert.NotEqual(t, key1, key3)
	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key3)
}

func TestCommandCache_IsCacheable(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	tests := []struct {
		name string
		cmd  *security.Command
		want bool
	}{
		{
			name: "cacheable read-only command",
			cmd: &security.Command{
				Program: "ls",
				Args:    []string{"-la"},
				WorkDir: "/tmp",
			},
			want: true,
		},
		{
			name: "non-cacheable write command",
			cmd: &security.Command{
				Program: "rm",
				Args:    []string{"-rf", "/tmp/test"},
				WorkDir: "/tmp",
			},
			want: false,
		},
		{
			name: "non-cacheable interactive command",
			cmd: &security.Command{
				Program: "vim",
				Args:    []string{"file.txt"},
				WorkDir: "/tmp",
			},
			want: false,
		},
		{
			name: "cacheable git command",
			cmd: &security.Command{
				Program: "git",
				Args:    []string{"status"},
				WorkDir: "/tmp",
			},
			want: true,
		},
		{
			name: "non-cacheable git write command",
			cmd: &security.Command{
				Program: "git",
				Args:    []string{"commit", "-m", "test"},
				WorkDir: "/tmp",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, cache.IsCacheable(tt.cmd))
		})
	}
}

func TestCommandCache_Stats(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	// Initially empty.
	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(1024*1024), stats.MaxSize)
	assert.Equal(t, 0, stats.Entries)

	// Add some items.
	result1 := &Result{Command: &security.Command{Program: "echo", Args: []string{"1"}}, Stdout: "output1", ExitCode: 0, Duration: 100 * time.Millisecond}
	result2 := &Result{Command: &security.Command{Program: "echo", Args: []string{"2"}}, Stdout: "output2", ExitCode: 0, Duration: 100 * time.Millisecond}

	cache.Set("key1", result1)
	cache.Set("key2", result2)

	// Get some items.
	cache.Get("key1")
	cache.Get("key2")
	cache.Get("non-existent")

	stats = cache.Stats()
	assert.Positive(t, stats.Size)
	assert.Equal(t, int64(1024*1024), stats.MaxSize)
	assert.Equal(t, 2, stats.Entries)
}

func TestCommandCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	// Test concurrent reads and writes.
	done := make(chan bool, 2)

	// Writer goroutine.
	go func() {
		for i := range 1000 {
			result := &Result{Command: &security.Command{Program: "echo", Args: []string{string(rune(i))}}, Stdout: string(rune(i)), ExitCode: 0, Duration: 100 * time.Millisecond}
			cache.Set(string(rune(i)), result)
		}

		done <- true
	}()

	// Reader goroutine.
	go func() {
		for i := range 1000 {
			cache.Get(string(rune(i)))
		}

		done <- true
	}()

	// Wait for both goroutines to complete.
	<-done
	<-done

	// Verify cache is in a consistent state.
	assert.GreaterOrEqual(t, cache.Size(), int64(0))
}

func TestCommandCache_SizeLimit(t *testing.T) {
	t.Parallel()
	// Create cache with very small size limit.
	cache := NewCommandCache(5*time.Second, 100)

	// Add items that exceed the size limit.
	for i := range 10 {
		result := &Result{
			Command:  &security.Command{Program: "echo", Args: []string{string(rune(i))}},
			Stdout:   "This is a very long output that should exceed the size limit when multiple items are added",
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		}
		cache.Set(string(rune(i)), result)
	}

	// Cache should not exceed the size limit.
	assert.LessOrEqual(t, cache.Size(), int64(100))
}

func TestCommandCache_UpdateExisting(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	key := "test-key"
	originalResult := &Result{Command: &security.Command{Program: "echo", Args: []string{"original"}}, Stdout: "original", ExitCode: 0, Duration: 100 * time.Millisecond}
	updatedResult := &Result{Command: &security.Command{Program: "echo", Args: []string{"updated"}}, Stdout: "updated", ExitCode: 0, Duration: 100 * time.Millisecond}

	// Set original result.
	cache.Set(key, originalResult)

	// Verify original result.
	retrieved, exists := cache.Get(key)
	assert.True(t, exists)
	assert.Equal(t, originalResult, retrieved)

	// Update result.
	cache.Set(key, updatedResult)

	// Verify updated result.
	retrieved, exists = cache.Get(key)
	assert.True(t, exists)
	assert.Equal(t, updatedResult, retrieved)
}

func TestCommandCache_Eviction(t *testing.T) {
	t.Parallel()
	// Create cache with very small size limit to force eviction.
	cache := NewCommandCache(5*time.Second, 50)

	// Add multiple items that will exceed the size limit.
	for i := range 5 {
		result := &Result{
			Command:  &security.Command{Program: "echo", Args: []string{string(rune(i))}},
			Stdout:   "This is a long output that will cause eviction",
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		}
		cache.Set(string(rune(i)), result)
	}

	// Cache should not exceed the size limit.
	assert.LessOrEqual(t, cache.Size(), int64(50))
}

func TestCacheStats_String(t *testing.T) {
	t.Parallel()
	stats := CacheStats{
		Size:     1024,
		MaxSize:  2048,
		Entries:  5,
		HitRate:  0.8,
		MissRate: 0.2,
	}

	str := stats.String()
	assert.NotEmpty(t, str)
	assert.Contains(t, str, "1024")
	assert.Contains(t, str, "2048")
	assert.Contains(t, str, "5")
}

func TestCommandCache_KeyConsistency(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	cmd := &security.Command{
		Program: "echo",
		Args:    []string{"hello", "world"},
		WorkDir: "/tmp",
	}

	// Generate key multiple times.
	key1 := cache.Key(cmd)
	key2 := cache.Key(cmd)
	key3 := cache.Key(cmd)

	// All keys should be identical.
	assert.Equal(t, key1, key2)
	assert.Equal(t, key2, key3)
	assert.NotEmpty(t, key1)
}

func TestCommandCache_KeyUniqueness(t *testing.T) {
	t.Parallel()
	cache := NewCommandCache(5*time.Second, 1024*1024)

	cmd1 := &security.Command{
		Program: "echo",
		Args:    []string{"hello"},
		WorkDir: "/tmp",
	}

	cmd2 := &security.Command{
		Program: "echo",
		Args:    []string{"world"},
		WorkDir: "/tmp",
	}

	cmd3 := &security.Command{
		Program: "ls",
		Args:    []string{"hello"},
		WorkDir: "/tmp",
	}

	key1 := cache.Key(cmd1)
	key2 := cache.Key(cmd2)
	key3 := cache.Key(cmd3)

	// All keys should be different.
	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key2, key3)
	assert.NotEqual(t, key1, key3)
}

func TestCommandCache_ZeroTTL(t *testing.T) {
	t.Parallel()
	// Test with zero TTL (should expire immediately).
	cache := NewCommandCache(0, 1024*1024)

	key := "test-key"
	result := &Result{
		Command:  &security.Command{Program: "echo", Args: []string{"hello"}},
		Stdout:   "test output",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}
	cache.Set(key, result)

	// Should not be found due to immediate expiration.
	retrieved, exists := cache.Get(key)
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

func TestCommandCache_ZeroMaxSize(t *testing.T) {
	t.Parallel()
	// Test with zero max size (should not cache anything).
	cache := NewCommandCache(5*time.Second, 0)

	key := "test-key"
	result := &Result{
		Command:  &security.Command{Program: "echo", Args: []string{"hello"}},
		Stdout:   "test output",
		ExitCode: 0,
		Duration: 100 * time.Millisecond,
	}
	cache.Set(key, result)

	// Should not be found due to zero max size.
	retrieved, exists := cache.Get(key)
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}
