//go:build !windows

package storage

// Journey: specs/journeys/JOURNEY-CTX-2.1.md.

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlockExclusiveWithContext_AcquiresImmediately(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)

	err := FlockExclusiveWithContext(t.Context(), fd)
	require.NoError(t, err)

	t.Cleanup(func() { _ = FlockUnlock(fd) })
}

func TestFlockSharedWithContext_AcquiresImmediately(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)

	err := FlockSharedWithContext(t.Context(), fd)
	require.NoError(t, err)

	t.Cleanup(func() { _ = FlockUnlock(fd) })
}

func TestFlockWithContext_CanceledContextReturnsError(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)

	// Hold exclusive lock so the second attempt will block.
	err := FlockExclusiveWithContext(t.Context(), fd)
	require.NoError(t, err)

	// Open a second fd on the same file.
	secondFd := openSameFd(t, fd)

	ctx := canceledContext()

	err = FlockExclusiveWithContext(ctx, secondFd)
	require.ErrorIs(t, err, context.Canceled)

	t.Cleanup(func() { _ = FlockUnlock(fd) })
}

func TestFlockWithContext_ContendedThenReleased(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)

	// Hold exclusive lock.
	err := FlockExclusiveWithContext(t.Context(), fd)
	require.NoError(t, err)

	secondFd := openSameFd(t, fd)

	// Release lock after a short delay.
	const releaseDelay = 50 * time.Millisecond

	go func() {
		time.Sleep(releaseDelay)

		_ = FlockUnlock(fd)
	}()

	// Second fd should acquire after the release.
	err = FlockExclusiveWithContext(t.Context(), secondFd)
	require.NoError(t, err)

	t.Cleanup(func() { _ = FlockUnlock(secondFd) })
}

func TestFlockWithContext_CancelDuringContention(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)

	// Hold exclusive lock.
	err := FlockExclusiveWithContext(t.Context(), fd)
	require.NoError(t, err)

	secondFd := openSameFd(t, fd)

	const cancelDelay = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(t.Context(), cancelDelay)
	defer cancel()

	start := time.Now()

	err = FlockExclusiveWithContext(ctx, secondFd)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// Should have returned near the cancel delay, not hung forever.
	elapsed := time.Since(start)
	assert.Less(t, elapsed, time.Second)

	t.Cleanup(func() { _ = FlockUnlock(fd) })
}

func TestFlockUnlock_ReleasesLock(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)

	err := FlockExclusiveWithContext(t.Context(), fd)
	require.NoError(t, err)

	err = FlockUnlock(fd)
	require.NoError(t, err)

	// Should be able to acquire again.
	secondFd := openSameFd(t, fd)

	err = FlockExclusiveWithContext(t.Context(), secondFd)
	require.NoError(t, err)

	t.Cleanup(func() { _ = FlockUnlock(secondFd) })
}

func TestFlockSharedWithContext_MultipleReaders(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)

	// Multiple shared locks should succeed concurrently.
	const readerCount = 3

	fds := make([]int, readerCount)
	fds[0] = fd

	for idx := range readerCount - 1 {
		fds[idx+1] = openSameFd(t, fd)
	}

	var wg sync.WaitGroup

	wg.Add(readerCount)

	for _, lockFd := range fds {
		go func() {
			defer wg.Done()

			lockErr := FlockSharedWithContext(t.Context(), lockFd)
			assert.NoError(t, lockErr)
		}()
	}

	wg.Wait()

	for _, lockFd := range fds {
		_ = FlockUnlock(lockFd)
	}
}

func TestSafeFlockFd_ValidFd(t *testing.T) {
	t.Parallel()

	require.Equal(t, 3, SafeFlockFd(3))
	require.Equal(t, 0, SafeFlockFd(0))
}

func TestSafeFlockFd_MaxUintptr(t *testing.T) {
	t.Parallel()

	require.Equal(t, -1, SafeFlockFd(^uintptr(0)))
}

func TestFlockWithContext_PreCanceledContext(t *testing.T) {
	t.Parallel()

	fd := openTempFd(t)
	ctx := canceledContext()

	err := FlockExclusiveWithContext(ctx, fd)
	require.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "flock")
}

// openTempFd creates a temp file and returns its fd (as int).
func openTempFd(t *testing.T) int {
	t.Helper()

	path := filepath.Join(t.TempDir(), "lockfile")

	file, err := os.Create(path)
	require.NoError(t, err)

	fd := SafeFlockFd(file.Fd())
	require.NotEqual(t, -1, fd)

	t.Cleanup(func() { file.Close() })

	return fd
}

// openSameFd opens a second file descriptor to the same file for lock contention tests.
func openSameFd(t *testing.T, existingFd int) int {
	t.Helper()

	// Read the path from /proc/self/fd (Linux-specific, works in tests).
	path, err := os.Readlink(procFdPath(existingFd))
	require.NoError(t, err)

	file, err := os.OpenFile(path, os.O_RDWR, 0)
	require.NoError(t, err)

	fd := SafeFlockFd(file.Fd())
	require.NotEqual(t, -1, fd)

	t.Cleanup(func() { file.Close() })

	return fd
}

// procFdPath returns the /proc path for a file descriptor.
func procFdPath(fd int) string {
	return filepath.Join("/proc/self/fd", strconv.Itoa(fd))
}
