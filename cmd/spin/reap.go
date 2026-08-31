package main

import "github.com/dmytrogajewski/spin/internal/agent/child"

// reapParentOrphans removes stale child pid/socket files from a prior parent.
func reapParentOrphans() {
	_ = child.ReapOnStart()
}
