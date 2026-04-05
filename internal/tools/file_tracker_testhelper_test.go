package tools

import "time"

// modTimeTolerance mirrors the tolerance from pkg/alg/pathx for test use.
const modTimeTolerance = 50 * time.Millisecond

// sleepBeyondTolerance sleeps long enough to exceed the filesystem timestamp tolerance.
const sleepBeyondTolerance = modTimeTolerance * 3
