package hashx

// Journey: specs/journeys/JOURNEY-R5.md.

import (
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hexPattern matches a string containing only lowercase hex characters.
var hexPattern = regexp.MustCompile(`^[0-9a-f]+$`)

// sha256EmptyHex is the SHA-256 hash of an empty byte slice (NIST FIPS 180-4).
const sha256EmptyHex = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// sha256ABCHex is the SHA-256 hash of "abc" (NIST FIPS 180-4).
const sha256ABCHex = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

// shortHashEmptyHex is the first 32 chars of sha256EmptyHex.
const shortHashEmptyHex = "e3b0c44298fc1c149afbf4c8996fb924"

// shortHashABCHex is the first 32 chars of sha256ABCHex.
const shortHashABCHex = "ba7816bf8f01cfea414140de5dae2223"

// expectedSHA256HexLen is the expected length of a SHA-256 hex string.
const expectedSHA256HexLen = 64

// expectedShortHashLen is the expected length of a ShortHash result.
const expectedShortHashLen = 32

func TestSHA256Hex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{name: "empty_input", input: []byte{}, want: sha256EmptyHex},
		{name: "abc", input: []byte("abc"), want: sha256ABCHex},
		{name: "nil_input", input: nil, want: sha256EmptyHex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SHA256Hex(tt.input)
			require.Equal(t, tt.want, got)
			require.Len(t, got, expectedSHA256HexLen)
		})
	}
}

func TestShortHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{name: "empty_input", input: []byte{}, want: shortHashEmptyHex},
		{name: "abc", input: []byte("abc"), want: shortHashABCHex},
		{name: "nil_input", input: nil, want: shortHashEmptyHex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ShortHash(tt.input)
			require.Equal(t, tt.want, got)
			require.Len(t, got, expectedShortHashLen)
		})
	}
}

func TestShortHash_is_prefix_of_SHA256(t *testing.T) {
	t.Parallel()

	data := []byte("deterministic content")
	full := SHA256Hex(data)
	short := ShortHash(data)

	require.Equal(t, full[:expectedShortHashLen], short)
}

func TestRandomHexID(t *testing.T) {
	t.Parallel()

	t.Run("correct_length", func(t *testing.T) {
		t.Parallel()

		for _, length := range []int{1, 7, 16, 32} {
			got := RandomHexID(length)
			require.Len(t, got, length)
		}
	})

	t.Run("hex_characters_only", func(t *testing.T) {
		t.Parallel()

		got := RandomHexID(32)
		require.Regexp(t, hexPattern, got)
	})

	t.Run("uniqueness", func(t *testing.T) {
		t.Parallel()

		seen := make(map[string]bool)

		for range 100 {
			id := RandomHexID(16)
			require.False(t, seen[id], "duplicate ID generated: %s", id)

			seen[id] = true
		}
	})

	t.Run("zero_length_returns_empty", func(t *testing.T) {
		t.Parallel()

		got := RandomHexID(0)
		require.Empty(t, got)
	})
}

func TestNewAtomicIDGenerator(t *testing.T) {
	t.Parallel()

	t.Run("monotonically_increasing", func(t *testing.T) {
		t.Parallel()

		gen := NewAtomicIDGenerator("task")

		require.Equal(t, "task-1", gen())
		require.Equal(t, "task-2", gen())
		require.Equal(t, "task-3", gen())
	})

	t.Run("correct_prefix", func(t *testing.T) {
		t.Parallel()

		gen := NewAtomicIDGenerator("block")
		id := gen()

		require.Contains(t, id, "block-")
	})

	t.Run("independent_generators", func(t *testing.T) {
		t.Parallel()

		genA := NewAtomicIDGenerator("a")
		genB := NewAtomicIDGenerator("b")

		require.Equal(t, "a-1", genA())
		require.Equal(t, "b-1", genB())
		require.Equal(t, "a-2", genA())
	})

	t.Run("concurrent_safety", func(t *testing.T) {
		t.Parallel()

		gen := NewAtomicIDGenerator("conc")
		seen := &sync.Map{}
		concurrency := 100

		var wg sync.WaitGroup

		wg.Add(concurrency)

		for range concurrency {
			go func() {
				defer wg.Done()

				id := gen()

				_, loaded := seen.LoadOrStore(id, true)
				// Use assert (not require) inside goroutines.
				assert.False(t, loaded, "duplicate ID: %s", id)
			}()
		}

		wg.Wait()
	})
}
