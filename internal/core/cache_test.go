package core

import (
	"strings"
	"testing"
	"time"
)

func TestCommandCache_GetSet(t *testing.T) {
	cache := NewCommandCache(5*time.Second, 10*1024*1024)

	cmd := &Command{
		Program: "git",
		Args:    []string{"status"},
		WorkDir: "/tmp",
	}

	result := &Result{
		Stdout:   "On branch main",
		ExitCode: 0,
	}

	key := cache.Key(cmd)

	// Initially, should miss
	if _, ok := cache.Get(key); ok {
		t.Fatal("expected cache miss")
	}

	// Set and get
	cache.Set(key, result)

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if got.Stdout != result.Stdout {
		t.Errorf("Stdout = %q, want %q", got.Stdout, result.Stdout)
	}
}

func TestCommandCache_TTL(t *testing.T) {
	cache := NewCommandCache(100*time.Millisecond, 10*1024*1024)

	cmd := &Command{
		Program: "test",
		Args:    []string{},
		WorkDir: "/tmp",
	}

	result := &Result{
		Stdout: "output",
	}

	key := cache.Key(cmd)
	cache.Set(key, result)

	// Should be available immediately
	if _, ok := cache.Get(key); !ok {
		t.Fatal("expected cache hit")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	if _, ok := cache.Get(key); ok {
		t.Fatal("expected cache miss after TTL expiration")
	}
}

func TestCommandCache_Eviction(t *testing.T) {
	// Small cache to force evictions
	cache := NewCommandCache(5*time.Second, 100) // 100 bytes

	// Add entries that exceed cache size
	for i := 0; i < 10; i++ {
		cmd := &Command{
			Program: "test",
			Args:    []string{string(rune(i))},
		}
		result := &Result{
			Stdout: strings.Repeat("x", 20), // 20 bytes each
		}
		key := cache.Key(cmd)
		cache.Set(key, result)
	}

	// Cache size should not exceed max
	if cache.Size() > 100 {
		t.Errorf("cache size = %d, want <= 100", cache.Size())
	}

	// Some entries should have been evicted
	stats := cache.Stats()
	if stats.Entries >= 10 {
		t.Errorf("expected evictions, got %d entries", stats.Entries)
	}
}

func TestCommandCache_Key(t *testing.T) {
	cache := NewCommandCache(5*time.Second, 10*1024*1024)

	tests := []struct {
		name string
		cmd1 *Command
		cmd2 *Command
		same bool
	}{
		{
			name: "identical commands",
			cmd1: &Command{Program: "ls", Args: []string{"-la"}, WorkDir: "/tmp"},
			cmd2: &Command{Program: "ls", Args: []string{"-la"}, WorkDir: "/tmp"},
			same: true,
		},
		{
			name: "different programs",
			cmd1: &Command{Program: "ls", Args: []string{}, WorkDir: "/tmp"},
			cmd2: &Command{Program: "pwd", Args: []string{}, WorkDir: "/tmp"},
			same: false,
		},
		{
			name: "different args",
			cmd1: &Command{Program: "git", Args: []string{"status"}, WorkDir: "/tmp"},
			cmd2: &Command{Program: "git", Args: []string{"log"}, WorkDir: "/tmp"},
			same: false,
		},
		{
			name: "different workdir",
			cmd1: &Command{Program: "ls", Args: []string{}, WorkDir: "/tmp"},
			cmd2: &Command{Program: "ls", Args: []string{}, WorkDir: "/home"},
			same: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := cache.Key(tt.cmd1)
			key2 := cache.Key(tt.cmd2)

			if tt.same && key1 != key2 {
				t.Errorf("expected same keys, got %q and %q", key1, key2)
			}
			if !tt.same && key1 == key2 {
				t.Errorf("expected different keys, got %q", key1)
			}
		})
	}
}

func TestCommandCache_IsCacheable(t *testing.T) {
	cache := NewCommandCache(5*time.Second, 10*1024*1024)

	tests := []struct {
		name      string
		cmd       *Command
		cacheable bool
	}{
		{
			name:      "ls is cacheable",
			cmd:       &Command{Program: "ls", Args: []string{"-la"}},
			cacheable: true,
		},
		{
			name:      "git status is cacheable",
			cmd:       &Command{Program: "git", Args: []string{"status"}},
			cacheable: true,
		},
		{
			name:      "git log is cacheable",
			cmd:       &Command{Program: "git", Args: []string{"log"}},
			cacheable: true,
		},
		{
			name:      "git commit is not cacheable",
			cmd:       &Command{Program: "git", Args: []string{"commit"}},
			cacheable: false,
		},
		{
			name:      "rm is not cacheable",
			cmd:       &Command{Program: "rm", Args: []string{"-rf", "/"}},
			cacheable: false,
		},
		{
			name:      "cat is cacheable",
			cmd:       &Command{Program: "cat", Args: []string{"file.txt"}},
			cacheable: true,
		},
		{
			name:      "pwd is cacheable",
			cmd:       &Command{Program: "pwd"},
			cacheable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.IsCacheable(tt.cmd)
			if got != tt.cacheable {
				t.Errorf("IsCacheable() = %v, want %v", got, tt.cacheable)
			}
		})
	}
}

func TestCommandCache_Clear(t *testing.T) {
	cache := NewCommandCache(5*time.Second, 10*1024*1024)

	// Add some entries
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Program: "test",
			Args:    []string{string(rune(i))},
		}
		result := &Result{Stdout: "output"}
		key := cache.Key(cmd)
		cache.Set(key, result)
	}

	// Verify entries exist
	stats := cache.Stats()
	if stats.Entries == 0 {
		t.Fatal("expected entries in cache")
	}

	// Clear
	cache.Clear()

	// Verify all entries removed
	stats = cache.Stats()
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries after Clear(), got %d", stats.Entries)
	}
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after Clear(), got %d", cache.Size())
	}
}

func TestCommandCache_Stats(t *testing.T) {
	cache := NewCommandCache(5*time.Second, 1024)

	stats := cache.Stats()
	if stats.MaxSize != 1024 {
		t.Errorf("MaxSize = %d, want 1024", stats.MaxSize)
	}
	if stats.Size != 0 {
		t.Errorf("Size = %d, want 0", stats.Size)
	}
	if stats.Entries != 0 {
		t.Errorf("Entries = %d, want 0", stats.Entries)
	}

	// Add entry
	cmd := &Command{Program: "test"}
	result := &Result{Stdout: "output"}
	cache.Set(cache.Key(cmd), result)

	stats = cache.Stats()
	if stats.Entries != 1 {
		t.Errorf("Entries = %d, want 1", stats.Entries)
	}
	if stats.Size == 0 {
		t.Error("expected non-zero size")
	}
}

func TestCommandCache_NilCommand(t *testing.T) {
	cache := NewCommandCache(5*time.Second, 1024)

	key := cache.Key(nil)
	if key != "" {
		t.Errorf("Key(nil) = %q, want empty string", key)
	}

	cacheable := cache.IsCacheable(nil)
	if cacheable {
		t.Error("IsCacheable(nil) = true, want false")
	}
}
