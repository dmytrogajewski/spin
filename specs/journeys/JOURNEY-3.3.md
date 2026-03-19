# JOURNEY-3.3: Provider Cache

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 3.3 |
| Title | Wire Provider Cache for Model Capabilities |
| User Story | As a developer, model capabilities are cached to disk so startup doesn't repeat provider discovery. |
| Paper Section | Persistence layer — provider cache |
| Roadmap Item | JOURNEY-3.3 (16 functions) |

## Phases

### Phase 1: Discovery
- `cache.ProviderCache` fully implemented with Get/Put/Load/persist, stale-while-revalidate
- `cache.Entry.IsFresh()` checks TTL
- Full unit tests exist. Never wired.

### Phase 2: Integration
- Create `cache.NewProviderCache()` in `builder.go::Build()` with `~/.spin/cache/` dir
- Load cached data for current provider
- Use cached context length when config doesn't override
- Put provider capabilities after initial detection
- Store cache on Conversation for future use

## Implementation

### Files Modified
- `internal/conversation/builder.go` — Added `initProviderCache()`, creates `NewProviderCache()` with `WithTimeFunc(time.Now)`, loads cached data, populates context window from cache, puts provider capabilities after detection
