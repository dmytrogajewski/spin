// Package hashx provides zero-dependency hashing and ID generation utilities
// that consolidate duplicated hash/ID operations across the codebase.
package hashx

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync/atomic"
)

// shortHashLen is the number of hex characters in a ShortHash result.
const shortHashLen = 32

// hexCharsPerByte is the number of hex characters per byte.
const hexCharsPerByte = 2

// SHA256Hex returns the lowercase hex-encoded SHA-256 hash of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// ShortHash returns a 32-character lowercase hex hash of data,
// derived from the first 16 bytes of its SHA-256 digest.
// Suitable for cache keying and content addressing where full
// SHA-256 length is unnecessary.
func ShortHash(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:shortHashLen/hexCharsPerByte])
}

// RandomHexID returns a cryptographically random hex string of the
// specified character length. Returns an empty string if length <= 0.
// Panics if the system's crypto/rand source fails (unrecoverable).
func RandomHexID(length int) string {
	if length <= 0 {
		return ""
	}

	// Each byte encodes to 2 hex characters; round up.
	byteLen := (length + 1) / hexCharsPerByte
	buf := make([]byte, byteLen)

	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}

	return hex.EncodeToString(buf)[:length]
}

// NewAtomicIDGenerator returns a closure that produces monotonically increasing
// IDs in the form "prefix-1", "prefix-2", etc. Thread-safe via atomic counter.
func NewAtomicIDGenerator(prefix string) func() string {
	var counter atomic.Int64

	return func() string {
		seq := counter.Add(1)

		return fmt.Sprintf("%s-%d", prefix, seq)
	}
}
