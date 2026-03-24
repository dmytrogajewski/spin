//go:build !windows

package storage

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"
)

// flockRetryInterval is the delay between non-blocking flock attempts.
const flockRetryInterval = 10 * time.Millisecond

// FlockWithContext attempts to acquire a file lock using [syscall.Flock] with
// non-blocking retries. It polls with LOCK_NB and checks ctx.Done() between
// attempts. If the context is canceled before the lock is acquired, it returns
// the wrapped context error. The how parameter should include the desired lock
// type (LOCK_EX or LOCK_SH); LOCK_NB is added automatically.
func FlockWithContext(ctx context.Context, fd, how int) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("flock: %w", err)
	}

	// First attempt — fast path without sleep.
	err := syscall.Flock(fd, how|syscall.LOCK_NB)
	if err == nil {
		return nil
	}

	if !errors.Is(err, syscall.EWOULDBLOCK) {
		return fmt.Errorf("flock: %w", err)
	}

	// Retry loop with context-aware polling.
	ticker := time.NewTicker(flockRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("flock: %w", ctx.Err())
		case <-ticker.C:
			err = syscall.Flock(fd, how|syscall.LOCK_NB)
			if err == nil {
				return nil
			}

			if !errors.Is(err, syscall.EWOULDBLOCK) {
				return fmt.Errorf("flock: %w", err)
			}
		}
	}
}

// FlockExclusiveWithContext acquires an exclusive file lock with context support.
func FlockExclusiveWithContext(ctx context.Context, fd int) error {
	return FlockWithContext(ctx, fd, syscall.LOCK_EX)
}

// FlockSharedWithContext acquires a shared file lock with context support.
func FlockSharedWithContext(ctx context.Context, fd int) error {
	return FlockWithContext(ctx, fd, syscall.LOCK_SH)
}

// FlockUnlock releases a file lock.
func FlockUnlock(fd int) error {
	err := syscall.Flock(fd, syscall.LOCK_UN)
	if err != nil {
		return fmt.Errorf("flock unlock: %w", err)
	}

	return nil
}

// SafeFlockFd converts a file descriptor from uintptr to int for [syscall.Flock].
// File descriptors are always small non-negative values on supported platforms.
func SafeFlockFd(fd uintptr) int {
	const maxFd = int(^uint(0) >> 1)
	if fd > uintptr(maxFd) {
		return -1
	}

	return int(fd)
}
