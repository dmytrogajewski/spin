# JOURNEY-extract-mathutil: Extract Cosine Similarity to internal/mathutil

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 2.1 — Extract Cosine Similarity to `internal/mathutil`
- Cluster: 4 (SPEC.md) | LIST.md findings 51, 58

## 1. Journey

When **a developer working on ACE vector operations** I want to **call a single, tested `mathutil.CosineSimilarity` function** so I can **avoid maintaining 4 identical copies of the same math and eliminate the buggy custom sqrt in the HNSW retriever**.

## 2. CJM

Four production packages independently implement `cosineSimilarity(a, b []float32) float64`:
- `internal/ace/playbook/search.go` (2 call-sites)
- `internal/ace/curator/deduplicator.go` (1 call-site)
- `internal/ace/retrieval/hnsw_retriever.go` (1 call-site, uses custom Newton's method sqrt)
- `internal/ace/refine/merge.go` (2 call-sites, method on MergeEngine)

Additionally, `internal/ace/embedding/ollama_embedder_test.go` has a buggy test variant that omits `math.Sqrt()`.

### Phase 1: Create mathutil package

**Actions:**
1. Create `internal/mathutil/vector.go` with `CosineSimilarity`, `DotProduct`, `Magnitude`.
2. Write comprehensive unit tests.

**Success Signal:** All edge cases covered: zero vectors, identical, orthogonal, unit, different lengths, empty.

### Phase 2: Migrate call-sites

**Actions:**
1. Replace 4 production `cosineSimilarity` functions with `mathutil.CosineSimilarity`.
2. Remove custom `sqrt` and `sqrtDivisor` from HNSW retriever.
3. Fix buggy test in ollama_embedder_test.go.

**Success Signal:** All ACE tests pass, no duplicate functions remain.

### Phase 3: Verification

**Actions:** `go vet`, `make lint`, `go test -race ./internal/mathutil/... ./internal/ace/...`

**Success Signal:** Zero issues, all tests green.

### North Star Summary

A single `mathutil.CosineSimilarity` function serves all vector similarity needs. Custom sqrt eliminated. Buggy test variant fixed.

## 3. Tests

### TC-01: identical vectors return 1.0
### TC-02: orthogonal vectors return 0.0
### TC-03: opposite vectors return -1.0
### TC-04: zero vector returns 0.0
### TC-05: different length vectors return 0.0
### TC-06: empty vectors return 0.0
### TC-07: unit vectors produce correct result
### TC-08: DotProduct correctness
### TC-09: Magnitude correctness

## Implementation

- Created: `internal/mathutil/vector.go` — `CosineSimilarity`, `DotProduct`, `Magnitude`
- Created: `internal/mathutil/vector_test.go` — comprehensive unit tests
- Modified: `internal/ace/playbook/search.go` — uses `mathutil.CosineSimilarity`
- Modified: `internal/ace/curator/deduplicator.go` — uses `mathutil.CosineSimilarity`
- Modified: `internal/ace/retrieval/hnsw_retriever.go` — uses `mathutil.CosineSimilarity`, removed custom `sqrt`
- Modified: `internal/ace/refine/merge.go` — uses `mathutil.CosineSimilarity`
- Modified: `internal/ace/embedding/ollama_embedder_test.go` — uses `mathutil.CosineSimilarity`
