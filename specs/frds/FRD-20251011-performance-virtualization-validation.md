# Feature Requirements Document

**FRD ID:** FRD-20251011-performance-virtualization-validation
**Feature:** Performance & Virtualization Validation (Phase 7.2)
**Author:** Spin Agent
**Created:** 2025-10-11
**Status:** Draft → Implementation
**Roadmap:** [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 7.2
**Related FRDs:**
- [FRD-20251010-block-timeline.md](FRD-20251010-block-timeline.md)
- [FRD-20251010-append-only-printer.md](FRD-20251010-append-only-printer.md)

---

## 1. Overview

Validate that the TUI meets performance requirements under stress: 10k+ blocks render smoothly, streaming doesn't stutter, and virtualization works correctly. This phase establishes performance benchmarks and identifies optimization opportunities.

### 1.1 Goals

- **Validate spec requirements:** 10k+ blocks timeline scrolls smoothly
- **Benchmark streaming:** 100k lines throughput without lag
- **Benchmark viewport:** Virtualized timeline rendering stays under 16ms (60fps)
- **Profile & optimize:** Identify bottlenecks, optimize hot paths
- **Document performance:** Set SLOs and publish characteristics

### 1.2 Non-Goals

- GUI performance (out of scope)
- Network latency optimization (handled in llm package)
- Memory optimization beyond leak prevention
- Embedded/low-power device support

---

## 2. Requirements

### 2.1 Performance Targets (from Spec Section 12)

Per [tui-new.md#12](../tui-implementation/tui-new.md#L185):

1. **Block capacity:** Must smoothly render **10k+ blocks** via viewport virtualization
2. **Streaming:** Large outputs (incremental append) for EXECUTE blocks
3. **Memory:** Keep last N blocks fully in memory; older summarized with lazy expansion

### 2.2 Derived SLOs

From industry standards (60fps = 16.67ms frame budget):

| Metric | Target | Stretch | Method |
|--------|--------|---------|--------|
| Viewport render | <16ms | <8ms | Benchmark 10k timeline, visible window |
| Scroll latency | <16ms | <8ms | Benchmark PgUp/PgDn operations |
| Stream throughput | >1000 lines/sec | >5000 lines/sec | Benchmark PrintChunks |
| Block append | <1ms | <500µs | Benchmark Timeline.Append |
| Memory stable | No leaks | <100MB for 10k blocks | Profiling, stress test |

### 2.3 Functional Requirements

**FR-7.2.1:** Benchmark timeline with 10k blocks, measure GetVisibleBlocks() performance
**FR-7.2.2:** Benchmark streaming 100k lines via PrintChunks
**FR-7.2.3:** Benchmark scroll operations (PgUp, PgDn, g, G) on large timeline
**FR-7.2.4:** Profile memory usage, ensure no leaks over 1-hour stress test
**FR-7.2.5:** Optimize viewport calculation if render time exceeds 16ms
**FR-7.2.6:** Optimize block renderer allocations (string builders, buffer pools)
**FR-7.2.7:** Document performance characteristics in README or docs

---

## 3. Benchmarking Strategy

### 3.1 Timeline Benchmarks

#### 3.1.1 Large Timeline Rendering

```go
func BenchmarkTimelineGetVisibleBlocks_10k(b *testing.B)
func BenchmarkTimelineGetVisibleBlocks_100k(b *testing.B)
```

**Setup:**
- Timeline with 10k/100k blocks (realistic metadata mix)
- Viewport height: 40 rows (typical terminal)
- No filter applied

**Measure:**
- `GetVisibleBlocks()` call time
- Memory allocations

**Target:** <1ms for 10k blocks, <10ms for 100k blocks

---

#### 3.1.2 Scroll Operations

```go
func BenchmarkTimelineScrollDown_10k(b *testing.B)
func BenchmarkTimelineScrollPgDn_10k(b *testing.B)
func BenchmarkTimelineScrollToBottom_10k(b *testing.B)
```

**Setup:**
- Timeline with 10k blocks
- Viewport height: 40 rows
- Start at top, scroll to bottom

**Measure:**
- Scroll operation time + GetVisibleBlocks
- Allocation overhead

**Target:** <16ms per scroll operation (60fps)

---

#### 3.1.3 Filter Performance

```go
func BenchmarkTimelineFilter_10k(b *testing.B)
```

**Setup:**
- Timeline with 10k blocks (50% EXECUTE, 50% other types)
- Apply filter: `Types: [BlockTypeExecute]`

**Measure:**
- `SetFilter()` + `GetVisibleBlocks()` time
- Memory allocations

**Target:** <10ms for filter application

---

### 3.2 Streaming Benchmarks

#### 3.2.1 Line Throughput

```go
func BenchmarkPrinterPrintChunks_100k(b *testing.B)
```

**Setup:**
- 100k chunks (avg 50 bytes each, simulates LLM tokens)
- Channel buffer: 1000
- CoalesceDelay: 50ms (default)

**Measure:**
- Total throughput (chunks/sec, MB/sec)
- Latency (time to flush complete lines)
- Memory allocations

**Target:** >1000 lines/sec, <5% memory overhead

---

#### 3.2.2 Large Payload Streaming

```go
func BenchmarkPrinterPrintChunks_LargeChunks(b *testing.B)
```

**Setup:**
- 1000 chunks of 10KB each (simulates file dumps)
- Verify large chunk bypass (no buffering)

**Measure:**
- Throughput (MB/sec)
- Allocation overhead (should be minimal)

**Target:** >100 MB/sec, <1KB allocations per chunk

---

### 3.3 Block Rendering Benchmarks

#### 3.3.1 Individual Block Rendering

```go
func BenchmarkRendererRender_Execute(b *testing.B)
func BenchmarkRendererRender_Diff(b *testing.B)
func BenchmarkRendererRender_Code(b *testing.B)
```

**Setup:**
- Realistic block sizes:
  - EXECUTE: 500-line transcript
  - APPLY_PATCH: 100-line diff
  - READ: 200-line code snippet

**Measure:**
- Render() call time
- String builder allocations

**Target:** <1ms per block, <10 allocations

---

#### 3.3.2 Viewport Rendering (end-to-end)

```go
func BenchmarkRenderViewport_40Blocks(b *testing.B)
```

**Setup:**
- Render 40 visible blocks (typical viewport)
- Mix of block types
- Width: 120ch

**Measure:**
- Total render time (GetVisibleBlocks + Render all)
- Memory allocations

**Target:** <16ms for 40 blocks (60fps)

---

### 3.4 Stress Tests

#### 3.4.1 Memory Stability

```go
func TestMemoryStability_1Hour(t *testing.T)
```

**Setup:**
- Append 1 block/sec for 3600 iterations
- Concurrent scroll operations
- Periodic filter changes
- Run with GC metrics

**Measure:**
- Heap size over time
- Goroutine count
- GC pause times

**Target:** Heap size stable (<10% growth), no goroutine leaks

---

#### 3.4.2 Concurrent Operations

```go
func TestConcurrentOperations_Stress(t *testing.T)
```

**Setup:**
- 10 goroutines appending blocks
- 5 goroutines scrolling
- 5 goroutines filtering
- Run for 10 seconds

**Measure:**
- No panics, deadlocks, or race conditions
- Verify with `-race` detector

**Target:** All operations complete, zero race conditions

---

## 4. Optimization Strategies

### 4.1 Viewport Calculation

**Current:** Linear scan through blocks
**If needed:** Binary search for scroll position, cache viewport bounds

```go
// Optimization candidate
type viewportCache struct {
    scrollPos int
    startIdx  int
    endIdx    int
    version   int // Invalidate on timeline mutation
}
```

---

### 4.2 String Building

**Current:** `strings.Builder` per block render
**If needed:** Sync.Pool for builders, pre-allocate capacity

```go
var builderPool = sync.Pool{
    New: func() interface{} {
        b := &strings.Builder{}
        b.Grow(4096) // Pre-allocate 4KB
        return b
    },
}
```

---

### 4.3 ANSI Escape Sequences

**Current:** String concatenation
**If needed:** Pre-compute common sequences, use byte slices

```go
var (
    clearLine  = []byte("\x1b[2K")
    hideCursor = []byte("\x1b[?25l")
    // ...
)
```

---

### 4.4 Filter Indexing

**Current:** Linear scan through all blocks on filter
**If needed:** Maintain block type index (map[BlockType][]int)

---

## 5. Acceptance Criteria

### 5.1 Benchmarks

- [ ] All benchmark files created: `timeline_bench_test.go`, `renderer_bench_test.go`, `printer_bench_test.go`
- [ ] Benchmarks run successfully: `go test -bench=. ./internal/ui/...`
- [ ] Results documented: `docs/performance.md` with tables
- [ ] All SLO targets met or documented exceptions

### 5.2 Quality Gates

- [ ] All tests pass with `-race`
- [ ] `make lint` clean (zero errors)
- [ ] Complexity ≤15 per function
- [ ] No new dead code introduced
- [ ] Coverage maintained ≥85%

### 5.3 Performance Gates

- [ ] Viewport render <16ms for 10k blocks (or optimization applied)
- [ ] Scroll latency <16ms per operation
- [ ] Stream throughput >1000 lines/sec
- [ ] No memory leaks (1-hour stress test)
- [ ] No goroutine leaks
- [ ] GC pauses <10ms p99

### 5.4 Documentation

- [ ] Performance characteristics documented in `docs/performance.md`
- [ ] Benchmark results included (table format)
- [ ] Known limitations documented
- [ ] ROADMAP.md updated (Phase 7.2 marked complete)

---

## 6. Implementation Plan

### 6.1 Phase 1: Benchmark Infrastructure (1-2 hours)

1. Create `internal/ui/blocks/timeline_bench_test.go`
2. Create `internal/ui/blocks/renderer_bench_test.go`
3. Create `internal/ui/output/printer_bench_test.go`
4. Implement timeline benchmarks (3.1.1, 3.1.2, 3.1.3)
5. Implement streaming benchmarks (3.2.1, 3.2.2)
6. Implement rendering benchmarks (3.3.1, 3.3.2)

### 6.2 Phase 2: Baseline Measurement (30 minutes)

1. Run all benchmarks: `go test -bench=. -benchmem ./internal/ui/...`
2. Capture results to `benchmark-baseline.txt`
3. Identify performance gaps vs SLOs
4. Prioritize optimizations (if needed)

### 6.3 Phase 3: Optimization (conditional, 2-4 hours)

**Only if SLOs not met:**

1. Implement priority optimizations (viewport caching, string pooling)
2. Re-run benchmarks, measure improvement
3. Iterate until SLOs met or documented exception

### 6.4 Phase 4: Stress Testing (1-2 hours)

1. Implement stress tests (3.4.1, 3.4.2)
2. Run 1-hour memory stability test
3. Run concurrent operations test with `-race`
4. Profile with `go tool pprof` (CPU, memory)
5. Verify no leaks (goroutines, heap)

### 6.5 Phase 5: Documentation (1 hour)

1. Create `docs/performance.md`
2. Document benchmark results (markdown tables)
3. Document known limitations
4. Update ROADMAP.md (mark 7.2 complete)
5. Update AGENTS.md (if new patterns introduced)

---

## 7. Testing Strategy

### 7.1 Unit Tests

Existing unit tests (Phase 1-6) remain passing. No new unit tests required.

### 7.2 Benchmarks

See Section 3 for full benchmark matrix.

**Run command:**

```bash
go test -bench=. -benchmem -benchtime=3s ./internal/ui/blocks/...
go test -bench=. -benchmem -benchtime=3s ./internal/ui/output/...
```

**Example output format:**

```
BenchmarkTimelineGetVisibleBlocks_10k-8    5000    2.8ms/op    1024 B/op    15 allocs/op
BenchmarkTimelineScrollDown_10k-8          3000    3.2ms/op    512 B/op     8 allocs/op
BenchmarkPrinterPrintChunks_100k-8         100     15ms/op     8192 B/op    50 allocs/op
```

### 7.3 Profiling

```bash
# CPU profile
go test -bench=BenchmarkTimelineGetVisibleBlocks_10k -cpuprofile=cpu.prof ./internal/ui/blocks
go tool pprof cpu.prof

# Memory profile
go test -bench=BenchmarkTimelineGetVisibleBlocks_10k -memprofile=mem.prof ./internal/ui/blocks
go tool pprof mem.prof

# Trace
go test -bench=BenchmarkTimelineGetVisibleBlocks_10k -trace=trace.out ./internal/ui/blocks
go tool trace trace.out
```

---

## 8. Risks & Mitigations

### 8.1 Risk: Performance targets not met

**Likelihood:** Medium
**Impact:** High (blocks Phase 8 completion)

**Mitigation:**
- Start with baseline benchmarks to understand gap
- Prioritize optimizations by ROI (impact vs effort)
- Document acceptable exceptions (e.g., 100k blocks = 20ms, still usable)
- Defer advanced optimizations to post-launch if UX acceptable

### 8.2 Risk: Optimization introduces bugs

**Likelihood:** Medium
**Impact:** Medium

**Mitigation:**
- Maintain 100% test pass rate (150+ tests)
- Run with `-race` detector on all benchmarks
- Compare benchmark results before/after (no regressions)
- Code review optimizations carefully

### 8.3 Risk: Memory profiling reveals leaks

**Likelihood:** Low (Phase 7.1 validated concurrency)
**Impact:** High

**Mitigation:**
- Fix leaks immediately (blocking issue)
- Add regression tests (TestMemoryStability_1Hour)
- Re-run E2E tests to verify fix

---

## 9. Success Metrics

**Primary:**
- All performance SLOs met (Section 2.2)
- Zero memory/goroutine leaks
- Documentation complete

**Secondary:**
- Benchmarks runnable in CI (future)
- Performance regression tests in place
- Optimization patterns documented for future work

---

## 10. Open Questions

**Q1:** Should we include terminal emulator benchmarks (kitty, alacritty, iTerm2)?
**A1:** Defer to Phase 8.2 (manual QA). Focus on code-level performance.

**Q2:** What is acceptable p99 latency for interactive operations?
**A2:** 16ms (60fps) for scroll, 50ms for filter changes (perceived instant).

**Q3:** Should we benchmark on low-spec hardware (Raspberry Pi)?
**A3:** Out of scope. Target modern developer machines (2015+).

---

## 11. References

- **Spec:** [tui-new.md](../tui-implementation/tui-new.md) Section 12
- **Roadmap:** [ROADMAP.md](../tui-implementation/ROADMAP.md) Phase 7.2
- **Related Packages:**
  - [ui-blocks](../../docs/packages/ui-blocks.md)
  - [ui-output](../../docs/packages/ui-output.md)
- **Go Benchmarking:** https://pkg.go.dev/testing#hdr-Benchmarks
- **Profiling Guide:** https://go.dev/blog/pprof

---

## 12. Appendix: Benchmark Template

```go
// internal/ui/blocks/timeline_bench_test.go
package blocks_test

import (
    "testing"
    "github.com/dmytrogajewski/spin/internal/ui/blocks"
)

func BenchmarkTimelineGetVisibleBlocks_10k(b *testing.B) {
    // Setup
    tl := blocks.NewTimeline()
    tl.SetViewportHeight(40)

    for i := 0; i < 10000; i++ {
        block := blocks.NewBlock(blocks.BlockTypeExecute)
        block.Title = fmt.Sprintf("Block %d", i)
        tl.Append(block)
    }

    // Reset timer after setup
    b.ResetTimer()

    // Benchmark
    for i := 0; i < b.N; i++ {
        _ = tl.GetVisibleBlocks()
    }
}
```

---

**Status:** Ready for implementation
**Next Steps:** Create benchmark files, run baseline, analyze results
